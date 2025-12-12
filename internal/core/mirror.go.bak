package core

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/IBM/sarama"
	
	"message-mirror/internal/pkg/logger"
	"message-mirror/internal/pkg/ratelimiter"
	"message-mirror/internal/pkg/retry"
	"message-mirror/internal/pkg/deduplicator"
	"message-mirror/internal/pkg/optimization"
	"message-mirror/internal/pkg/metrics"
	"message-mirror/internal/plugins"
)

// MirrorMaker 消息镜像器
type MirrorMaker struct {
	config          *Config
	source          plugins.SourcePlugin
	producer        *MirrorProducer
	logger          *logger.Logger
	metrics         *metrics.Metrics
	retryManager    *retry.RetryManager
	deduplicator    *deduplicator.Deduplicator
	batchProcessor  *optimization.BatchProcessor  // 批处理器（如果启用）
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	stats           *Stats
	errorChan       chan error
	consumerLimiter *ratelimiter.RateLimiter  // 消费速率限制器（按消息）
	producerLimiter *ratelimiter.RateLimiter  // 生产速率限制器（按消息）
	bytesLimiter    *ratelimiter.RateLimiter  // 字节速率限制器（按字节，优先使用）
	mu              sync.RWMutex   // 保护配置更新
}

// Stats 统计信息
type Stats struct {
	mu                sync.RWMutex
	MessagesConsumed  int64
	MessagesProduced  int64
	BytesConsumed     int64
	BytesProduced     int64
	Errors            int64
	LastMessageTime   time.Time
	StartTime         time.Time
}

// NewMirrorMaker 创建新的MirrorMaker实例
func NewMirrorMaker(config *Config) (*MirrorMaker, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// 创建数据源插件
	source, err := plugins.CreatePlugin(config.Source.Type)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("创建数据源插件失败: %w", err)
	}

	// 初始化插件
	if err := source.Initialize(config.Source.Config); err != nil {
		cancel()
		return nil, fmt.Errorf("初始化数据源插件失败: %w", err)
	}

	// 创建生产者
	producer, err := NewMirrorProducer(config)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("创建生产者失败: %w", err)
	}

	stats := &Stats{
		StartTime: time.Now(),
	}

	// 创建日志管理器
	logConfig := &logger.LogConfig{
		FilePath:         config.Log.FilePath,
		StatsInterval:    config.Log.StatsInterval,
		RotateInterval:   config.Log.RotateInterval,
		MaxArchiveFiles:  config.Log.MaxArchiveFiles,
		AsyncBufferSize:  config.Log.AsyncBufferSize,
	}
	loggerInstance, err := logger.NewLogger(logConfig, ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("创建日志管理器失败: %w", err)
	}

	// 创建速率限制器
	var consumerLimiter, producerLimiter, bytesLimiter *ratelimiter.RateLimiter

	// 字节速率限制器（优先使用）
	if config.Mirror.BytesRateLimit > 0 {
		burstSize := config.Mirror.BytesBurstSize
		if burstSize <= 0 {
			burstSize = 10485760 // 默认10MB
		}
		bytesLimiter = ratelimiter.NewBytesRateLimiter(config.Mirror.BytesRateLimit, burstSize)
	} else {
		// 如果没有字节限制，使用消息限制
		if config.Mirror.ConsumerRateLimit > 0 {
			burstSize := config.Mirror.ConsumerBurstSize
			if burstSize <= 0 {
				burstSize = 100
			}
			consumerLimiter = ratelimiter.NewRateLimiter(config.Mirror.ConsumerRateLimit, burstSize, true)
		}

		if config.Mirror.ProducerRateLimit > 0 {
			burstSize := config.Mirror.ProducerBurstSize
			if burstSize <= 0 {
				burstSize = 100
			}
			producerLimiter = ratelimiter.NewRateLimiter(config.Mirror.ProducerRateLimit, burstSize, true)
		}
	}

	// 创建指标管理器
	metricsInstance := metrics.NewMetrics()
	metricsInstance.Register()

	// 创建重试管理器
	var retryManager *retry.RetryManager
	if config.Retry.Enabled {
		retryConfig := &retry.RetryConfigInternal{
			MaxRetries:      config.Retry.MaxRetries,
			InitialInterval: config.Retry.InitialInterval,
			MaxInterval:     config.Retry.MaxInterval,
			Multiplier:      config.Retry.Multiplier,
			Jitter:          config.Retry.Jitter,
		}
		if retryConfig.MaxRetries == 0 {
			retryConfig = retry.DefaultRetryConfig()
		}
		retryManager = retry.NewRetryManager(retryConfig)
	}

	// 创建去重器
	dedupConfig := &deduplicator.DedupConfig{
		Enabled:        config.Dedup.Enabled,
		Strategy:       config.Dedup.Strategy,
		TTL:            config.Dedup.TTL,
		MaxEntries:     config.Dedup.MaxEntries,
		CleanupInterval: config.Dedup.CleanupInterval,
	}
	deduplicatorInstance := deduplicator.NewDeduplicator(dedupConfig, ctx)

	mm := &MirrorMaker{
		config:          config,
		source:          source,
		producer:        producer,
		logger:          loggerInstance,
		metrics:         metricsInstance,
		retryManager:    retryManager,
		deduplicator:    deduplicatorInstance,
		batchProcessor:  nil, // 稍后初始化
		ctx:             ctx,
		cancel:          cancel,
		stats:           stats,
		errorChan:       make(chan error, 100),
		consumerLimiter: consumerLimiter,
		producerLimiter: producerLimiter,
		bytesLimiter:    bytesLimiter,
	}
	
	// 如果启用批处理，创建批处理器（需要mm实例）
	if config.Mirror.BatchEnabled && config.Mirror.BatchSize > 0 {
		batchSize := config.Mirror.BatchSize
		batchTimeout := config.Mirror.BatchTimeout
		if batchTimeout <= 0 {
			batchTimeout = 100 * time.Millisecond
		}
		
		// 批处理函数：批量发送消息
		processor := func(batch []*optimization.Message) error {
			for _, optMsg := range batch {
				// 将optimization.Message转换为plugins.Message
				msg := &plugins.Message{
					Key:       optMsg.Key,
					Value:     optMsg.Value,
					Headers:   optMsg.Headers,
					Timestamp: optMsg.Timestamp,
					Source:    optMsg.Source,
					Metadata:  optMsg.Metadata,
				}
				if err := mm.processMessage(msg); err != nil {
					return err
				}
			}
			return nil
		}
		
		mm.batchProcessor = optimization.NewBatchProcessor(batchSize, batchTimeout, processor)
	}
	
	return mm, nil
}

// Start 启动镜像服务
func (mm *MirrorMaker) Start() error {
	log.Println("启动MirrorMaker...")

	// 启动统计信息打印
	mm.wg.Add(1)
	go mm.printStats()

	// 启动错误处理
	mm.wg.Add(1)
	go mm.handleErrors()

	// 启动数据源插件
	if err := mm.source.Start(mm.ctx); err != nil {
		return fmt.Errorf("启动数据源插件失败: %w", err)
	}

	// 启动生产者
	if err := mm.producer.Start(mm.ctx); err != nil {
		return fmt.Errorf("启动生产者失败: %w", err)
	}

	// 启动批处理器（如果启用）
	if mm.batchProcessor != nil {
		mm.batchProcessor.Start(mm.ctx)
		log.Println("批处理器已启动")
	}

	// 启动worker goroutines
	for i := 0; i < mm.config.Mirror.WorkerCount; i++ {
		mm.wg.Add(1)
		go mm.worker(i)
	}

	log.Printf("MirrorMaker已启动，使用 %d 个worker", mm.config.Mirror.WorkerCount)
	return nil
}

// worker 工作协程，处理消息镜像
func (mm *MirrorMaker) worker(id int) {
	defer mm.wg.Done()

	log.Printf("Worker %d 启动", id)

	for {
		select {
		case <-mm.ctx.Done():
			log.Printf("Worker %d 停止", id)
			return

		case msg, ok := <-mm.source.Messages():
			if !ok {
				log.Printf("Worker %d: 消息通道已关闭", id)
				return
			}

			// 应用消费速率限制
			if err := mm.applyConsumerRateLimit(msg); err != nil {
				if err == mm.ctx.Err() {
					return
				}
				log.Printf("Worker %d: 应用消费速率限制失败: %v", id, err)
				continue
			}

			// 如果启用批处理，将消息添加到批处理器
			if mm.batchProcessor != nil {
				// 将plugin.Message转换为optimization.Message
				optMsg := &optimization.Message{
					Key:       msg.Key,
					Value:     msg.Value,
					Headers:   msg.Headers,
					Timestamp: msg.Timestamp,
					Source:    msg.Source,
					Metadata:  msg.Metadata,
				}
				if err := mm.batchProcessor.Add(optMsg); err != nil {
					log.Printf("Worker %d: 添加消息到批处理器失败: %v", id, err)
					mm.stats.mu.Lock()
					mm.stats.Errors++
					mm.stats.mu.Unlock()
				}
			} else {
				// 否则直接处理单条消息
				if err := mm.processMessage(msg); err != nil {
					log.Printf("Worker %d: 处理消息失败: %v", id, err)
					mm.stats.mu.Lock()
					mm.stats.Errors++
					mm.stats.mu.Unlock()
					mm.errorChan <- fmt.Errorf("worker %d: %w", id, err)
				}
			}
		}
	}
}

// applyConsumerRateLimit 应用消费速率限制
func (mm *MirrorMaker) applyConsumerRateLimit(msg *plugins.Message) error {
	// 优先使用字节速率限制器
	if mm.bytesLimiter != nil {
		return mm.bytesLimiter.WaitBytes(mm.ctx, len(msg.Value)+len(msg.Key))
	}

	// 使用消息速率限制器
	if mm.consumerLimiter != nil {
		return mm.consumerLimiter.Wait(mm.ctx)
	}

	return nil
}


// processMessage 处理单条消息
func (mm *MirrorMaker) processMessage(msg *plugins.Message) error {
	startTime := time.Now()

	// 检查消息是否重复（去重）
	if mm.deduplicator != nil {
		// 将plugin.Message转换为deduplicator.Message
		dedupMsg := &deduplicator.Message{
			Key:       msg.Key,
			Value:     msg.Value,
			Headers:   msg.Headers,
			Timestamp: msg.Timestamp,
			Source:    msg.Source,
			Metadata:  msg.Metadata,
		}
		isDuplicate, err := mm.deduplicator.IsDuplicate(dedupMsg)
		if err != nil {
			log.Printf("去重检查失败: %v", err)
			// 去重检查失败不影响消息处理，继续处理
		} else if isDuplicate {
			// 消息重复，跳过处理
			log.Printf("消息重复，已跳过: key=%s", string(msg.Key))
			// 仍然更新消费统计（因为确实消费了消息）
			mm.stats.mu.Lock()
			mm.stats.MessagesConsumed++
			mm.stats.BytesConsumed += int64(len(msg.Value))
			mm.stats.LastMessageTime = time.Now()
			mm.stats.mu.Unlock()

			if mm.metrics != nil {
				mm.metrics.RecordMessageConsumed(len(msg.Value) + len(msg.Key))
			}

			// 确认消息已处理（即使是重复的）
			if err := mm.source.Ack(msg); err != nil {
				log.Printf("确认重复消息失败: %v", err)
			}
			return nil
		}
	}

	// 更新统计信息
	mm.stats.mu.Lock()
	mm.stats.MessagesConsumed++
	mm.stats.BytesConsumed += int64(len(msg.Value))
	mm.stats.LastMessageTime = time.Now()
	mm.stats.mu.Unlock()

	// 记录消费指标
	if mm.metrics != nil {
		mm.metrics.RecordMessageConsumed(len(msg.Value) + len(msg.Key))
	}

	// 确定目标topic
	targetTopic := mm.getTargetTopic(msg)

	// 创建要发送的消息
	producerMessage := &sarama.ProducerMessage{
		Topic:     targetTopic,
		Key:       sarama.ByteEncoder(msg.Key),
		Value:     sarama.ByteEncoder(msg.Value),
		Headers:   mm.convertHeaders(msg.Headers),
		Timestamp: msg.Timestamp,
	}

	// 如果配置了保留分区且消息来自Kafka，则设置分区
	if mm.config.Mirror.PreservePartition && msg.Source == "kafka" {
		if partition, ok := msg.Metadata["partition"].(int32); ok {
			producerMessage.Partition = partition
		}
	}

	// 应用生产速率限制
	if err := mm.applyProducerRateLimit(msg); err != nil {
		if err == mm.ctx.Err() {
			return err
		}
		return fmt.Errorf("应用生产速率限制失败: %w", err)
	}

	// 发送消息到目标集群（带重试）
	var sendErr error
	if mm.retryManager != nil && mm.config.Retry.Enabled {
		sendErr = mm.retryManager.Retry(mm.ctx, func() error {
			return mm.producer.Send(producerMessage)
		})
	} else {
		sendErr = mm.producer.Send(producerMessage)
	}

	if sendErr != nil {
		// 记录失败指标
		if mm.metrics != nil {
			mm.metrics.RecordMessageFailed()
		}
		mm.stats.mu.Lock()
		mm.stats.Errors++
		mm.stats.mu.Unlock()
		return fmt.Errorf("发送消息失败 (topic=%s): %w", targetTopic, sendErr)
	}

	// 更新生产统计
	mm.stats.mu.Lock()
	mm.stats.MessagesProduced++
	mm.stats.BytesProduced += int64(len(msg.Value))
	mm.stats.mu.Unlock()

	// 记录生产指标和延迟
	if mm.metrics != nil {
		mm.metrics.RecordMessageProduced(len(msg.Value) + len(msg.Key))
		latency := time.Since(startTime)
		mm.metrics.RecordLatency(latency)
	}

	// 确认消息已处理
	if err := mm.source.Ack(msg); err != nil {
		log.Printf("确认消息失败: %v", err)
	}

	return nil
}

// applyProducerRateLimit 应用生产速率限制
func (mm *MirrorMaker) applyProducerRateLimit(msg *plugins.Message) error {
	// 优先使用字节速率限制器
	if mm.bytesLimiter != nil {
		return mm.bytesLimiter.WaitBytes(mm.ctx, len(msg.Value)+len(msg.Key))
	}

	// 使用消息速率限制器
	if mm.producerLimiter != nil {
		return mm.producerLimiter.Wait(mm.ctx)
	}

	return nil
}

// getTargetTopic 获取目标topic名称
func (mm *MirrorMaker) getTargetTopic(msg *plugins.Message) string {
	// 如果配置了目标topic，直接使用
	if mm.config.Target.Topic != "" {
		return mm.config.Target.Topic
	}

	// 如果消息来自Kafka，尝试从metadata获取topic
	if msg.Source == "kafka" {
		if topic, ok := msg.Metadata["topic"].(string); ok {
			return topic
		}
	}

	// 否则使用默认topic名称
	return "mirrored-messages"
}

// convertHeaders 转换消息头
func (mm *MirrorMaker) convertHeaders(headers map[string][]byte) []sarama.RecordHeader {
	if len(headers) == 0 {
		return nil
	}

	result := make([]sarama.RecordHeader, 0, len(headers))
	for k, v := range headers {
		result = append(result, sarama.RecordHeader{
			Key:   []byte(k),
			Value: v,
		})
	}

	return result
}

// Stop 停止镜像服务
func (mm *MirrorMaker) Stop() error {
	log.Println("正在停止MirrorMaker...")

	// 取消上下文
	mm.cancel()

	// 停止批处理器（如果启用）
	if mm.batchProcessor != nil {
		mm.batchProcessor.Stop()
		log.Println("批处理器已停止")
	}

	// 停止数据源插件
	if err := mm.source.Stop(); err != nil {
		log.Printf("停止数据源插件失败: %v", err)
	}

	// 停止生产者
	if err := mm.producer.Stop(); err != nil {
		log.Printf("停止生产者失败: %v", err)
	}

	// 等待所有goroutine完成
	mm.wg.Wait()

	// 停止日志管理器
	if mm.logger != nil {
		if err := mm.logger.Stop(); err != nil {
			log.Printf("停止日志管理器失败: %v", err)
		}
	}

	// 停止去重器
	if mm.deduplicator != nil {
		mm.deduplicator.Stop()
	}

	log.Println("MirrorMaker已停止")
	return nil
}

// GetMetrics 获取指标管理器
func (mm *MirrorMaker) GetMetrics() *metrics.Metrics {
	return mm.metrics
}

// OnConfigReload 实现ConfigReloadListener接口，处理配置热重载
func (mm *MirrorMaker) OnConfigReload(oldConfig, newConfig *Config) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	log.Println("开始配置热重载...")

	// 更新配置
	mm.config = newConfig

	// 更新速率限制器
	if newConfig.Mirror.BytesRateLimit > 0 {
		burstSize := newConfig.Mirror.BytesBurstSize
		if burstSize <= 0 {
			burstSize = 10485760 // 默认10MB
		}
		mm.bytesLimiter = ratelimiter.NewBytesRateLimiter(newConfig.Mirror.BytesRateLimit, burstSize)
		mm.consumerLimiter = nil
		mm.producerLimiter = nil
	} else {
		mm.bytesLimiter = nil
		if newConfig.Mirror.ConsumerRateLimit > 0 {
			burstSize := newConfig.Mirror.ConsumerBurstSize
			if burstSize <= 0 {
				burstSize = 100
			}
			mm.consumerLimiter = ratelimiter.NewRateLimiter(newConfig.Mirror.ConsumerRateLimit, burstSize, true)
		}
		if newConfig.Mirror.ProducerRateLimit > 0 {
			burstSize := newConfig.Mirror.ProducerBurstSize
			if burstSize <= 0 {
				burstSize = 100
			}
			mm.producerLimiter = ratelimiter.NewRateLimiter(newConfig.Mirror.ProducerRateLimit, burstSize, true)
		}
	}

	// 更新批处理器
	if newConfig.Mirror.BatchEnabled {
		if mm.batchProcessor == nil {
		// 创建新的批处理器
		processor := func(batch []*optimization.Message) error {
			for _, optMsg := range batch {
				// 将optimization.Message转换为plugins.Message
				msg := &plugins.Message{
					Key:       optMsg.Key,
					Value:     optMsg.Value,
					Headers:   optMsg.Headers,
					Timestamp: optMsg.Timestamp,
					Source:    optMsg.Source,
					Metadata:  optMsg.Metadata,
				}
				if err := mm.processMessage(msg); err != nil {
					log.Printf("批处理中处理消息失败: %v", err)
					// 继续处理其他消息
				}
			}
			return nil
		}
		mm.batchProcessor = optimization.NewBatchProcessor(newConfig.Mirror.BatchSize, newConfig.Mirror.BatchTimeout, processor)
			mm.batchProcessor.Start(mm.ctx)
		} else {
			// 更新现有批处理器的配置
			mm.batchProcessor.UpdateConfig(newConfig.Mirror.BatchSize, newConfig.Mirror.BatchTimeout)
		}
	} else {
		if mm.batchProcessor != nil {
			mm.batchProcessor.Stop()
			mm.batchProcessor = nil
		}
	}

	// 更新重试管理器
	if newConfig.Retry.Enabled {
		retryConfig := &retry.RetryConfigInternal{
			MaxRetries:      newConfig.Retry.MaxRetries,
			InitialInterval: newConfig.Retry.InitialInterval,
			MaxInterval:     newConfig.Retry.MaxInterval,
			Multiplier:      newConfig.Retry.Multiplier,
			Jitter:          newConfig.Retry.Jitter,
		}
		if retryConfig.MaxRetries == 0 {
			retryConfig = retry.DefaultRetryConfig()
		}
		mm.retryManager = retry.NewRetryManager(retryConfig)
	} else {
		mm.retryManager = nil
	}

	// 更新去重器配置
	if mm.deduplicator != nil {
		dedupConfig := &deduplicator.DedupConfig{
			Enabled:        newConfig.Dedup.Enabled,
			Strategy:       newConfig.Dedup.Strategy,
			TTL:            newConfig.Dedup.TTL,
			MaxEntries:     newConfig.Dedup.MaxEntries,
			CleanupInterval: newConfig.Dedup.CleanupInterval,
		}
		mm.deduplicator.UpdateConfig(dedupConfig)
	}

	// 更新生产者配置（需要重启生产者）
	if err := mm.producer.UpdateConfig(newConfig); err != nil {
		log.Printf("更新生产者配置失败: %v", err)
		// 不返回错误，因为其他配置已更新
	}

	log.Println("配置热重载完成")
	return nil
}

// printStats 定期打印统计信息
func (mm *MirrorMaker) printStats() {
	defer mm.wg.Done()

	interval := mm.config.Log.StatsInterval
	if interval <= 0 {
		interval = 10 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-mm.ctx.Done():
			return
		case <-ticker.C:
			mm.stats.mu.RLock()
			stats := *mm.stats
			mm.stats.mu.RUnlock()

			uptime := time.Since(stats.StartTime)
			var msgRate, bytesRate float64
			if uptime.Seconds() > 0 {
				msgRate = float64(stats.MessagesConsumed) / uptime.Seconds()
				bytesRate = float64(stats.BytesConsumed) / uptime.Seconds()
			}

			statsMsg := fmt.Sprintf("统计信息 - 运行时间: %v, 消费消息: %d (%.2f msg/s), "+
				"生产消息: %d, 错误: %d, 消费速率: %.2f KB/s",
				uptime.Round(time.Second),
				stats.MessagesConsumed, msgRate,
				stats.MessagesProduced,
				stats.Errors,
				bytesRate/1024)

			// 同时输出到标准输出和日志文件
			log.Println(statsMsg)
			if mm.logger != nil {
				mm.logger.Println(statsMsg)
			}
		}
	}
}

// handleErrors 处理错误
func (mm *MirrorMaker) handleErrors() {
	defer mm.wg.Done()

	for {
		select {
		case <-mm.ctx.Done():
			return
		case err := <-mm.errorChan:
			if err != nil {
				errMsg := fmt.Sprintf("错误: %v", err)
				log.Println(errMsg)
				if mm.logger != nil {
					mm.logger.Println(errMsg)
				}
			}
		}
	}
}

// GetStats 获取统计信息
func (mm *MirrorMaker) GetStats() Stats {
	mm.stats.mu.RLock()
	defer mm.stats.mu.RUnlock()
	return *mm.stats
}
