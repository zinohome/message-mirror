package main

import (
	"context"
	"testing"
)

func TestRateLimiter_NilCases(t *testing.T) {
	// 测试nil限流器的各种方法
	var limiter *RateLimiter = nil
	ctx := context.Background()
	
	// Wait应该不阻塞
	if err := limiter.Wait(ctx); err != nil {
		t.Errorf("nil限流器Wait应该不返回错误: %v", err)
	}
	
	// WaitN应该不阻塞
	if err := limiter.WaitN(ctx, 10); err != nil {
		t.Errorf("nil限流器WaitN应该不返回错误: %v", err)
	}
	
	// WaitBytes应该不阻塞
	if err := limiter.WaitBytes(ctx, 1000); err != nil {
		t.Errorf("nil限流器WaitBytes应该不返回错误: %v", err)
	}
	
	// Allow应该总是返回true
	if !limiter.Allow() {
		t.Error("nil限流器Allow应该返回true")
	}
	
	// AllowN应该总是返回true
	if !limiter.AllowN(10) {
		t.Error("nil限流器AllowN应该返回true")
	}
	
	// AllowBytes应该总是返回true
	if !limiter.AllowBytes(1000) {
		t.Error("nil限流器AllowBytes应该返回true")
	}
	
	// SetRate应该不panic
	limiter.SetRate(100.0)
	
	// GetRate应该返回0
	if limiter.GetRate() != 0 {
		t.Errorf("nil限流器GetRate应该返回0，实际%f", limiter.GetRate())
	}
}

func TestRateLimiter_ZeroRate(t *testing.T) {
	// 测试速率为0的情况（应该返回nil）
	limiter := NewRateLimiter(0.0, 10, true)
	if limiter != nil {
		t.Error("速率为0时应该返回nil")
	}
	
	limiter = NewRateLimiter(-1.0, 10, true)
	if limiter != nil {
		t.Error("速率为负数时应该返回nil")
	}
}

func TestRateLimiter_ContextCancellation(t *testing.T) {
	limiter := NewRateLimiter(1.0, 1, true)
	ctx, cancel := context.WithCancel(context.Background())
	
	// 立即取消上下文
	cancel()
	
	// 应该返回上下文取消错误
	err := limiter.Wait(ctx)
	if err == nil {
		t.Error("应该返回上下文取消错误")
	}
	
	if err != context.Canceled {
		t.Errorf("期望context.Canceled错误，实际%v", err)
	}
}

func TestRateLimiter_AllowBytes(t *testing.T) {
	limiter := NewBytesRateLimiter(1024.0, 1024)
	
	// 第一次应该允许
	if !limiter.AllowBytes(512) {
		t.Error("应该允许512字节")
	}
	
	// 再获取512字节应该允许（在突发范围内）
	if !limiter.AllowBytes(512) {
		t.Error("应该允许剩余的512字节")
	}
	
	// 再获取1字节应该被拒绝（超过突发）
	if limiter.AllowBytes(1) {
		t.Error("应该拒绝（超过突发大小）")
	}
}

func TestRateLimiter_GetRate(t *testing.T) {
	limiter := NewRateLimiter(100.0, 10, true)
	
	rate := limiter.GetRate()
	if rate != 100.0 {
		t.Errorf("期望速率100.0，实际%f", rate)
	}
	
	// 设置新速率
	limiter.SetRate(200.0)
	rate = limiter.GetRate()
	if rate != 200.0 {
		t.Errorf("期望速率200.0，实际%f", rate)
	}
}

