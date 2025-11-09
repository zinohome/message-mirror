package deduplicator

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestDeduplicator_Cleanup 测试过期记录清理
func TestDeduplicator_Cleanup(t *testing.T) {
	config := &DedupConfig{
		Enabled:        true,
		Strategy:       "key_value",
		TTL:            100 * time.Millisecond, // 短TTL用于测试
		MaxEntries:     1000,
		CleanupInterval: 50 * time.Millisecond, // 短清理间隔
	}

	ctx := context.Background()
	deduplicator := NewDeduplicator(config, ctx)
	defer deduplicator.Stop()

	// 创建消息
	msg := &Message{
		Key:   []byte("test-key"),
		Value: []byte("test-value"),
	}

	// 第一次检查（不是重复）
	isDup, err := deduplicator.IsDuplicate(msg)
	if err != nil {
		t.Fatalf("去重检查失败: %v", err)
	}
	if isDup {
		t.Error("第一条消息不应该是重复的")
	}

	// 等待TTL过期
	time.Sleep(150 * time.Millisecond)

	// 再次检查（应该不是重复，因为记录已过期）
	isDup, err = deduplicator.IsDuplicate(msg)
	if err != nil {
		t.Fatalf("去重检查失败: %v", err)
	}
	if isDup {
		t.Error("过期后的消息不应该被认为是重复的")
	}
}

// TestDeduplicator_MaxEntries 测试最大条目限制
func TestDeduplicator_MaxEntries(t *testing.T) {
	config := &DedupConfig{
		Enabled:        true,
		Strategy:       "key_value",
		TTL:            1 * time.Hour,
		MaxEntries:     5, // 小值用于测试
		CleanupInterval: 1 * time.Hour,
	}

	ctx := context.Background()
	deduplicator := NewDeduplicator(config, ctx)
	defer deduplicator.Stop()

	// 添加多条消息直到达到MaxEntries
	for i := 0; i < 10; i++ {
		msg := &Message{
			Key:   []byte("test-key"),
			Value: []byte{byte(i)}, // 不同的值
		}

		_, err := deduplicator.IsDuplicate(msg)
		if err != nil {
			t.Fatalf("去重检查失败: %v", err)
		}
	}

	// 验证条目数量不超过MaxEntries
	_, entries := deduplicator.GetStats()
	if int64(entries) > config.MaxEntries {
		t.Errorf("条目数量不应该超过MaxEntries，实际%d，最大%d", entries, config.MaxEntries)
	}
}

// TestDeduplicator_ConcurrentAccess 测试并发访问
func TestDeduplicator_ConcurrentAccess(t *testing.T) {
	config := &DedupConfig{
		Enabled:        true,
		Strategy:       "key_value",
		TTL:            1 * time.Hour,
		MaxEntries:     1000,
		CleanupInterval: 1 * time.Hour,
	}

	ctx := context.Background()
	deduplicator := NewDeduplicator(config, ctx)
	defer deduplicator.Stop()

	// 并发访问
	var wg sync.WaitGroup
	concurrency := 20
	messagesPerGoroutine := 10

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				msg := &Message{
					Key:   []byte{byte(id)},
					Value: []byte{byte(j)},
				}

				_, err := deduplicator.IsDuplicate(msg)
				if err != nil {
					t.Errorf("去重检查失败: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// 验证没有panic或数据竞争
	totalDedup, entries := deduplicator.GetStats()
	t.Logf("并发访问后统计: 去重%d条，条目%d", totalDedup, entries)
}

// TestDeduplicator_Strategy_Hash 测试hash策略
func TestDeduplicator_Strategy_Hash(t *testing.T) {
	config := &DedupConfig{
		Enabled:        true,
		Strategy:       "hash",
		TTL:            1 * time.Hour,
		MaxEntries:     1000,
		CleanupInterval: 1 * time.Hour,
	}

	ctx := context.Background()
	deduplicator := NewDeduplicator(config, ctx)
	defer deduplicator.Stop()

	// 创建消息
	msg1 := &Message{
		Key:   []byte("test-key"),
		Value: []byte("test-value"),
	}

	// 第一次检查
	isDup, err := deduplicator.IsDuplicate(msg1)
	if err != nil {
		t.Fatalf("去重检查失败: %v", err)
	}
	if isDup {
		t.Error("第一条消息不应该是重复的")
	}

	// 相同消息应该重复
	isDup, err = deduplicator.IsDuplicate(msg1)
	if err != nil {
		t.Fatalf("去重检查失败: %v", err)
	}
	if !isDup {
		t.Error("相同消息应该被认为是重复的（hash策略）")
	}

	// 不同消息不应该重复
	msg2 := &Message{
		Key:   []byte("different-key"),
		Value: []byte("different-value"),
	}

	isDup, err = deduplicator.IsDuplicate(msg2)
	if err != nil {
		t.Fatalf("去重检查失败: %v", err)
	}
	if isDup {
		t.Error("不同消息不应该是重复的")
	}
}

// TestDeduplicator_UpdateConfig 测试更新配置
func TestDeduplicator_UpdateConfig(t *testing.T) {
	config := &DedupConfig{
		Enabled:        true,
		Strategy:       "key_value",
		TTL:            1 * time.Hour,
		MaxEntries:     1000,
		CleanupInterval: 1 * time.Hour,
	}

	ctx := context.Background()
	deduplicator := NewDeduplicator(config, ctx)
	defer deduplicator.Stop()

	// 更新配置
	newConfig := &DedupConfig{
		Enabled:        true,
		Strategy:       "key",
		TTL:            2 * time.Hour,
		MaxEntries:     2000,
		CleanupInterval: 2 * time.Hour,
	}

	deduplicator.UpdateConfig(newConfig)

	// 验证配置已更新
	if deduplicator.config.Strategy != "key" {
		t.Errorf("期望策略='key'，实际'%s'", deduplicator.config.Strategy)
	}
	if deduplicator.config.TTL != 2*time.Hour {
		t.Errorf("期望TTL=2h，实际%v", deduplicator.config.TTL)
	}
}

// TestDeduplicator_Stop 测试停止去重器
func TestDeduplicator_Stop(t *testing.T) {
	config := &DedupConfig{
		Enabled:        true,
		Strategy:       "key_value",
		TTL:            1 * time.Hour,
		MaxEntries:     1000,
		CleanupInterval: 100 * time.Millisecond, // 短间隔用于测试
	}

	ctx := context.Background()
	deduplicator := NewDeduplicator(config, ctx)

	// 添加一些消息
	for i := 0; i < 10; i++ {
		msg := &Message{
			Key:   []byte{byte(i)},
			Value: []byte{byte(i)},
		}
		deduplicator.IsDuplicate(msg)
	}

	// 停止去重器
	deduplicator.Stop()

	// 再次停止应该不会出错
	deduplicator.Stop()
}

// TestDeduplicator_LargeScale 测试大规模消息
func TestDeduplicator_LargeScale(t *testing.T) {
	config := &DedupConfig{
		Enabled:        true,
		Strategy:       "key_value",
		TTL:            1 * time.Hour,
		MaxEntries:     10000,
		CleanupInterval: 1 * time.Hour,
	}

	ctx := context.Background()
	deduplicator := NewDeduplicator(config, ctx)
	defer deduplicator.Stop()

	// 添加大量消息
	for i := 0; i < 1000; i++ {
		msg := &Message{
			Key:   []byte{byte(i >> 8), byte(i)},
			Value: []byte{byte(i >> 8), byte(i)},
		}

		_, err := deduplicator.IsDuplicate(msg)
		if err != nil {
			t.Fatalf("去重检查失败: %v", err)
		}
	}

	// 验证统计信息
	totalDedup, entries := deduplicator.GetStats()
	t.Logf("大规模测试统计: 去重%d条，条目%d", totalDedup, entries)

	if int64(entries) > config.MaxEntries {
		t.Errorf("条目数量不应该超过MaxEntries，实际%d，最大%d", entries, config.MaxEntries)
	}
}
