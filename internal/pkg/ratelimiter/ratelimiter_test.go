package ratelimiter

import (
	"context"
	"testing"
	"time"
)

func TestRateLimiter_Wait(t *testing.T) {
	// 测试速率限制器：每秒10个令牌
	limiter := NewRateLimiter(10.0, 20, true)
	ctx := context.Background()

	start := time.Now()
	
	// 连续获取20个令牌（应该很快，因为有突发）
	for i := 0; i < 20; i++ {
		if err := limiter.Wait(ctx); err != nil {
			t.Fatalf("获取令牌失败: %v", err)
		}
	}
	
	elapsed := time.Since(start)
	
	// 突发应该很快完成（小于1秒）
	if elapsed > 2*time.Second {
		t.Errorf("突发获取令牌耗时过长: %v", elapsed)
	}

	// 再获取10个令牌，应该需要等待约1秒
	start = time.Now()
	for i := 0; i < 10; i++ {
		if err := limiter.Wait(ctx); err != nil {
			t.Fatalf("获取令牌失败: %v", err)
		}
	}
	elapsed = time.Since(start)
	
	// 应该在1秒左右（允许一些误差）
	if elapsed < 800*time.Millisecond || elapsed > 2*time.Second {
		t.Errorf("获取令牌耗时异常: %v，期望约1秒", elapsed)
	}
}

func TestRateLimiter_WaitN(t *testing.T) {
	limiter := NewRateLimiter(10.0, 5, true) // 突发大小5
	ctx := context.Background()

	start := time.Now()
	
	// 一次性获取5个令牌（在突发范围内）
	if err := limiter.WaitN(ctx, 5); err != nil {
		t.Fatalf("获取令牌失败: %v", err)
	}
	
	elapsed := time.Since(start)
	
	// 应该很快（在突发范围内）
	if elapsed > 1*time.Second {
		t.Errorf("获取令牌耗时过长: %v", elapsed)
	}

	// 再获取10个，应该需要等待约1秒（因为突发只有5，需要等待令牌恢复）
	start = time.Now()
	if err := limiter.WaitN(ctx, 10); err != nil {
		t.Fatalf("获取令牌失败: %v", err)
	}
	elapsed = time.Since(start)
	
	// 应该需要等待（因为突发大小为5，需要等待5个令牌恢复）
	if elapsed < 400*time.Millisecond || elapsed > 2*time.Second {
		t.Errorf("获取令牌耗时异常: %v，期望约0.5-1秒", elapsed)
	}
}

func TestRateLimiter_WaitBytes(t *testing.T) {
	limiter := NewBytesRateLimiter(1024.0, 1024) // 1KB/s, 1KB突发
	ctx := context.Background()

	start := time.Now()
	
	// 获取1KB（应该很快，在突发范围内）
	if err := limiter.WaitBytes(ctx, 1024); err != nil {
		t.Fatalf("获取字节令牌失败: %v", err)
	}
	
	elapsed := time.Since(start)
	if elapsed > 1*time.Second {
		t.Errorf("获取字节令牌耗时过长: %v", elapsed)
	}

	// 再获取1KB，应该需要等待约1秒（因为突发已经用完）
	start = time.Now()
	if err := limiter.WaitBytes(ctx, 1024); err != nil {
		t.Fatalf("获取字节令牌失败: %v", err)
	}
	elapsed = time.Since(start)
	
	// 应该需要等待约1秒（因为需要等待1024字节的令牌恢复）
	if elapsed < 800*time.Millisecond || elapsed > 2*time.Second {
		t.Errorf("获取字节令牌耗时异常: %v，期望约1秒", elapsed)
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	limiter := NewRateLimiter(10.0, 5, true)
	
	// 前5个应该允许
	for i := 0; i < 5; i++ {
		if !limiter.Allow() {
			t.Errorf("第%d个令牌应该被允许", i+1)
		}
	}
	
	// 第6个应该被拒绝（超过突发大小）
	// 等待一小段时间让令牌恢复
	time.Sleep(100 * time.Millisecond)
	
	// 现在应该允许（因为令牌已经恢复）
	if !limiter.Allow() {
		t.Error("令牌应该已经恢复，应该允许")
	}
}

func TestRateLimiter_AllowN(t *testing.T) {
	limiter := NewRateLimiter(10.0, 10, true)
	
	// 获取10个应该允许
	if !limiter.AllowN(10) {
		t.Error("应该允许获取10个令牌")
	}
	
	// 再获取1个应该被拒绝
	if limiter.AllowN(1) {
		t.Error("应该拒绝获取（超过突发大小）")
	}
}

func TestRateLimiter_NilLimiter(t *testing.T) {
	// nil限流器应该不限制
	var limiter *RateLimiter = nil
	ctx := context.Background()
	
	if err := limiter.Wait(ctx); err != nil {
		t.Errorf("nil限流器应该不限制: %v", err)
	}
	
	if !limiter.Allow() {
		t.Error("nil限流器应该允许所有请求")
	}
}

func TestRateLimiter_SetRate(t *testing.T) {
	limiter := NewRateLimiter(10.0, 10, true)
	
	// 设置新速率
	limiter.SetRate(20.0)
	
	rate := limiter.GetRate()
	if rate != 20.0 {
		t.Errorf("速率设置失败，期望20.0，实际%f", rate)
	}
}

