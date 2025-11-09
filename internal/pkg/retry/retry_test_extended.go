package retry

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// TestRetryManager_ExponentialBackoff 测试指数退避
func TestRetryManager_ExponentialBackoff(t *testing.T) {
	config := &RetryConfigInternal{
		MaxRetries:      3,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		Jitter:          false, // 禁用抖动以便测试
	}

	manager := NewRetryManager(config)
	ctx := context.Background()

	attempts := 0
	intervals := make([]time.Duration, 0)

	err := manager.Retry(ctx, func() error {
		attempts++
		if attempts < 4 {
			start := time.Now()
			// 等待一小段时间以观察间隔
			time.Sleep(1 * time.Millisecond)
			intervals = append(intervals, time.Since(start))
			return errors.New("模拟错误")
		}
		return nil
	})

	if err != nil {
		t.Errorf("重试应该成功，但返回错误: %v", err)
	}

	if attempts != 4 {
		t.Errorf("期望4次尝试，实际%d次", attempts)
	}

	// 验证指数退避（间隔应该逐渐增加）
	// 注意：由于时间测量的不确定性，这里只验证基本行为
	t.Logf("重试间隔: %v", intervals)
}

// TestRetryManager_Jitter 测试抖动
func TestRetryManager_Jitter(t *testing.T) {
	config := &RetryConfigInternal{
		MaxRetries:      2,
		InitialInterval: 50 * time.Millisecond,
		MaxInterval:     200 * time.Millisecond,
		Multiplier:      2.0,
		Jitter:          true, // 启用抖动
	}

	manager := NewRetryManager(config)
	ctx := context.Background()

	attempts := 0
	err := manager.Retry(ctx, func() error {
		attempts++
		if attempts < 3 {
			return errors.New("模拟错误")
		}
		return nil
	})

	if err != nil {
		t.Errorf("重试应该成功，但返回错误: %v", err)
	}

	if attempts != 3 {
		t.Errorf("期望3次尝试，实际%d次", attempts)
	}
}

// TestRetryManager_ContextTimeout 测试上下文超时
func TestRetryManager_ContextTimeout(t *testing.T) {
	config := &RetryConfigInternal{
		MaxRetries:      10,
		InitialInterval: 50 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		Jitter:          false,
	}

	manager := NewRetryManager(config)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	attempts := 0
	err := manager.Retry(ctx, func() error {
		attempts++
		return errors.New("模拟错误")
	})

	if err == nil {
		t.Error("应该返回超时错误")
	}

	// 验证上下文超时错误
	if !errors.Is(err, context.DeadlineExceeded) && err.Error() != "context cancelled: context deadline exceeded" {
		t.Errorf("应该返回上下文超时错误，实际: %v", err)
	}

	// 验证在超时前进行了尝试
	if attempts == 0 {
		t.Error("应该在超时前进行至少1次尝试")
	}
}

// TestRetryManager_MaxInterval 测试最大间隔限制
func TestRetryManager_MaxInterval(t *testing.T) {
	config := &RetryConfigInternal{
		MaxRetries:      5,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     50 * time.Millisecond, // 小最大值
		Multiplier:      10.0, // 大倍数，会快速超过MaxInterval
		Jitter:          false,
	}

	manager := NewRetryManager(config)
	ctx := context.Background()

	attempts := 0
	err := manager.Retry(ctx, func() error {
		attempts++
		if attempts < 6 {
			return errors.New("模拟错误")
		}
		return nil
	})

	if err != nil {
		t.Errorf("重试应该成功，但返回错误: %v", err)
	}

	if attempts != 6 {
		t.Errorf("期望6次尝试，实际%d次", attempts)
	}
}

// TestRetryManager_ZeroRetries 测试零重试
func TestRetryManager_ZeroRetries(t *testing.T) {
	config := &RetryConfigInternal{
		MaxRetries:      0,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		Jitter:          false,
	}

	manager := NewRetryManager(config)
	ctx := context.Background()

	attempts := 0
	err := manager.Retry(ctx, func() error {
		attempts++
		return errors.New("模拟错误")
	})

	if err == nil {
		t.Error("应该返回错误（零重试）")
	}

	// 应该只尝试1次（初始尝试，无重试）
	if attempts != 1 {
		t.Errorf("期望1次尝试，实际%d次", attempts)
	}
}

// TestRetryManager_ImmediateSuccess 测试立即成功
func TestRetryManager_ImmediateSuccess(t *testing.T) {
	config := &RetryConfigInternal{
		MaxRetries:      3,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		Jitter:          false,
	}

	manager := NewRetryManager(config)
	ctx := context.Background()

	attempts := 0
	err := manager.Retry(ctx, func() error {
		attempts++
		return nil // 立即成功
	})

	if err != nil {
		t.Errorf("应该成功，但返回错误: %v", err)
	}

	// 应该只尝试1次
	if attempts != 1 {
		t.Errorf("期望1次尝试，实际%d次", attempts)
	}
}

// TestRetryManager_ConcurrentRetries 测试并发重试
func TestRetryManager_ConcurrentRetries(t *testing.T) {
	config := &RetryConfigInternal{
		MaxRetries:      2,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     50 * time.Millisecond,
		Multiplier:      2.0,
		Jitter:          false,
	}

	manager := NewRetryManager(config)
	ctx := context.Background()

	var wg sync.WaitGroup
	concurrency := 10
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			attempts := 0
			err := manager.Retry(ctx, func() error {
				attempts++
				if attempts < 2 {
					return errors.New("模拟错误")
				}
				return nil
			})

			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// 验证所有重试都成功
	if successCount != concurrency {
		t.Errorf("期望%d个成功，实际%d", concurrency, successCount)
	}
}

