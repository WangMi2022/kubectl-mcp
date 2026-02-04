package server

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrRateLimitExceeded 限流错误
var ErrRateLimitExceeded = errors.New("请求过于频繁，请稍后重试")

// ErrConcurrencyLimitExceeded 并发限制错误
var ErrConcurrencyLimitExceeded = errors.New("服务器繁忙，请稍后重试")

// RateLimiter 限流器接口
type RateLimiter interface {
	// Allow 检查是否允许请求
	Allow() bool
	// Wait 等待直到允许请求或上下文取消
	Wait(ctx context.Context) error
	// Close 关闭限流器
	Close()
}

// TokenBucketLimiter 令牌桶限流器
// 实现平滑的请求限流
type TokenBucketLimiter struct {
	rate       float64   // 每秒生成的令牌数
	capacity   float64   // 桶容量
	tokens     float64   // 当前令牌数
	lastUpdate time.Time // 上次更新时间
	mu         sync.Mutex
	closed     bool
}

// NewTokenBucketLimiter 创建令牌桶限流器
// 参数:
//   - rate: 每秒允许的请求数
//   - burst: 突发容量（桶大小）
func NewTokenBucketLimiter(rate float64, burst int) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		rate:       rate,
		capacity:   float64(burst),
		tokens:     float64(burst), // 初始满桶
		lastUpdate: time.Now(),
	}
}

// Allow 检查是否允许请求（非阻塞）
func (l *TokenBucketLimiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return false
	}

	l.refill()

	if l.tokens >= 1 {
		l.tokens--
		return true
	}
	return false
}

// Wait 等待直到允许请求或上下文取消
func (l *TokenBucketLimiter) Wait(ctx context.Context) error {
	for {
		l.mu.Lock()
		if l.closed {
			l.mu.Unlock()
			return ErrRateLimitExceeded
		}

		l.refill()

		if l.tokens >= 1 {
			l.tokens--
			l.mu.Unlock()
			return nil
		}

		// 计算需要等待的时间
		waitTime := time.Duration((1 - l.tokens) / l.rate * float64(time.Second))
		l.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// 继续尝试
		}
	}
}

// refill 补充令牌（必须在持有锁的情况下调用）
func (l *TokenBucketLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(l.lastUpdate).Seconds()
	l.tokens += elapsed * l.rate
	if l.tokens > l.capacity {
		l.tokens = l.capacity
	}
	l.lastUpdate = now
}

// Close 关闭限流器
func (l *TokenBucketLimiter) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.closed = true
}

// ConcurrencyLimiter 并发限制器
// 限制同时处理的请求数量
type ConcurrencyLimiter struct {
	maxConcurrent int
	semaphore     chan struct{}
	current       int64
	mu            sync.Mutex
}

// NewConcurrencyLimiter 创建并发限制器
// 参数:
//   - maxConcurrent: 最大并发数
func NewConcurrencyLimiter(maxConcurrent int) *ConcurrencyLimiter {
	if maxConcurrent <= 0 {
		maxConcurrent = 100 // 默认值
	}
	return &ConcurrencyLimiter{
		maxConcurrent: maxConcurrent,
		semaphore:     make(chan struct{}, maxConcurrent),
	}
}

// Acquire 获取许可（阻塞）
func (l *ConcurrencyLimiter) Acquire(ctx context.Context) error {
	select {
	case l.semaphore <- struct{}{}:
		l.mu.Lock()
		l.current++
		l.mu.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TryAcquire 尝试获取许可（非阻塞）
func (l *ConcurrencyLimiter) TryAcquire() bool {
	select {
	case l.semaphore <- struct{}{}:
		l.mu.Lock()
		l.current++
		l.mu.Unlock()
		return true
	default:
		return false
	}
}

// Release 释放许可
func (l *ConcurrencyLimiter) Release() {
	l.mu.Lock()
	if l.current > 0 {
		l.current--
	}
	l.mu.Unlock()

	select {
	case <-l.semaphore:
	default:
	}
}

// Current 获取当前并发数
func (l *ConcurrencyLimiter) Current() int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.current
}

// Available 获取可用许可数
func (l *ConcurrencyLimiter) Available() int {
	return l.maxConcurrent - len(l.semaphore)
}

// MaxConcurrent 获取最大并发数
func (l *ConcurrencyLimiter) MaxConcurrent() int {
	return l.maxConcurrent
}

// SlidingWindowLimiter 滑动窗口限流器
// 更精确的限流实现
type SlidingWindowLimiter struct {
	windowSize  time.Duration // 窗口大小
	maxRequests int           // 窗口内最大请求数
	requests    []time.Time   // 请求时间戳
	mu          sync.Mutex
	closed      bool
}

// NewSlidingWindowLimiter 创建滑动窗口限流器
// 参数:
//   - windowSize: 窗口大小
//   - maxRequests: 窗口内最大请求数
func NewSlidingWindowLimiter(windowSize time.Duration, maxRequests int) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		windowSize:  windowSize,
		maxRequests: maxRequests,
		requests:    make([]time.Time, 0, maxRequests),
	}
}

// Allow 检查是否允许请求
func (l *SlidingWindowLimiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return false
	}

	now := time.Now()
	windowStart := now.Add(-l.windowSize)

	// 清理过期的请求
	validRequests := make([]time.Time, 0, len(l.requests))
	for _, t := range l.requests {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}
	l.requests = validRequests

	// 检查是否超过限制
	if len(l.requests) >= l.maxRequests {
		return false
	}

	// 记录新请求
	l.requests = append(l.requests, now)
	return true
}

// Wait 等待直到允许请求或上下文取消
func (l *SlidingWindowLimiter) Wait(ctx context.Context) error {
	for {
		if l.Allow() {
			return nil
		}

		l.mu.Lock()
		if l.closed {
			l.mu.Unlock()
			return ErrRateLimitExceeded
		}

		// 计算需要等待的时间
		var waitTime time.Duration
		if len(l.requests) > 0 {
			oldest := l.requests[0]
			waitTime = oldest.Add(l.windowSize).Sub(time.Now())
			if waitTime < 0 {
				waitTime = time.Millisecond * 10
			}
		} else {
			waitTime = time.Millisecond * 10
		}
		l.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// 继续尝试
		}
	}
}

// Close 关闭限流器
func (l *SlidingWindowLimiter) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.closed = true
}

// CompositeRateLimiter 组合限流器
// 组合多个限流策略
type CompositeRateLimiter struct {
	rateLimiter        RateLimiter
	concurrencyLimiter *ConcurrencyLimiter
}

// NewCompositeRateLimiter 创建组合限流器
func NewCompositeRateLimiter(rateLimiter RateLimiter, concurrencyLimiter *ConcurrencyLimiter) *CompositeRateLimiter {
	return &CompositeRateLimiter{
		rateLimiter:        rateLimiter,
		concurrencyLimiter: concurrencyLimiter,
	}
}

// Acquire 获取许可
func (l *CompositeRateLimiter) Acquire(ctx context.Context) error {
	// 先检查速率限制
	if l.rateLimiter != nil {
		if err := l.rateLimiter.Wait(ctx); err != nil {
			return ErrRateLimitExceeded
		}
	}

	// 再检查并发限制
	if l.concurrencyLimiter != nil {
		if err := l.concurrencyLimiter.Acquire(ctx); err != nil {
			return ErrConcurrencyLimitExceeded
		}
	}

	return nil
}

// Release 释放许可
func (l *CompositeRateLimiter) Release() {
	if l.concurrencyLimiter != nil {
		l.concurrencyLimiter.Release()
	}
}

// Close 关闭限流器
func (l *CompositeRateLimiter) Close() {
	if l.rateLimiter != nil {
		l.rateLimiter.Close()
	}
}
