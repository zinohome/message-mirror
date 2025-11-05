package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetryManager_Retry_Success(t *testing.T) {
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
		if attempts == 1 {
			return errors.New("模拟错误")
		}
		return nil
	})
	
	if err != nil {
		t.Errorf("重试应该成功，但返回错误: %v", err)
	}
	
	if attempts != 2 {
		t.Errorf("期望2次尝试，实际%d次", attempts)
	}
}

func TestRetryManager_Retry_MaxRetriesExceeded(t *testing.T) {
	config := &RetryConfigInternal{
		MaxRetries:      2,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     50 * time.Millisecond,
		Multiplier:      2.0,
		Jitter:          false,
	}
	
	manager := NewRetryManager(config)
	ctx := context.Background()
	
	attempts := 0
	err := manager.Retry(ctx, func() error {
		attempts++
		return errors.New("始终失败")
	})
	
	if err == nil {
		t.Error("应该返回错误（超过最大重试次数）")
	}
	
	// 应该尝试3次（初始尝试 + 2次重试）
	if attempts != 3 {
		t.Errorf("期望3次尝试，实际%d次", attempts)
	}
}

func TestRetryManager_Retry_ContextCancellation(t *testing.T) {
	config := &RetryConfigInternal{
		MaxRetries:      10,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		Jitter:          false,
	}
	
	manager := NewRetryManager(config)
	ctx, cancel := context.WithCancel(context.Background())
	
	// 立即取消上下文
	cancel()
	
	err := manager.Retry(ctx, func() error {
		return errors.New("模拟错误")
	})
	
	if err == nil {
		t.Error("应该返回上下文取消错误")
	}
	
	if !errors.Is(err, context.Canceled) && err.Error() != "context cancelled: context canceled" {
		t.Errorf("应该返回上下文取消错误，实际: %v", err)
	}
}

func TestRetryManager_MultipleRetries(t *testing.T) {
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

func TestRetryManager_DefaultConfig(t *testing.T) {
	manager := NewRetryManager(nil)
	
	// 测试默认配置
	ctx := context.Background()
	attempts := 0
	err := manager.Retry(ctx, func() error {
		attempts++
		if attempts == 1 {
			return errors.New("模拟错误")
		}
		return nil
	})
	
	if err != nil {
		t.Errorf("重试应该成功，但返回错误: %v", err)
	}
	
	if attempts != 2 {
		t.Errorf("期望2次尝试，实际%d次", attempts)
	}
}

