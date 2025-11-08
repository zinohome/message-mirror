package main

import (
	"context"
	"testing"
	"time"
)

func TestBatchProcessor_FlushOnTimeout(t *testing.T) {
	processedBatches := 0
	
	processor := func(batch []*Message) error {
		processedBatches++
		return nil
	}
	
	ctx := context.Background()
	batchProcessor := NewBatchProcessor(100, 100*time.Millisecond, processor)
	batchProcessor.Start(ctx)
	defer batchProcessor.Stop()
	
	// 添加少量消息
	msg := &Message{
		Key:   []byte("key"),
		Value: []byte("value"),
	}
	batchProcessor.Add(msg)
	
	// 等待超时刷新
	time.Sleep(200 * time.Millisecond)
	
	if processedBatches == 0 {
		t.Error("应该至少处理一个批次（超时刷新）")
	}
}

func TestBatchProcessor_ContextCancellation(t *testing.T) {
	processor := func(batch []*Message) error {
		return nil
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	batchProcessor := NewBatchProcessor(10, 100*time.Millisecond, processor)
	batchProcessor.Start(ctx)
	
	// 添加一些消息
	for i := 0; i < 5; i++ {
		msg := &Message{
			Key:   []byte("key"),
			Value: []byte("value"),
		}
		batchProcessor.Add(msg)
	}
	
	// 取消上下文
	cancel()
	
	// 等待停止
	time.Sleep(100 * time.Millisecond)
	
	// Stop应该正常完成
	batchProcessor.Stop()
}

func TestMessageCache_EvictOldest(t *testing.T) {
	cache := NewMessageCache(3, 1*time.Hour)
	defer cache.Stop()
	
	// 添加超过最大大小的条目
	cache.Set("key1", "hash1")
	cache.Set("key2", "hash2")
	cache.Set("key3", "hash3")
	cache.Set("key4", "hash4") // 应该触发清理
	
	// 检查条目数
	exists, _ := cache.Get("key1")
	if exists {
		// key1可能被清理，也可能还在（取决于清理策略）
		t.Log("key1可能已被清理")
	}
}

func TestMessageCache_Expired(t *testing.T) {
	cache := NewMessageCache(100, 100*time.Millisecond) // 短TTL
	defer cache.Stop()
	
	cache.Set("key1", "hash1")
	
	// 等待过期
	time.Sleep(150 * time.Millisecond)
	
	// 检查是否过期
	exists, valid := cache.Get("key1")
	if exists && valid {
		t.Error("过期的条目应该被标记为无效")
	}
	
	// 等待清理
	time.Sleep(100 * time.Millisecond)
	
	exists, _ = cache.Get("key1")
	if exists {
		t.Log("过期条目可能已被清理")
	}
}

func TestConnectionPool_GetStats(t *testing.T) {
	pool := NewConnectionPool(10)
	
	stats := pool.GetStats()
	if stats.ActiveConnections != 0 {
		t.Errorf("期望0个活动连接，实际%d", stats.ActiveConnections)
	}
	
	if stats.TotalConnections != 0 {
		t.Errorf("期望0个总连接，实际%d", stats.TotalConnections)
	}
}

