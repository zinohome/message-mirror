package optimization

import (
	"context"
	"log"
	"sync"
	"time"
)

// Message 统一的消息格式（临时定义，后续会从core包导入）
type Message struct {
	Key       []byte
	Value     []byte
	Headers   map[string][]byte
	Timestamp time.Time
	Source    string
	Metadata  map[string]interface{}
}

// BatchProcessor 批处理器，用于优化消息处理性能
type BatchProcessor struct {
	batchSize     int
	batchTimeout  time.Duration
	processor     func([]*Message) error
	messageChan   chan *Message
	batch         []*Message
	mu            sync.Mutex
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewBatchProcessor 创建新的批处理器
func NewBatchProcessor(batchSize int, batchTimeout time.Duration, processor func([]*Message) error) *BatchProcessor {
	return &BatchProcessor{
		batchSize:    batchSize,
		batchTimeout: batchTimeout,
		processor:    processor,
		messageChan:  make(chan *Message, batchSize*2),
		batch:        make([]*Message, 0, batchSize),
	}
}

// Start 启动批处理器
func (bp *BatchProcessor) Start(ctx context.Context) {
	bp.ctx, bp.cancel = context.WithCancel(ctx)
	bp.wg.Add(1)
	go bp.processBatch()
}

// Stop 停止批处理器
func (bp *BatchProcessor) Stop() {
	if bp.cancel != nil {
		bp.cancel()
	}
	if bp.messageChan != nil {
		close(bp.messageChan)
	}
	bp.wg.Wait()
}

// UpdateConfig 更新批处理器配置
func (bp *BatchProcessor) UpdateConfig(batchSize int, batchTimeout time.Duration) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.batchSize = batchSize
	bp.batchTimeout = batchTimeout
}

// Add 添加消息到批处理队列
func (bp *BatchProcessor) Add(msg *Message) error {
	select {
	case <-bp.ctx.Done():
		return bp.ctx.Err()
	case bp.messageChan <- msg:
		return nil
	}
}

// processBatch 处理批次消息
func (bp *BatchProcessor) processBatch() {
	defer bp.wg.Done()
	
	ticker := time.NewTicker(bp.batchTimeout)
	defer ticker.Stop()
	
	for {
		select {
		case <-bp.ctx.Done():
			// 处理剩余消息
			bp.flushBatch()
			return
			
		case msg, ok := <-bp.messageChan:
			if !ok {
				bp.flushBatch()
				return
			}
			
			bp.mu.Lock()
			bp.batch = append(bp.batch, msg)
			shouldFlush := len(bp.batch) >= bp.batchSize
			bp.mu.Unlock()
			
			if shouldFlush {
				bp.flushBatch()
				ticker.Reset(bp.batchTimeout)
			}
			
		case <-ticker.C:
			bp.flushBatch()
		}
	}
}

// flushBatch 刷新批次
func (bp *BatchProcessor) flushBatch() {
	bp.mu.Lock()
	if len(bp.batch) == 0 {
		bp.mu.Unlock()
		return
	}
	
	batch := make([]*Message, len(bp.batch))
	copy(batch, bp.batch)
	bp.batch = bp.batch[:0]
	bp.mu.Unlock()
	
	// 处理批次
	if bp.processor != nil {
		if err := bp.processor(batch); err != nil {
			log.Printf("批处理失败: %v", err)
		}
	}
}

// ConnectionPool 连接池（用于Kafka生产者优化）
type ConnectionPool struct {
	poolSize int
	mu       sync.RWMutex
	stats    *PoolStats
}

// PoolStats 连接池统计信息
type PoolStats struct {
	ActiveConnections int64
	TotalConnections  int64
	ConnectionErrors  int64
}

// NewConnectionPool 创建新的连接池
func NewConnectionPool(poolSize int) *ConnectionPool {
	return &ConnectionPool{
		poolSize: poolSize,
		stats:    &PoolStats{},
	}
}

// GetStats 获取连接池统计信息
func (cp *ConnectionPool) GetStats() PoolStats {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return *cp.stats
}

// MessageCache 消息缓存（用于去重优化）
type MessageCache struct {
	cache     map[string]*CacheEntry
	mu        sync.RWMutex
	maxSize   int
	ttl       time.Duration
	cleanupCh chan struct{}
}

// CacheEntry 缓存条目
type CacheEntry struct {
	Hash      string
	Timestamp time.Time
}

// NewMessageCache 创建新的消息缓存
func NewMessageCache(maxSize int, ttl time.Duration) *MessageCache {
	cache := &MessageCache{
		cache:     make(map[string]*CacheEntry),
		maxSize:   maxSize,
		ttl:       ttl,
		cleanupCh: make(chan struct{}),
	}
	
	// 启动清理goroutine
	go cache.cleanup()
	
	return cache
}

// Get 获取缓存条目
func (mc *MessageCache) Get(key string) (bool, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	entry, exists := mc.cache[key]
	if !exists {
		return false, false
	}
	
	// 检查是否过期
	if time.Since(entry.Timestamp) >= mc.ttl {
		return false, true // 存在但已过期
	}
	
	return true, true // 存在且有效
}

// Set 设置缓存条目
func (mc *MessageCache) Set(key string, hash string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	// 如果超过最大大小，清理最旧的条目
	if len(mc.cache) >= mc.maxSize {
		mc.evictOldest()
	}
	
	mc.cache[key] = &CacheEntry{
		Hash:      hash,
		Timestamp: time.Now(),
	}
}

// evictOldest 清理最旧的条目
func (mc *MessageCache) evictOldest() {
	oldestKey := ""
	oldestTime := time.Now()
	
	for k, v := range mc.cache {
		if v.Timestamp.Before(oldestTime) {
			oldestTime = v.Timestamp
			oldestKey = k
		}
	}
	
	if oldestKey != "" {
		delete(mc.cache, oldestKey)
	}
}

// cleanup 定期清理过期条目
func (mc *MessageCache) cleanup() {
	ticker := time.NewTicker(mc.ttl / 2)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			mc.cleanupExpired()
		case <-mc.cleanupCh:
			return
		}
	}
}

// cleanupExpired 清理过期条目
func (mc *MessageCache) cleanupExpired() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	now := time.Now()
	for k, v := range mc.cache {
		if now.Sub(v.Timestamp) >= mc.ttl {
			delete(mc.cache, k)
		}
	}
}

// Stop 停止缓存
func (mc *MessageCache) Stop() {
	close(mc.cleanupCh)
}

