package deduplicator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"
)

// DedupConfig 去重配置（临时定义，后续会从core包导入）
type DedupConfig struct {
	Enabled        bool
	Strategy       string
	TTL            time.Duration
	MaxEntries     int64
	CleanupInterval time.Duration
}

// Message 统一的消息格式（临时定义，后续会从core包导入）
// 注意：这个类型需要与plugin.go中的Message类型兼容
type Message struct {
	Key       []byte
	Value     []byte
	Headers   map[string][]byte
	Timestamp time.Time
	Source    string
	Metadata  map[string]interface{}
}

// DefaultDedupConfig 默认去重配置
func DefaultDedupConfig() *DedupConfig {
	return &DedupConfig{
		Enabled:        false,
		Strategy:       "key_value",
		TTL:            24 * time.Hour,
		MaxEntries:     1000000, // 100万条记录
		CleanupInterval: 1 * time.Hour,
	}
}

// DedupEntry 去重记录
type DedupEntry struct {
	Hash      string
	Timestamp time.Time
}

// Deduplicator 消息去重器
type Deduplicator struct {
	config     *DedupConfig
	entries    map[string]*DedupEntry
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	totalDedup int64 // 去重的消息总数
}

// NewDeduplicator 创建新的去重器
func NewDeduplicator(config *DedupConfig, ctx context.Context) *Deduplicator {
	if config == nil {
		config = DefaultDedupConfig()
	}

	dedupCtx, cancel := context.WithCancel(ctx)

	deduplicator := &Deduplicator{
		config:  config,
		entries: make(map[string]*DedupEntry),
		ctx:     dedupCtx,
		cancel:  cancel,
	}

	// 如果启用去重，启动清理goroutine
	if config.Enabled {
		deduplicator.wg.Add(1)
		go deduplicator.cleanupWorker()
	}

	return deduplicator
}

// IsDuplicate 检查消息是否重复
// 注意：这里接受interface{}，然后转换为内部Message类型
// 后续会统一Message类型定义
func (d *Deduplicator) IsDuplicate(msg interface{}) (bool, error) {
	if !d.config.Enabled {
		return false, nil
	}

	// 类型断言转换为内部Message类型
	m, ok := msg.(*Message)
	if !ok {
		// 如果不是内部Message类型，尝试从外部Message转换
		// 这里需要根据实际Message结构进行转换
		// 暂时返回错误，后续会统一Message类型
		return false, fmt.Errorf("不支持的消息类型: %T，期望 *deduplicator.Message", msg)
	}

	// 生成去重标识
	hash, err := d.generateHash(m)
	if err != nil {
		return false, fmt.Errorf("生成去重hash失败: %w", err)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// 检查是否已存在
	if entry, exists := d.entries[hash]; exists {
		// 检查是否过期
		if time.Since(entry.Timestamp) < d.config.TTL {
			d.totalDedup++
			return true, nil
		}
		// 已过期，删除旧记录
		delete(d.entries, hash)
	}

	// 检查是否超过最大记录数
	if int64(len(d.entries)) >= d.config.MaxEntries {
		// 清理最旧的记录
		d.cleanupOldest()
	}

	// 添加新记录
	d.entries[hash] = &DedupEntry{
		Hash:      hash,
		Timestamp: time.Now(),
	}

	return false, nil
}

// generateHash 生成消息的去重hash
func (d *Deduplicator) generateHash(msg *Message) (string, error) {
	var data []byte

	switch d.config.Strategy {
	case "key":
		// 仅使用Key
		data = msg.Key
	case "value":
		// 仅使用Value
		data = msg.Value
	case "key_value":
		// 使用Key和Value的组合
		data = append(msg.Key, msg.Value...)
	case "hash":
		// 使用消息的完整内容（包括headers）
		data = append(msg.Key, msg.Value...)
		for k, v := range msg.Headers {
			data = append(data, []byte(k)...)
			data = append(data, v...)
		}
	default:
		// 默认使用key_value策略
		data = append(msg.Key, msg.Value...)
	}

	// 生成SHA256 hash
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// cleanupOldest 清理最旧的记录（保留一半）
func (d *Deduplicator) cleanupOldest() {
	// 按时间戳排序
	type entryTime struct {
		hash string
		time time.Time
	}

	entries := make([]entryTime, 0, len(d.entries))
	for hash, entry := range d.entries {
		entries = append(entries, entryTime{hash: hash, time: entry.Timestamp})
	}

	// 简单清理：删除一半最旧的记录
	toDelete := len(d.entries) / 2
	for i := 0; i < toDelete && i < len(entries); i++ {
		delete(d.entries, entries[i].hash)
	}
}

// cleanupWorker 定期清理过期记录的goroutine
func (d *Deduplicator) cleanupWorker() {
	defer d.wg.Done()

	ticker := time.NewTicker(d.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			d.cleanupExpired()
		}
	}
}

// cleanupExpired 清理过期的记录
func (d *Deduplicator) cleanupExpired() {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	expiredCount := 0

	for hash, entry := range d.entries {
		if now.Sub(entry.Timestamp) >= d.config.TTL {
			delete(d.entries, hash)
			expiredCount++
		}
	}

	if expiredCount > 0 {
		log.Printf("[去重器] 清理了 %d 条过期记录，当前记录数: %d", expiredCount, len(d.entries))
	}
}

// GetStats 获取去重统计信息
func (d *Deduplicator) GetStats() (int64, int) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.totalDedup, len(d.entries)
}

// Stop 停止去重器
func (d *Deduplicator) Stop() {
	d.cancel()
	d.wg.Wait()

	d.mu.Lock()
	defer d.mu.Unlock()

	log.Printf("[去重器] 已停止，总共去重 %d 条消息，当前记录数: %d", d.totalDedup, len(d.entries))
}

// UpdateConfig 更新去重器配置
func (d *Deduplicator) UpdateConfig(newConfig *DedupConfig) {
	d.mu.Lock()
	defer d.mu.Unlock()

	oldEnabled := d.config.Enabled
	d.config = newConfig

	// 如果从禁用变为启用，启动清理goroutine
	if !oldEnabled && newConfig.Enabled {
		d.wg.Add(1)
		go d.cleanupWorker()
	}

	// 如果从启用变为禁用，停止清理goroutine
	if oldEnabled && !newConfig.Enabled {
		d.cancel()
		d.wg.Wait()
		// 重新创建context
		d.ctx, d.cancel = context.WithCancel(context.Background())
	}
}

