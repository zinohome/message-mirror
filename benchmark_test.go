package main

import (
	"context"
	"testing"
	"time"
)

// BenchmarkRateLimiter_Wait 测试速率限制器的性能
func BenchmarkRateLimiter_Wait(b *testing.B) {
	limiter := NewRateLimiter(1000.0, 1000, true)
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Wait(ctx)
	}
}

// BenchmarkRateLimiter_Allow 测试Allow方法的性能
func BenchmarkRateLimiter_Allow(b *testing.B) {
	limiter := NewRateLimiter(1000.0, 1000, true)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow()
	}
}

// BenchmarkDeduplicator_IsDuplicate 测试去重器的性能
func BenchmarkDeduplicator_IsDuplicate(b *testing.B) {
	config := &DedupConfig{
		Enabled:        true,
		Strategy:       "key_value",
		TTL:            1 * time.Hour,
		MaxEntries:     1000000,
		CleanupInterval: 1 * time.Hour,
	}
	
	ctx := context.Background()
	deduplicator := NewDeduplicator(config, ctx)
	defer deduplicator.Stop()
	
	msg := &Message{
		Key:   []byte("benchmark-key"),
		Value: []byte("benchmark-value"),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		deduplicator.IsDuplicate(msg)
	}
}

// BenchmarkMessageProcessing 测试消息处理的性能
func BenchmarkMessageProcessing(b *testing.B) {
	msg := &Message{
		Key:       []byte("test-key"),
		Value:     make([]byte, 1024), // 1KB消息
		Headers:   make(map[string][]byte),
		Timestamp: time.Now(),
		Source:    "kafka",
		Metadata: map[string]interface{}{
			"topic":     "test-topic",
			"partition": int32(0),
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 模拟消息处理的关键步骤
		_ = len(msg.Value)
		_ = msg.Timestamp
		_ = msg.Source
	}
}

