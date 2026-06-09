package auth

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimitEntry struct {
	count    int
	resetAt  time.Time
}

type rateLimiter struct {
	mu       sync.Mutex
	entries  map[string]*rateLimitEntry
	limit    int
	window   time.Duration
}

var (
	authLimiter   = newRateLimiter(5, time.Minute)    // 登录/重置等：5次/分钟
	generalLimiter = newRateLimiter(100, time.Minute) // 通用API：100次/分钟
)

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		entries: make(map[string]*rateLimitEntry),
		limit:   limit,
		window:  window,
	}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	entry, exists := rl.entries[ip]
	if !exists || now.After(entry.resetAt) {
		rl.entries[ip] = &rateLimitEntry{count: 1, resetAt: now.Add(rl.window)}
		return true
	}

	if entry.count >= rl.limit {
		return false
	}
	entry.count++
	return true
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, entry := range rl.entries {
			if now.After(entry.resetAt) {
				delete(rl.entries, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimitAuth 对认证接口（登录、忘记密码、重置密码）的严格限流
func RateLimitAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !authLimiter.allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "请求过于频繁，请稍后再试"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RateLimitGeneral 通用 API 限流
func RateLimitGeneral() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !generalLimiter.allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "请求过于频繁，请稍后再试"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RateLimitAdmin 管理接口限流（比通用稍宽松，但仍需防护）
func RateLimitAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !generalLimiter.allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "请求过于频繁，请稍后再试"})
			c.Abort()
			return
		}
		c.Next()
	}
}
