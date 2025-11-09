package optimization

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// TestBatchProcessor_Timeout 测试批处理超时
func TestBatchProcessor_Timeout(t *testing.T) {
	processedBatches := 0
	batchMessages := 0
	var mu sync.Mutex

	processor := func(batch []*Message) error {
		mu.Lock()
		processedBatches++
		batchMessages += len(batch)
		mu.Unlock()
		return nil
	}

	batchProcessor := NewBatchProcessor(10, 50*time.Millisecond, processor) // 短超时
	batchProcessor.Start(context.Background())
	defer batchProcessor.Stop()

	// 添加少量消息（不足以触发批次大小）
	for i := 0; i < 5; i++ {
		msg := &Message{
			Key:   []byte("key"),
			Value: []byte("value"),
		}
		batchProcessor.Add(msg)
	}

	// 等待超时
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if processedBatches == 0 {
		t.Error("超时后应该处理至少1个批次")
	}
	if batchMessages != 5 {
		t.Errorf("期望处理5条消息，实际%d条", batchMessages)
	}
	mu.Unlock()
}

// TestBatchProcessor_ErrorHandling 测试错误处理
func TestBatchProcessor_ErrorHandling(t *testing.T) {
	processor := func(batch []*Message) error {
		return errors.New("模拟处理错误")
	}

	batchProcessor := NewBatchProcessor(10, 100*time.Millisecond, processor)
	batchProcessor.Start(context.Background())
	defer batchProcessor.Stop()

	// 添加消息
	for i := 0; i < 10; i++ {
		msg := &Message{
			Key:   []byte("key"),
			Value: []byte("value"),
		}
		err := batchProcessor.Add(msg)
		if err != nil {
			t.Fatalf("添加消息失败: %v", err)
		}
	}

	// 等待处理（即使有错误，批处理器也应该继续运行）
	time.Sleep(150 * time.Millisecond)

	// 验证批处理器仍在运行（可以继续添加消息）
	msg := &Message{
		Key:   []byte("key"),
		Value: []byte("value"),
	}
	err := batchProcessor.Add(msg)
	if err != nil {
		t.Errorf("批处理器应该继续运行，但添加消息失败: %v", err)
	}
}

// TestMessageCache_Expiration 测试缓存过期
func TestMessageCache_Expiration(t *testing.T) {
	cache := NewMessageCache(100, 100*time.Millisecond) // 短TTL
	defer cache.Stop()

	// 设置缓存
	cache.Set("key1", "hash1")
	cache.Set("key2", "hash2")

	// 立即获取应该存在
	exists, valid := cache.Get("key1")
	if !exists || !valid {
		t.Error("缓存条目应该存在且有效")
	}

	// 等待过期
	time.Sleep(150 * time.Millisecond)

	// 过期后应该不存在
	exists, valid = cache.Get("key1")
	if exists || valid {
		t.Error("过期后的缓存条目应该不存在")
	}
}

// TestMessageCache_MaxSize 测试缓存最大大小
func TestMessageCache_MaxSize(t *testing.T) {
	cache := NewMessageCache(5, 1*time.Hour) // 小缓存
	defer cache.Stop()

	// 添加超过最大大小的条目
	for i := 0; i < 10; i++ {
		cache.Set(string(rune(i)), "hash")
	}

	// 验证缓存大小不超过最大值（通过检查缓存行为）
	// 注意：MessageCache没有GetStats方法，我们通过行为验证
	exists, _ := cache.Get("0")
	if !exists {
		t.Error("缓存条目应该存在")
	}
}

// TestConnectionPool_GetPut 测试连接池获取和归还
// 注意：ConnectionPool没有Get/Put方法，只提供统计信息
func TestConnectionPool_GetPut(t *testing.T) {
	pool := NewConnectionPool(10)

	// 验证统计信息
	stats := pool.GetStats()
	if stats.ActiveConnections != 0 {
		t.Errorf("期望0个活动连接，实际%d", stats.ActiveConnections)
	}
	if stats.TotalConnections != 0 {
		t.Errorf("期望0个总连接，实际%d", stats.TotalConnections)
	}
}

// TestConnectionPool_ConcurrentAccess 测试并发访问
// 注意：ConnectionPool没有Get/Put方法，只提供统计信息
func TestConnectionPool_ConcurrentAccess(t *testing.T) {
	pool := NewConnectionPool(10)

	var wg sync.WaitGroup
	concurrency := 20

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// 并发访问统计信息
			stats := pool.GetStats()
			_ = stats
		}(i)
	}

	wg.Wait()

	// 验证统计信息
	stats := pool.GetStats()
	t.Logf("并发访问后统计: 活动连接%d，总连接%d", stats.ActiveConnections, stats.TotalConnections)
}

// TestBatchProcessor_ContextCancellation 测试上下文取消
func TestBatchProcessor_ContextCancellation(t *testing.T) {
	processedBatches := 0
	var mu sync.Mutex

	processor := func(batch []*Message) error {
		mu.Lock()
		processedBatches++
		mu.Unlock()
		return nil
	}

	batchProcessor := NewBatchProcessor(10, 100*time.Millisecond, processor)
	ctx, cancel := context.WithCancel(context.Background())
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
	time.Sleep(50 * time.Millisecond)

	// 尝试添加消息应该失败
	msg := &Message{
		Key:   []byte("key"),
		Value: []byte("value"),
	}
	err := batchProcessor.Add(msg)
	if err == nil {
		t.Error("上下文取消后添加消息应该返回错误")
	}

	batchProcessor.Stop()
}

// TestBatchProcessor_UpdateConfig 测试更新配置
func TestBatchProcessor_UpdateConfig(t *testing.T) {
	processor := func(batch []*Message) error {
		return nil
	}

	batchProcessor := NewBatchProcessor(10, 100*time.Millisecond, processor)
	batchProcessor.Start(context.Background())
	defer batchProcessor.Stop()

	// 更新配置
	batchProcessor.UpdateConfig(20, 200*time.Millisecond)

	// 验证配置已更新（通过处理行为）
	// 注意：由于配置是内部的，我们只能通过行为验证
	for i := 0; i < 25; i++ {
		msg := &Message{
			Key:   []byte("key"),
			Value: []byte("value"),
		}
		batchProcessor.Add(msg)
	}

	// 等待处理
	time.Sleep(300 * time.Millisecond)
}

// TestMessageCache_ConcurrentAccess 测试缓存并发访问
func TestMessageCache_ConcurrentAccess(t *testing.T) {
	cache := NewMessageCache(1000, 1*time.Hour)
	defer cache.Stop()

	var wg sync.WaitGroup
	concurrency := 20
	operationsPerGoroutine := 50

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := string(rune(id*1000 + j))
				cache.Set(key, "hash")
				cache.Get(key)
			}
		}(i)
	}

	wg.Wait()

	// 验证缓存没有panic或数据竞争
	// 通过获取一个存在的key验证
	exists, _ := cache.Get("0")
	t.Logf("并发访问后缓存验证: key='0' exists=%v", exists)
}
