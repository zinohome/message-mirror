package main

import (
	"context"
	"testing"
	"time"
)

func TestDeduplicator_CleanupExpired(t *testing.T) {
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
	
	msg1 := &Message{
		Key:   []byte("test-key-1"),
		Value: []byte("test-value-1"),
	}
	
	msg2 := &Message{
		Key:   []byte("test-key-2"),
		Value: []byte("test-value-2"),
	}
	
	// 添加消息
	deduplicator.IsDuplicate(msg1)
	deduplicator.IsDuplicate(msg2)
	
	// 等待TTL过期
	time.Sleep(150 * time.Millisecond)
	
	// 等待清理
	time.Sleep(100 * time.Millisecond)
	
	// 检查统计信息
	totalDedup, entries := deduplicator.GetStats()
	t.Logf("去重统计: 总数=%d, 当前记录数=%d", totalDedup, entries)
}

func TestDeduplicator_CleanupOldest(t *testing.T) {
	config := &DedupConfig{
		Enabled:        true,
		Strategy:       "key_value",
		TTL:            1 * time.Hour,
		MaxEntries:     3, // 小数量用于测试清理
		CleanupInterval: 1 * time.Hour,
	}
	
	ctx := context.Background()
	deduplicator := NewDeduplicator(config, ctx)
	defer deduplicator.Stop()
	
	// 添加超过最大记录数的消息
	for i := 0; i < 5; i++ {
		msg := &Message{
			Key:   []byte("test-key"),
			Value: []byte{byte(i)},
		}
		deduplicator.IsDuplicate(msg)
	}
	
	// 检查统计信息
	totalDedup, entries := deduplicator.GetStats()
	t.Logf("去重统计: 总数=%d, 当前记录数=%d", totalDedup, entries)
	
	// 记录数应该不超过MaxEntries
	if entries > int(config.MaxEntries) {
		t.Errorf("记录数不应该超过MaxEntries，实际%d，最大%d", entries, config.MaxEntries)
	}
}

func TestDeduplicator_GenerateHash_AllStrategies(t *testing.T) {
	configs := []struct {
		name    string
		config  *DedupConfig
	}{
		{"key", &DedupConfig{Enabled: true, Strategy: "key", TTL: 1 * time.Hour, MaxEntries: 1000, CleanupInterval: 1 * time.Hour}},
		{"value", &DedupConfig{Enabled: true, Strategy: "value", TTL: 1 * time.Hour, MaxEntries: 1000, CleanupInterval: 1 * time.Hour}},
		{"key_value", &DedupConfig{Enabled: true, Strategy: "key_value", TTL: 1 * time.Hour, MaxEntries: 1000, CleanupInterval: 1 * time.Hour}},
		{"hash", &DedupConfig{Enabled: true, Strategy: "hash", TTL: 1 * time.Hour, MaxEntries: 1000, CleanupInterval: 1 * time.Hour}},
		{"invalid", &DedupConfig{Enabled: true, Strategy: "invalid", TTL: 1 * time.Hour, MaxEntries: 1000, CleanupInterval: 1 * time.Hour}},
	}
	
	msg := &Message{
		Key:   []byte("test-key"),
		Value: []byte("test-value"),
		Headers: map[string][]byte{
			"header1": []byte("value1"),
		},
	}
	
	for _, tc := range configs {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			deduplicator := NewDeduplicator(tc.config, ctx)
			defer deduplicator.Stop()
			
			// 测试hash生成（通过IsDuplicate间接测试）
			isDup, err := deduplicator.IsDuplicate(msg)
			if err != nil {
				t.Errorf("生成hash失败: %v", err)
			}
			
			if isDup {
				t.Error("第一条消息不应该是重复的")
			}
			
			// 再次检查应该是重复的
			isDup, err = deduplicator.IsDuplicate(msg)
			if err != nil {
				t.Errorf("生成hash失败: %v", err)
			}
			
			if !isDup {
				t.Error("相同消息应该是重复的")
			}
		})
	}
}

