package core

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/IBM/sarama"
)

// BenchmarkEndToEndThroughput 端到端吞吐量基准测试
// 测试每秒可以处理多少条消息
func BenchmarkEndToEndThroughput(b *testing.B) {
	if testing.Short() {
		b.Skip("跳过基准测试")
	}

	ctx := context.Background()

	// 1. 启动Kafka容器
	kafkaContainer, brokers, err := startKafkaContainer(ctx)
	if err != nil {
		b.Fatalf("启动Kafka容器失败: %v", err)
	}
	defer kafkaContainer.Terminate(ctx)

	time.Sleep(5 * time.Second)

	// 2. 创建topics
	sourceTopic := "benchmark-source"
	targetTopic := "benchmark-target"

	createTopic(brokers, sourceTopic)
	createTopic(brokers, targetTopic)

	// 3. 创建配置
	config := &Config{
		Source: SourceConfig{
			Type: "kafka",
			Config: map[string]interface{}{
				"brokers":           []interface{}{brokers[0]},
				"topic":             sourceTopic,
				"group_id":          "benchmark-group",
				"auto_offset_reset": "earliest",
			},
		},
		Target: TargetConfig{
			Brokers: brokers,
			Topic:   targetTopic,
		},
		Mirror: MirrorConfig{
			Enabled:      true,
			WorkerCount:  4,
			BatchEnabled: true,
			BatchSize:    100,
			BatchTimeout: 100 * time.Millisecond,
		},
		Producer: ProducerConfig{
			RequiredAcks:    1,
			CompressionType: "snappy",
		},
	}

	// 4. 启动MirrorMaker
	mm, err := NewMirrorMaker(config)
	if err != nil {
		b.Fatalf("创建MirrorMaker失败: %v", err)
	}
	if err := mm.Start(); err != nil {
		b.Fatalf("启动MirrorMaker失败: %v", err)
	}
	defer mm.Stop()

	time.Sleep(2 * time.Second)

	// 5. 创建生产者
	producer, err := createProducer(brokers)
	if err != nil {
		b.Fatalf("创建生产者失败: %v", err)
	}
	defer producer.Close()

	// 准备测试消息
	messageSize := 1024 // 1KB
	testMessage := make([]byte, messageSize)
	for i := range testMessage {
		testMessage[i] = byte('A' + (i % 26))
	}

	b.ResetTimer()
	b.SetBytes(int64(messageSize))

	// 6. 基准测试
	for i := 0; i < b.N; i++ {
		_, _, err := producer.SendMessage(&sarama.ProducerMessage{
			Topic: sourceTopic,
			Key:   sarama.StringEncoder(fmt.Sprintf("key-%d", i)),
			Value: sarama.ByteEncoder(testMessage),
		})
		if err != nil {
			b.Errorf("发送消息失败: %v", err)
		}
	}

	b.StopTimer()

	// 等待所有消息处理完成
	time.Sleep(5 * time.Second)

	// 报告统计
	stats := mm.GetStats()
	b.Logf("统计信息: consumed=%d, produced=%d, errors=%d",
		stats.MessagesConsumed, stats.MessagesProduced, stats.Errors)

	// 计算吞吐量
	throughput := float64(stats.MessagesConsumed) / time.Since(stats.StartTime).Seconds()
	b.Logf("吞吐量: %.2f msg/s", throughput)
}

// BenchmarkConfigReload 配置重载基准测试
// 测试配置重载的时间开销
func BenchmarkConfigReload(b *testing.B) {
	ctx := context.Background()

	// 1. 启动Kafka容器
	kafkaContainer, brokers, err := startKafkaContainer(ctx)
	if err != nil {
		b.Fatalf("启动Kafka容器失败: %v", err)
	}
	defer kafkaContainer.Terminate(ctx)

	time.Sleep(5 * time.Second)

	// 2. 创建topics
	sourceTopic := "reload-bench-source"
	targetTopic := "reload-bench-target"

	createTopic(brokers, sourceTopic)
	createTopic(brokers, targetTopic)

	// 3. 创建初始配置
	config := &Config{
		Source: SourceConfig{
			Type: "kafka",
			Config: map[string]interface{}{
				"brokers":           []interface{}{brokers[0]},
				"topic":             sourceTopic,
				"group_id":          "reload-bench-group",
				"auto_offset_reset": "earliest",
			},
		},
		Target: TargetConfig{
			Brokers: brokers,
			Topic:   targetTopic,
		},
		Mirror: MirrorConfig{
			Enabled:           true,
			WorkerCount:       2,
			ConsumerRateLimit: 100,
		},
		Producer: ProducerConfig{
			RequiredAcks:    1,
			CompressionType: "none",
		},
	}

	// 4. 启动MirrorMaker
	mm, err := NewMirrorMaker(config)
	if err != nil {
		b.Fatalf("创建MirrorMaker失败: %v", err)
	}
	if err := mm.Start(); err != nil {
		b.Fatalf("启动MirrorMaker失败: %v", err)
	}
	defer mm.Stop()

	time.Sleep(2 * time.Second)

	// 5. 创建新配置（用于重载）
	newConfig := &Config{
		Source: SourceConfig{
			Type: "kafka",
			Config: map[string]interface{}{
				"brokers":           []interface{}{brokers[0]},
				"topic":             sourceTopic,
				"group_id":          "reload-bench-group",
				"auto_offset_reset": "earliest",
			},
		},
		Target: TargetConfig{
			Brokers: brokers,
			Topic:   targetTopic,
		},
		Mirror: MirrorConfig{
			Enabled:           true,
			WorkerCount:       4,
			ConsumerRateLimit: 200,
		},
		Producer: ProducerConfig{
			RequiredAcks:    1,
			CompressionType: "snappy",
		},
	}

	b.ResetTimer()

	// 6. 基准测试配置重载
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			err = mm.OnConfigReload(config, newConfig)
		} else {
			err = mm.OnConfigReload(newConfig, config)
		}
		if err != nil {
			b.Errorf("配置重载失败: %v", err)
		}
	}

	b.StopTimer()
}

// BenchmarkMessageProcessing 消息处理基准测试
// 测试单个消息的处理延迟
func BenchmarkMessageProcessing(b *testing.B) {
	if testing.Short() {
		b.Skip("跳过基准测试")
	}

	ctx := context.Background()

	// 1. 启动Kafka容器
	kafkaContainer, brokers, err := startKafkaContainer(ctx)
	if err != nil {
		b.Fatalf("启动Kafka容器失败: %v", err)
	}
	defer kafkaContainer.Terminate(ctx)

	time.Sleep(5 * time.Second)

	// 2. 创建topics
	sourceTopic := "process-bench-source"
	targetTopic := "process-bench-target"

	createTopic(brokers, sourceTopic)
	createTopic(brokers, targetTopic)

	// 3. 创建优化的配置
	config := &Config{
		Source: SourceConfig{
			Type: "kafka",
			Config: map[string]interface{}{
				"brokers":           []interface{}{brokers[0]},
				"topic":             sourceTopic,
				"group_id":          "process-bench-group",
				"auto_offset_reset": "earliest",
			},
		},
		Target: TargetConfig{
			Brokers: brokers,
			Topic:   targetTopic,
		},
		Mirror: MirrorConfig{
			Enabled:      true,
			WorkerCount:  8, // 更多workers
			BatchEnabled: true,
			BatchSize:    50,
			BatchTimeout: 50 * time.Millisecond,
		},
		Producer: ProducerConfig{
			RequiredAcks:    1,
			CompressionType: "lz4", // 更快的压缩
		},
		Retry: RetryConfig{
			Enabled: false, // 禁用重试以获得更纯粹的基准
		},
		Dedup: DedupConfig{
			Enabled: false, // 禁用去重以获得更纯粹的基准
		},
	}

	// 4. 启动MirrorMaker
	mm, err := NewMirrorMaker(config)
	if err != nil {
		b.Fatalf("创建MirrorMaker失败: %v", err)
	}
	if err := mm.Start(); err != nil {
		b.Fatalf("启动MirrorMaker失败: %v", err)
	}
	defer mm.Stop()

	time.Sleep(2 * time.Second)

	// 5. 创建生产者
	producer, err := createProducer(brokers)
	if err != nil {
		b.Fatalf("创建生产者失败: %v", err)
	}
	defer producer.Close()

	// 小消息测试
	smallMessage := []byte("small test message")

	b.ResetTimer()
	b.SetBytes(int64(len(smallMessage)))

	// 6. 基准测试
	for i := 0; i < b.N; i++ {
		_, _, err := producer.SendMessage(&sarama.ProducerMessage{
			Topic: sourceTopic,
			Key:   sarama.StringEncoder(fmt.Sprintf("key-%d", i)),
			Value: sarama.ByteEncoder(smallMessage),
		})
		if err != nil {
			b.Errorf("发送消息失败: %v", err)
		}
	}

	b.StopTimer()

	// 等待处理完成
	time.Sleep(3 * time.Second)

	stats := mm.GetStats()
	avgLatency := time.Since(stats.StartTime) / time.Duration(stats.MessagesConsumed)
	b.Logf("平均延迟: %v", avgLatency)
	b.Logf("统计: consumed=%d, produced=%d",
		stats.MessagesConsumed, stats.MessagesProduced)
}

// BenchmarkBatchProcessing 批处理基准测试
// 对比启用/禁用批处理的性能差异
func BenchmarkBatchProcessing(b *testing.B) {
	if testing.Short() {
		b.Skip("跳过基准测试")
	}

	testCases := []struct {
		name         string
		batchEnabled bool
		batchSize    int
		batchTimeout time.Duration
	}{
		{"NoBatch", false, 0, 0},
		{"Batch10", true, 10, 100 * time.Millisecond},
		{"Batch50", true, 50, 100 * time.Millisecond},
		{"Batch100", true, 100, 100 * time.Millisecond},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			ctx := context.Background()

			// 启动Kafka
			kafkaContainer, brokers, err := startKafkaContainer(ctx)
			if err != nil {
				b.Fatalf("启动Kafka失败: %v", err)
			}
			defer kafkaContainer.Terminate(ctx)

			time.Sleep(5 * time.Second)

			sourceTopic := fmt.Sprintf("batch-bench-%s-source", tc.name)
			targetTopic := fmt.Sprintf("batch-bench-%s-target", tc.name)

			createTopic(brokers, sourceTopic)
			createTopic(brokers, targetTopic)

			config := &Config{
				Source: SourceConfig{
					Type: "kafka",
					Config: map[string]interface{}{
						"brokers":           []interface{}{brokers[0]},
						"topic":             sourceTopic,
						"group_id":          fmt.Sprintf("batch-bench-%s", tc.name),
						"auto_offset_reset": "earliest",
					},
				},
				Target: TargetConfig{
					Brokers: brokers,
					Topic:   targetTopic,
				},
				Mirror: MirrorConfig{
					Enabled:      true,
					WorkerCount:  4,
					BatchEnabled: tc.batchEnabled,
					BatchSize:    tc.batchSize,
					BatchTimeout: tc.batchTimeout,
				},
				Producer: ProducerConfig{
					RequiredAcks:    1,
					CompressionType: "snappy",
				},
			}

			mm, err := NewMirrorMaker(config)
			if err != nil {
				b.Fatalf("创建MirrorMaker失败: %v", err)
			}
			if err := mm.Start(); err != nil {
				b.Fatalf("启动MirrorMaker失败: %v", err)
			}
			defer mm.Stop()

			time.Sleep(2 * time.Second)

			producer, err := createProducer(brokers)
			if err != nil {
				b.Fatalf("创建生产者失败: %v", err)
			}
			defer producer.Close()

			testMessage := []byte("batch benchmark test message")

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				producer.SendMessage(&sarama.ProducerMessage{
					Topic: sourceTopic,
					Key:   sarama.StringEncoder(fmt.Sprintf("key-%d", i)),
					Value: sarama.ByteEncoder(testMessage),
				})
			}

			b.StopTimer()
			time.Sleep(3 * time.Second)

			stats := mm.GetStats()
			throughput := float64(stats.MessagesConsumed) / time.Since(stats.StartTime).Seconds()
			b.Logf("吞吐量: %.2f msg/s", throughput)
		})
	}
}
