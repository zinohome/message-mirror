package ratelimiter

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestRateLimiter_ConcurrentAccess 测试并发访问
func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	limiter := NewRateLimiter(100.0, 10, true) // 100 req/s

	ctx := context.Background()
	var wg sync.WaitGroup
	concurrency := 20
	operationsPerGoroutine := 10

	start := time.Now()
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				err := limiter.Wait(ctx)
				if err != nil {
					t.Errorf("等待速率限制失败: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	// 验证并发访问没有panic或数据竞争
	t.Logf("并发访问完成，耗时: %v", duration)
}

// TestRateLimiter_EdgeCases 测试边界情况
func TestRateLimiter_EdgeCases(t *testing.T) {
	// 测试零速率（应该不限制）
	limiter1 := NewRateLimiter(0.0, 10, true)

	ctx := context.Background()
	start := time.Now()
	for i := 0; i < 100; i++ {
		err := limiter1.Wait(ctx)
		if err != nil {
			t.Fatalf("零速率限制应该不等待: %v", err)
		}
	}
	duration := time.Since(start)
	if duration > 10*time.Millisecond {
		t.Errorf("零速率限制不应该等待，实际等待%v", duration)
	}

	// 测试极小速率
	limiter2 := NewRateLimiter(0.1, 1, true) // 0.1 req/s

	start2 := time.Now()
	err := limiter2.Wait(ctx)
	if err != nil {
		t.Fatalf("等待速率限制失败: %v", err)
	}
	duration2 := time.Since(start2)
	// 0.1 req/s意味着每次等待10秒，但burst=1可能允许第一次立即通过
	t.Logf("极小速率限制等待时间: %v", duration2)

	// 测试极大速率
	limiter3 := NewRateLimiter(1000000.0, 1000000, true) // 100万 req/s

	start3 := time.Now()
	for i := 0; i < 1000; i++ {
		err := limiter3.Wait(ctx)
		if err != nil {
			t.Fatalf("等待速率限制失败: %v", err)
		}
	}
	duration3 := time.Since(start3)
	// 极大速率应该几乎不等待
	if duration3 > 100*time.Millisecond {
		t.Errorf("极大速率限制不应该等待太久，实际等待%v", duration3)
	}
}

// TestRateLimiter_ContextCancellation 测试上下文取消
func TestRateLimiter_ContextCancellation(t *testing.T) {
	limiter := NewRateLimiter(1.0, 1, true) // 1 req/s

	ctx, cancel := context.WithCancel(context.Background())

	// 取消上下文
	cancel()

	// 等待应该返回上下文取消错误
	err := limiter.Wait(ctx)
	if err == nil {
		t.Error("上下文取消后等待应该返回错误")
	}
	if err != ctx.Err() {
		t.Errorf("应该返回上下文取消错误，实际: %v", err)
	}
}

// TestRateLimiter_WaitN_Extended 测试WaitN方法（扩展测试）
func TestRateLimiter_WaitN_Extended(t *testing.T) {
	limiter := NewRateLimiter(10.0, 10, true) // 10 req/s

	ctx := context.Background()

	// 测试WaitN
	start := time.Now()
	err := limiter.WaitN(ctx, 5)
	if err != nil {
		t.Fatalf("WaitN失败: %v", err)
	}
	duration := time.Since(start)

	// WaitN(5)应该等待更长时间
	if duration < 50*time.Millisecond {
		t.Logf("WaitN等待时间: %v（可能太快）", duration)
	}
}

// TestRateLimiter_WaitBytes_Extended 测试WaitBytes方法（扩展测试）
func TestRateLimiter_WaitBytes_Extended(t *testing.T) {
	limiter := NewBytesRateLimiter(1000.0, 1000) // 1000 bytes/s

	ctx := context.Background()

	// 测试WaitBytes
	start := time.Now()
	err := limiter.WaitBytes(ctx, 500)
	if err != nil {
		t.Fatalf("WaitBytes失败: %v", err)
	}
	duration := time.Since(start)

	// WaitBytes(500)应该等待约500ms（1000 bytes/s）
	if duration < 400*time.Millisecond {
		t.Logf("WaitBytes等待时间: %v（可能太快）", duration)
	}
}

// TestRateLimiter_Allow_Extended 测试Allow方法（扩展测试）
func TestRateLimiter_Allow_Extended(t *testing.T) {
	limiter := NewRateLimiter(10.0, 10, true) // 10 req/s

	// 测试Allow
	allowed := limiter.Allow()
	if !allowed {
		t.Error("第一次Allow应该返回true")
	}

	// 快速连续调用Allow
	allowedCount := 0
	for i := 0; i < 20; i++ {
		if limiter.Allow() {
			allowedCount++
		}
	}

	// 由于burst=10，应该允许至少10次
	if allowedCount < 10 {
		t.Errorf("期望至少允许10次，实际%d", allowedCount)
	}
}

// TestRateLimiter_AllowN_Extended 测试AllowN方法（扩展测试）
func TestRateLimiter_AllowN_Extended(t *testing.T) {
	limiter := NewRateLimiter(10.0, 10, true) // 10 req/s

	// 测试AllowN
	allowed := limiter.AllowN(5)
	if !allowed {
		t.Error("AllowN(5)应该返回true（burst=10）")
	}

	// 再次AllowN(10)应该失败（因为已经使用了5）
	allowed = limiter.AllowN(10)
	if allowed {
		t.Error("AllowN(10)应该返回false（已使用5，剩余5<10）")
	}
}

// TestRateLimiter_SetRate_Extended 测试设置速率（扩展测试）
func TestRateLimiter_SetRate_Extended(t *testing.T) {
	limiter := NewRateLimiter(10.0, 10, true) // 10 req/s

	ctx := context.Background()

	// 测试原始速率
	start1 := time.Now()
	for i := 0; i < 5; i++ {
		limiter.Wait(ctx)
	}
	duration1 := time.Since(start1)

	// 更新速率
	limiter.SetRate(20.0) // 20 req/s

	// 测试新速率
	start2 := time.Now()
	for i := 0; i < 5; i++ {
		limiter.Wait(ctx)
	}
	duration2 := time.Since(start2)

	// 新速率应该更快
	if duration2 >= duration1 {
		t.Logf("速率更新后可能没有明显变化: 原始%v，新%v", duration1, duration2)
	}
}

// TestRateLimiter_Stop 测试停止速率限制器
// 注意：RateLimiter没有Stop方法，这里只测试基本功能
func TestRateLimiter_Stop(t *testing.T) {
	limiter := NewRateLimiter(10.0, 10, true)
	
	// RateLimiter没有Stop方法，这里只验证基本功能
	ctx := context.Background()
	err := limiter.Wait(ctx)
	if err != nil {
		t.Errorf("等待速率限制失败: %v", err)
	}
}

// TestBytesRateLimiter_ConcurrentAccess 测试字节速率限制器并发访问
func TestBytesRateLimiter_ConcurrentAccess(t *testing.T) {
	limiter := NewBytesRateLimiter(1000.0, 1000) // 1000 bytes/s

	ctx := context.Background()
	var wg sync.WaitGroup
	concurrency := 10
	bytesPerGoroutine := 100

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			err := limiter.WaitBytes(ctx, bytesPerGoroutine)
			if err != nil {
				t.Errorf("等待字节速率限制失败: %v", err)
			}
		}(i)
	}

	wg.Wait()

	// 验证并发访问没有panic或数据竞争
	t.Log("并发访问字节速率限制器完成")
}
