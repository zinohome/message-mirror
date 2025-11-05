package main

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// RetryConfigInternal 重试配置（内部使用）
type RetryConfigInternal struct {
	MaxRetries      int           // 最大重试次数
	InitialInterval time.Duration // 初始重试间隔
	MaxInterval     time.Duration // 最大重试间隔
	Multiplier      float64       // 退避倍数（指数退避）
	Jitter          bool          // 是否添加随机抖动
}

// DefaultRetryConfig 默认重试配置
func DefaultRetryConfig() *RetryConfigInternal {
	return &RetryConfigInternal{
		MaxRetries:      3,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     10 * time.Second,
		Multiplier:      2.0,
		Jitter:          true,
	}
}

// RetryManager 重试管理器
type RetryManager struct {
	config *RetryConfigInternal
	mu     sync.RWMutex
}

// NewRetryManager 创建新的重试管理器
func NewRetryManager(config *RetryConfigInternal) *RetryManager {
	if config == nil {
		config = DefaultRetryConfig()
	}
	return &RetryManager{
		config: config,
	}
}

// Retry 执行重试操作
func (rm *RetryManager) Retry(ctx context.Context, fn func() error) error {
	rm.mu.RLock()
	maxRetries := rm.config.MaxRetries
	initialInterval := rm.config.InitialInterval
	maxInterval := rm.config.MaxInterval
	multiplier := rm.config.Multiplier
	jitter := rm.config.Jitter
	rm.mu.RUnlock()

	var lastErr error
	interval := initialInterval

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// 尝试执行操作
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// 如果是最后一次尝试，返回错误
		if attempt >= maxRetries {
			break
		}

		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		default:
		}

		// 计算下次重试的间隔（指数退避）
		if attempt < maxRetries {
			// 指数退避
			nextInterval := time.Duration(float64(interval) * multiplier)

			// 限制最大间隔
			if nextInterval > maxInterval {
				nextInterval = maxInterval
			}

			// 添加抖动（可选）
			if jitter {
				nextInterval = addJitter(nextInterval)
			}

			// 等待重试间隔
			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled: %w", ctx.Err())
			case <-time.After(interval):
				interval = nextInterval
			}
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// RetryWithBackoff 执行带退避的重试（自定义重试逻辑）
func (rm *RetryManager) RetryWithBackoff(ctx context.Context, fn func(attempt int) error) error {
	rm.mu.RLock()
	maxRetries := rm.config.MaxRetries
	initialInterval := rm.config.InitialInterval
	maxInterval := rm.config.MaxInterval
	multiplier := rm.config.Multiplier
	jitter := rm.config.Jitter
	rm.mu.RUnlock()

	var lastErr error
	interval := initialInterval

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// 尝试执行操作
		err := fn(attempt)
		if err == nil {
			return nil
		}

		lastErr = err

		// 如果是最后一次尝试，返回错误
		if attempt >= maxRetries {
			break
		}

		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		default:
		}

		// 计算下次重试的间隔
		nextInterval := time.Duration(float64(interval) * multiplier)
		if nextInterval > maxInterval {
			nextInterval = maxInterval
		}

		if jitter {
			nextInterval = addJitter(nextInterval)
		}

		// 等待重试间隔
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		case <-time.After(interval):
			interval = nextInterval
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// addJitter 添加随机抖动（±25%）
func addJitter(interval time.Duration) time.Duration {
	jitter := time.Duration(float64(interval) * 0.25)
	jitterAmount := time.Duration(float64(jitter) * (math.Mod(float64(time.Now().UnixNano()), 2.0) - 0.5))
	return interval + jitterAmount
}

// SetConfig 更新重试配置
func (rm *RetryManager) SetConfig(config *RetryConfigInternal) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.config = config
}

// GetConfig 获取当前重试配置
func (rm *RetryManager) GetConfig() *RetryConfigInternal {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.config
}

