package deduplicator

import (
	"context"
	"testing"
	"time"
)

// TestDeduplicator_KeyStrategy 测试key策略去重
func TestDeduplicator_KeyStrategy(t *testing.T) {
	config := &DedupConfig{
		Enabled:         true,
		Strategy:        "key",
		TTL:             1 * time.Second,
		MaxEntries:      100,
		CleanupInterval: 100 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dedup := NewDeduplicator(config, ctx)
	defer dedup.Stop()

	msg1 := &Message{
		Key:   []byte("key1"),
		Value: []byte("value1"),
	}
	msg2 := &Message{
		Key:   []byte("key1"),
		Value: []byte("value2"), // 不同value，但key相同
	}
	msg3 := &Message{
		Key:   []byte("key2"),
		Value: []byte("value1"),
	}

	// 第一条消息不是重复
	isDup, _ := dedup.IsDuplicate(msg1)
	if isDup {
		t.Error("首次消息不应被标记为重复")
	}

	// 相同key的消息是重复
	isDup, _ = dedup.IsDuplicate(msg2)
	if !isDup {
		t.Error("相同key的消息应被标记为重复")
	}

	// 不同key的消息不是重复
	isDup, _ = dedup.IsDuplicate(msg3)
	if isDup {
		t.Error("不同key的消息不应被标记为重复")
	}
}

// TestDeduplicator_ValueStrategy 测试value策略去重
func TestDeduplicator_ValueStrategy(t *testing.T) {
	config := &DedupConfig{
		Enabled:         true,
		Strategy:        "value",
		TTL:             1 * time.Second,
		MaxEntries:      100,
		CleanupInterval: 100 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dedup := NewDeduplicator(config, ctx)
	defer dedup.Stop()

	msg1 := &Message{
		Key:   []byte("key1"),
		Value: []byte("value1"),
	}
	msg2 := &Message{
		Key:   []byte("key2"),
		Value: []byte("value1"), // 不同key，但value相同
	}

	isDup, _ := dedup.IsDuplicate(msg1)
	if isDup {
		t.Error("首次消息不应被重复")
	}

	isDup, _ = dedup.IsDuplicate(msg2)
	if !isDup {
		t.Error("相同value的消息应被标记为重复")
	}
}

// TestDeduplicator_KeyValueStrategy 测试key+value策略
func TestDeduplicator_KeyValueStrategy(t *testing.T) {
	config := &DedupConfig{
		Enabled:         true,
		Strategy:        "key_value",
		TTL:             1 * time.Second,
		MaxEntries:      100,
		CleanupInterval: 100 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dedup := NewDeduplicator(config, ctx)
	defer dedup.Stop()

	msg1 := &Message{
		Key:   []byte("key1"),
		Value: []byte("value1"),
	}
	msg2 := &Message{
		Key:   []byte("key1"),
		Value: []byte("value2"), // 相同key，不同value
	}
	msg3 := &Message{
		Key:   []byte("key1"),
		Value: []byte("value1"), // 完全相同
	}

	isDup, _ := dedup.IsDuplicate(msg1)
	if isDup {
		t.Error("首次消息不应重复")
	}

	isDup, _ = dedup.IsDuplicate(msg2)
	if isDup {
		t.Error("key相同但value不同，不应重复")
	}

	isDup, _ = dedup.IsDuplicate(msg3)
	if !isDup {
		t.Error("完全相同的消息应被标记为重复")
	}
}

// TestDeduplicator_HashStrategy 测试hash策略
func TestDeduplicator_HashStrategy(t *testing.T) {
	config := &DedupConfig{
		Enabled:         true,
		Strategy:        "hash",
		TTL:             1 * time.Second,
		MaxEntries:      100,
		CleanupInterval: 100 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dedup := NewDeduplicator(config, ctx)
	defer dedup.Stop()

	msg1 := &Message{
		Key:   []byte("key1"),
		Value: []byte("value1"),
		Headers: map[string][]byte{
			"header1": []byte("h1"),
		},
	}
	msg2 := &Message{
		Key:   []byte("key1"),
		Value: []byte("value1"),
		Headers: map[string][]byte{
			"header1": []byte("h1"),
		},
	}

	isDup, _ := dedup.IsDuplicate(msg1)
	if isDup {
		t.Error("首次消息不应重复")
	}

	isDup, _ = dedup.IsDuplicate(msg2)
	if !isDup {
		t.Error("相同内容的消息应被标记为重复")
	}
}

// TestDeduplicator_TTLExpiration 测试TTL过期
func TestDeduplicator_TTLExpiration(t *testing.T) {
	config := &DedupConfig{
		Enabled:         true,
		Strategy:        "key",
		TTL:             50 * time.Millisecond,
		MaxEntries:      100,
		CleanupInterval: 20 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dedup := NewDeduplicator(config, ctx)
	defer dedup.Stop()

	msg := &Message{
		Key:   []byte("key1"),
		Value: []byte("value1"),
	}

	// 第一次不重复
	isDup, _ := dedup.IsDuplicate(msg)
	if isDup {
		t.Error("首次消息不应重复")
	}

	// 立即检查，应该重复
	isDup, _ = dedup.IsDuplicate(msg)
	if !isDup {
		t.Error("立即检查应该重复")
	}

	// 等待TTL过期
	time.Sleep(100 * time.Millisecond)

	// TTL过期后，不应再重复
	isDup, _ = dedup.IsDuplicate(msg)
	if isDup {
		t.Error("TTL过期后不应重复")
	}
}

// TestDeduplicator_MaxEntriesLimit 测试最大条目限制不会崩溃
func TestDeduplicator_MaxEntriesLimit(t *testing.T) {
	config := &DedupConfig{
		Enabled:         true,
		Strategy:        "key",
		TTL:             10 * time.Second,
		MaxEntries:      5,
		CleanupInterval: 100 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dedup := NewDeduplicator(config, ctx)
	defer dedup.Stop()

	// 插入超过maxEntries的消息
	for i := 0; i < 10; i++ {
		msg := &Message{
			Key:   []byte(string(rune('a' + i))),
			Value: []byte("value"),
		}
		dedup.IsDuplicate(msg)
	}

	// 验证不会崩溃，但具体行为取决于实现
	// 这里主要测试不会panic
	msg := &Message{
		Key:   []byte("z"),
		Value: []byte("value"),
	}
	_, err := dedup.IsDuplicate(msg)
	if err != nil {
		t.Errorf("添加消息失败: %v", err)
	}
}

// TestDeduplicator_DisabledBehavior 测试禁用去重行为
func TestDeduplicator_DisabledBehavior(t *testing.T) {
	config := &DedupConfig{
		Enabled:         false,
		Strategy:        "key",
		TTL:             1 * time.Second,
		MaxEntries:      100,
		CleanupInterval: 100 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dedup := NewDeduplicator(config, ctx)
	defer dedup.Stop()

	msg := &Message{
		Key:   []byte("key1"),
		Value: []byte("value1"),
	}

	// 禁用时，所有消息都不应重复
	isDup, _ := dedup.IsDuplicate(msg)
	if isDup {
		t.Error("禁用去重时不应标记为重复")
	}

	isDup, _ = dedup.IsDuplicate(msg)
	if isDup {
		t.Error("禁用去重时不应标记为重复")
	}
}

// TestDeduplicator_UpdateConfigRuntime 测试运行时配置更新
func TestDeduplicator_UpdateConfigRuntime(t *testing.T) {
	config := &DedupConfig{
		Enabled:         true,
		Strategy:        "key",
		TTL:             1 * time.Second,
		MaxEntries:      100,
		CleanupInterval: 100 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dedup := NewDeduplicator(config, ctx)
	defer dedup.Stop()

	// 更新配置
	newConfig := &DedupConfig{
		Enabled:         false,
		Strategy:        "value",
		TTL:             2 * time.Second,
		MaxEntries:      200,
		CleanupInterval: 200 * time.Millisecond,
	}

	dedup.UpdateConfig(newConfig)

	// 验证更新后禁用去重
	msg := &Message{
		Key:   []byte("key1"),
		Value: []byte("value1"),
	}

	isDup, _ := dedup.IsDuplicate(msg)
	if isDup {
		t.Error("更新配置为禁用后不应重复")
	}
}
