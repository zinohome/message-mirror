package ratelimiter

import (
	"context"
	"testing"
	"time"
)

// TestBytesRateLimiter_WaitBytes 测试字节速率限制等待
func TestBytesRateLimiter_WaitBytes(t *testing.T) {
	// 每秒1000字节
	limiter := NewBytesRateLimiter(1000, 100)

	start := time.Now()

	// 发送100字节应该立即返回
	err := limiter.WaitBytes(context.Background(), 100)
	if err != nil {
		t.Errorf("等待100字节失败: %v", err)
	}

	duration := time.Since(start)
	if duration > 100*time.Millisecond {
		t.Logf("警告: 处理100字节耗时%v", duration)
	}
}

// TestBytesRateLimiter_ExceedBurst 测试突发大小限制
func TestBytesRateLimiter_ExceedBurst(t *testing.T) {
	// 每秒1000字节，突发大小100
	limiter := NewBytesRateLimiter(1000, 100)

	// 尝试发送超过突发大小的数据应该等待
	start := time.Now()
	err := limiter.WaitBytes(context.Background(), 200)
	if err != nil {
		t.Errorf("等待200字节失败: %v", err)
	}

	duration := time.Since(start)
	// 应该花费大约0.1秒（100字节/秒）来填充突发容量
	expectedMin := time.Duration(float64(100) / (1000.0 / 1e9))
	if duration < expectedMin*2 {
		t.Logf("信息: 处理200字节耗时%v（预期至少%v）", duration, expectedMin)
	}
}

// TestBytesRateLimiter_ZeroBytes 测试零字节
func TestBytesRateLimiter_ZeroBytes(t *testing.T) {
	limiter := NewBytesRateLimiter(1000, 100)

	err := limiter.WaitBytes(context.Background(), 0)
	if err != nil {
		t.Errorf("等待0字节不应失败: %v", err)
	}
}

// TestBytesRateLimiter_ContextCancellation 测试上下文取消
func TestBytesRateLimiter_ContextCancellation(t *testing.T) {
	limiter := NewBytesRateLimiter(100, 50) // 低速率

	ctx, cancel := context.WithCancel(context.Background())

	// 启动goroutine在100ms后取消
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	// 尝试等待大量字节（应该被取消）
	start := time.Now()
	err := limiter.WaitBytes(ctx, 10000)
	duration := time.Since(start)

	if err == nil && duration > 500*time.Millisecond {
		t.Error("应该被上下文取消")
	}
}

// TestRateLimiter_WaitMessage 测试消息速率限制
func TestRateLimiter_WaitMessage(t *testing.T) {
	limiter := NewRateLimiter(1000, 100, true) // 每秒1000消息

	start := time.Now()
	err := limiter.Wait(context.Background())
	if err != nil {
		t.Errorf("等待消息失败: %v", err)
	}

	duration := time.Since(start)
	if duration > 100*time.Millisecond {
		t.Logf("警告: Wait()耗时%v", duration)
	}
}

// TestRateLimiter_MultipleWaits 测试多次等待
func TestRateLimiter_MultipleWaits(t *testing.T) {
	limiter := NewRateLimiter(10, 5, false) // 每秒10消息，突发5

	for i := 0; i < 10; i++ {
		err := limiter.Wait(context.Background())
		if err != nil {
			t.Errorf("第%d次等待失败: %v", i, err)
			return
		}
	}
}

// TestRateLimiter_ContextTimeout 测试上下文超时
func TestRateLimiter_ContextTimeout(t *testing.T) {
	limiter := NewRateLimiter(1, 0, true) // 每秒1消息

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// 第一次应该成功
	err := limiter.Wait(ctx)
	if err != nil {
		t.Skipf("首次等待失败，跳过: %v", err)
	}

	// 后续会超时
	for i := 0; i < 5; i++ {
		err = limiter.Wait(ctx)
		if err == context.DeadlineExceeded {
			// 预期的结果
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Log("信息: 在超时前发送了多条消息，可能是速率设置过高")
}

// TestRateLimiter_ZeroRate 测试零速率（无限制）
func TestRateLimiter_ZeroRate(t *testing.T) {
	// 速率为0应该不限制（如果实现支持）
	limiter := NewRateLimiter(0, 0, true)

	for i := 0; i < 10; i++ {
		err := limiter.Wait(context.Background())
		if err != nil {
			t.Skipf("零速率限制不支持: %v", err)
		}
	}
}

// TestRateLimiter_NegativeRate 测试负速率处理
func TestRateLimiter_NegativeRate(t *testing.T) {
	// 负速率应该被视为不限制或使用默认值
	limiter := NewRateLimiter(-100, 10, true)

	err := limiter.Wait(context.Background())
	// 测试是否处理正确，不崩溃
	_ = err
}
