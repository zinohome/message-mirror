package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestRetryManager_SuccessOnFirstAttempt 测试第一次尝试成功
func TestRetryManager_SuccessOnFirstAttempt(t *testing.T) {
	config := DefaultRetryConfig()
	rm := NewRetryManager(config)

	callCount := 0
	err := rm.Retry(context.Background(), func() error {
		callCount++
		return nil
	})

	if err != nil {
		t.Errorf("期望成功，但得到错误: %v", err)
	}
	if callCount != 1 {
		t.Errorf("期望调用1次，实际调用%d次", callCount)
	}
}

// TestRetryManager_SuccessAfterRetries 测试重试后成功
func TestRetryManager_SuccessAfterRetries(t *testing.T) {
	config := DefaultRetryConfig()
	config.MaxRetries = 3
	config.InitialInterval = 10 * time.Millisecond
	rm := NewRetryManager(config)

	callCount := 0
	err := rm.Retry(context.Background(), func() error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("期望最终成功，但得到错误: %v", err)
	}
	if callCount != 3 {
		t.Errorf("期望调用3次，实际调用%d次", callCount)
	}
}

// TestRetryManager_ExhaustRetries 测试重试次数耗尽
func TestRetryManager_ExhaustRetries(t *testing.T) {
	config := DefaultRetryConfig()
	config.MaxRetries = 2
	config.InitialInterval = 1 * time.Millisecond
	rm := NewRetryManager(config)

	callCount := 0
	permanentErr := errors.New("permanent error")
	err := rm.Retry(context.Background(), func() error {
		callCount++
		return permanentErr
	})

	if err == nil {
		t.Error("期望返回错误，但得到nil")
	}
	// MaxRetries=2 意味着初次尝试 + 2次重试 = 3次调用
	if callCount != 3 {
		t.Errorf("期望调用3次（1次初始+2次重试），实际调用%d次", callCount)
	}
}

// TestRetryManager_ContextCancellation 测试上下文取消
func TestRetryManager_ContextCancellation(t *testing.T) {
	config := DefaultRetryConfig()
	config.MaxRetries = 10
	config.InitialInterval = 100 * time.Millisecond
	rm := NewRetryManager(config)

	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := rm.Retry(ctx, func() error {
		callCount++
		return errors.New("test error")
	})

	if err == nil {
		t.Error("期望上下文取消错误")
	}
	if err != context.Canceled && !errors.Is(err, context.Canceled) {
		t.Errorf("期望context.Canceled错误，得到: %v", err)
	}
	if callCount > 3 {
		t.Errorf("上下文取消后不应继续重试，但调用了%d次", callCount)
	}
}

// TestRetryManager_ExponentialBackoffIntervals 测试指数退避间隔递增
func TestRetryManager_ExponentialBackoffIntervals(t *testing.T) {
	config := DefaultRetryConfig()
	config.MaxRetries = 3
	config.InitialInterval = 10 * time.Millisecond
	config.Multiplier = 2.0
	config.Jitter = false
	rm := NewRetryManager(config)

	var intervals []time.Duration
	lastTime := time.Now()

	err := rm.Retry(context.Background(), func() error {
		now := time.Now()
		if len(intervals) > 0 {
			intervals = append(intervals, now.Sub(lastTime))
		}
		lastTime = now
		return errors.New("test error")
	})

	if err == nil {
		t.Error("期望返回错误")
	}

	// 验证退避间隔递增
	if len(intervals) < 2 {
		t.Skip("间隔数据不足")
	}

	for i := 1; i < len(intervals); i++ {
		if intervals[i] < intervals[i-1] {
			t.Errorf("退避间隔应该递增: %v -> %v", intervals[i-1], intervals[i])
		}
	}
}

// TestRetryManager_WithJitter 测试带抖动的重试
func TestRetryManager_WithJitter(t *testing.T) {
	config := DefaultRetryConfig()
	config.MaxRetries = 5
	config.InitialInterval = 10 * time.Millisecond
	config.Jitter = true
	rm := NewRetryManager(config)

	var intervals []time.Duration
	lastTime := time.Now()

	rm.Retry(context.Background(), func() error {
		now := time.Now()
		if len(intervals) > 0 {
			intervals = append(intervals, now.Sub(lastTime))
		}
		lastTime = now
		return errors.New("test error")
	})

	// 有抖动时，间隔不应完全相同
	if len(intervals) >= 2 {
		allSame := true
		first := intervals[0]
		for _, interval := range intervals[1:] {
			if interval != first {
				allSame = false
				break
			}
		}
		if allSame {
			t.Log("警告: 启用抖动但所有间隔相同，可能是随机性问题")
		}
	}
}

// TestRetryManager_MaxIntervalLimit 测试最大间隔限制不被突破
func TestRetryManager_MaxIntervalLimit(t *testing.T) {
	config := DefaultRetryConfig()
	config.MaxRetries = 5
	config.InitialInterval = 10 * time.Millisecond
	config.MaxInterval = 50 * time.Millisecond
	config.Multiplier = 10.0
	config.Jitter = false
	rm := NewRetryManager(config)

	var intervals []time.Duration
	lastTime := time.Now()

	rm.Retry(context.Background(), func() error {
		now := time.Now()
		if len(intervals) > 0 {
			interval := now.Sub(lastTime)
			intervals = append(intervals, interval)
			// 验证间隔不超过最大值（加一些误差容忍）
			if interval > config.MaxInterval+10*time.Millisecond {
				t.Errorf("间隔%v超过最大间隔%v", interval, config.MaxInterval)
			}
		}
		lastTime = now
		return errors.New("test error")
	})
}

// TestDefaultRetryConfig 测试默认配置
func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxRetries <= 0 {
		t.Error("MaxRetries应该大于0")
	}
	if config.InitialInterval <= 0 {
		t.Error("InitialInterval应该大于0")
	}
	if config.MaxInterval <= 0 {
		t.Error("MaxInterval应该大于0")
	}
	if config.Multiplier <= 1.0 {
		t.Error("Multiplier应该大于1.0")
	}
	if config.MaxInterval < config.InitialInterval {
		t.Error("MaxInterval应该大于等于InitialInterval")
	}
}
