package optimization

import (
	"context"
	"testing"
	"time"
)

func TestBatchProcessor(t *testing.T) {
	processedBatches := 0
	batchMessages := 0
	
	processor := func(batch []*Message) error {
		processedBatches++
		batchMessages += len(batch)
		return nil
	}
	
	batchProcessor := NewBatchProcessor(10, 100*time.Millisecond, processor)
	batchProcessor.Start(context.Background())
	defer batchProcessor.Stop()
	
	// 添加消息
	for i := 0; i < 25; i++ {
		msg := &Message{
			Key:   []byte("key"),
			Value: []byte("value"),
		}
		batchProcessor.Add(msg)
	}
	
	// 等待批处理完成
	time.Sleep(200 * time.Millisecond)
	
	if processedBatches < 2 {
		t.Errorf("期望至少2个批次，实际%d个", processedBatches)
	}
	
	if batchMessages != 25 {
		t.Errorf("期望处理25条消息，实际%d条", batchMessages)
	}
}

func TestMessageCache(t *testing.T) {
	cache := NewMessageCache(100, 1*time.Hour)
	defer cache.Stop()
	
	// 设置缓存
	cache.Set("key1", "hash1")
	cache.Set("key2", "hash2")
	
	// 获取缓存
	exists, valid := cache.Get("key1")
	if !exists || !valid {
		t.Error("缓存条目应该存在且有效")
	}
	
	exists, valid = cache.Get("key3")
	if exists || valid {
		t.Error("不存在的缓存条目应该返回false")
	}
}

func TestConnectionPool(t *testing.T) {
	pool := NewConnectionPool(10)
	
	stats := pool.GetStats()
	if stats.ActiveConnections != 0 {
		t.Errorf("期望0个活动连接，实际%d", stats.ActiveConnections)
	}
}

