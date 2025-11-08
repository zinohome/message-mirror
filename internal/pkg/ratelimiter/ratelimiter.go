package ratelimiter

import (
	"context"
	"sync"
	"time"
)

// RateLimiter 令牌桶速率限制器
type RateLimiter struct {
	mu            sync.Mutex
	tokens        float64       // 当前令牌数
	maxTokens     float64       // 最大令牌数
	rate          float64       // 每秒生成的令牌数
	lastUpdate    time.Time     // 上次更新时间
	burstSize     int           // 突发大小（允许的突发请求数）
	perMessage    bool          // 是否按消息计数（true）或按字节计数（false）
	bytesPerToken float64       // 每个令牌对应的字节数（当perMessage=false时使用）
}

// NewRateLimiter 创建新的速率限制器
// rate: 速率限制（消息/秒 或 字节/秒）
// burst: 突发大小（允许的突发数量）
// perMessage: true表示按消息计数，false表示按字节计数
func NewRateLimiter(rate float64, burst int, perMessage bool) *RateLimiter {
	if rate <= 0 {
		// 如果速率为0或负数，表示不限制
		return nil
	}

	rl := &RateLimiter{
		tokens:     float64(burst),
		maxTokens:  float64(burst),
		rate:       rate,
		lastUpdate: time.Now(),
		burstSize:  burst,
		perMessage: perMessage,
	}

	if !perMessage {
		// 按字节计数，每个令牌对应1字节
		rl.bytesPerToken = 1.0
	}

	return rl
}

// NewBytesRateLimiter 创建字节速率限制器
// rate: 字节/秒
// burst: 突发字节数
func NewBytesRateLimiter(rate float64, burst int) *RateLimiter {
	return NewRateLimiter(rate, burst, false)
}

// Wait 等待直到有足够的令牌可用
// 如果限制器为nil（不限制），则立即返回
func (rl *RateLimiter) Wait(ctx context.Context) error {
	if rl == nil {
		return nil
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// 更新令牌数
	rl.updateTokens()

	// 如果令牌足够，消耗一个令牌
	if rl.tokens >= 1.0 {
		rl.tokens -= 1.0
		return nil
	}

	// 计算需要等待的时间
	needed := 1.0 - rl.tokens
	waitTime := time.Duration(float64(time.Second) * needed / rl.rate)

	// 等待直到有令牌可用
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitTime):
		// 更新令牌数
		rl.updateTokens()
		if rl.tokens >= 1.0 {
			rl.tokens -= 1.0
		}
		return nil
	}
}

// WaitN 等待直到有N个令牌可用
func (rl *RateLimiter) WaitN(ctx context.Context, n int) error {
	if rl == nil {
		return nil
	}

	if n <= 0 {
		return nil
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// 更新令牌数
	rl.updateTokens()

	// 如果令牌足够，消耗N个令牌
	if rl.tokens >= float64(n) {
		rl.tokens -= float64(n)
		return nil
	}

	// 计算需要等待的时间
	needed := float64(n) - rl.tokens
	waitTime := time.Duration(float64(time.Second) * needed / rl.rate)

	// 等待直到有足够的令牌
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitTime):
		// 更新令牌数
		rl.updateTokens()
		if rl.tokens >= float64(n) {
			rl.tokens -= float64(n)
		}
		return nil
	}
}

// WaitBytes 等待直到有足够的字节令牌可用
func (rl *RateLimiter) WaitBytes(ctx context.Context, bytes int) error {
	if rl == nil {
		return nil
	}

	if bytes <= 0 {
		return nil
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// 更新令牌数
	rl.updateTokens()

	// 计算需要的令牌数
	neededTokens := float64(bytes) * rl.bytesPerToken

	// 如果令牌足够，消耗相应数量的令牌
	if rl.tokens >= neededTokens {
		rl.tokens -= neededTokens
		return nil
	}

	// 计算需要等待的时间
	needed := neededTokens - rl.tokens
	waitTime := time.Duration(float64(time.Second) * needed / rl.rate)

	// 等待直到有足够的令牌
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitTime):
		// 更新令牌数
		rl.updateTokens()
		if rl.tokens >= neededTokens {
			rl.tokens -= neededTokens
		}
		return nil
	}
}

// Allow 检查是否有足够的令牌（不等待）
func (rl *RateLimiter) Allow() bool {
	if rl == nil {
		return true
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.updateTokens()

	if rl.tokens >= 1.0 {
		rl.tokens -= 1.0
		return true
	}

	return false
}

// AllowN 检查是否有N个令牌可用（不等待）
func (rl *RateLimiter) AllowN(n int) bool {
	if rl == nil {
		return true
	}

	if n <= 0 {
		return true
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.updateTokens()

	if rl.tokens >= float64(n) {
		rl.tokens -= float64(n)
		return true
	}

	return false
}

// AllowBytes 检查是否有足够的字节令牌可用（不等待）
func (rl *RateLimiter) AllowBytes(bytes int) bool {
	if rl == nil {
		return true
	}

	if bytes <= 0 {
		return true
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.updateTokens()

	neededTokens := float64(bytes) * rl.bytesPerToken
	if rl.tokens >= neededTokens {
		rl.tokens -= neededTokens
		return true
	}

	return false
}

// updateTokens 更新令牌数（需要先获取锁）
func (rl *RateLimiter) updateTokens() {
	now := time.Now()
	elapsed := now.Sub(rl.lastUpdate).Seconds()

	// 计算新增的令牌数
	newTokens := elapsed * rl.rate

	// 更新令牌数（不超过最大值）
	rl.tokens = min(rl.tokens+newTokens, rl.maxTokens)
	rl.lastUpdate = now
}

// SetRate 动态设置速率
func (rl *RateLimiter) SetRate(rate float64) {
	if rl == nil {
		return
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.updateTokens()
	rl.rate = rate
}

// GetRate 获取当前速率
func (rl *RateLimiter) GetRate() float64 {
	if rl == nil {
		return 0
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	return rl.rate
}

// min 返回两个float64中的较小值
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
