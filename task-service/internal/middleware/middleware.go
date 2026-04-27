package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/yourname/ai-learning-platform/task-service/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// ── Logger 中间件 ────────────────────────────────────────────

func Logger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("trace_id", c.GetString("trace_id")),
		)
	}
}

// ── Tracing 中间件（注入 Trace-ID） ──────────────────────────

func Tracing() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = generateTraceID()
		}
		c.Set("trace_id", traceID)
		c.Header("X-Trace-ID", traceID)
		c.Next()
	}
}

func generateTraceID() string {
	return time.Now().Format("20060102150405.000000")
}

// ── JWT 中间件 ───────────────────────────────────────────────

const jwtSecret = "your-secret-key" // TODO: 从配置读取

func JWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			response.Fail(c, http.StatusUnauthorized, "missing token")
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		userID, err := parseJWT(tokenStr)
		if err != nil {
			response.Fail(c, http.StatusUnauthorized, "invalid token")
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

func parseJWT(tokenStr string) (uint64, error) {
	// TODO: 使用 github.com/golang-jwt/jwt/v5 解析
	// 这里先返回 mock 值
	_ = tokenStr
	return 1, nil
}

// ── 令牌桶限流中间件 ─────────────────────────────────────────
//
// 面试重点：手写令牌桶，不依赖第三方限流库
// 原理：固定速率往桶里放令牌，请求消耗令牌，桶空则拒绝
//
// 这里实现分布式令牌桶（Redis），支持多实例部署

const rateLimitLuaScript = `
local key = KEYS[1]
local capacity = tonumber(ARGV[1])   -- 桶容量
local rate = tonumber(ARGV[2])       -- 每秒放入速率
local now = tonumber(ARGV[3])        -- 当前时间戳(ms)
local requested = tonumber(ARGV[4])  -- 请求消耗令牌数

local last_tokens = tonumber(redis.call("HGET", key, "tokens") or capacity)
local last_time = tonumber(redis.call("HGET", key, "last_time") or now)

-- 计算新增令牌
local elapsed = math.max(0, now - last_time)
local new_tokens = math.min(capacity, last_tokens + (elapsed / 1000.0) * rate)

local allowed = 0
if new_tokens >= requested then
    new_tokens = new_tokens - requested
    allowed = 1
end

redis.call("HSET", key, "tokens", new_tokens, "last_time", now)
redis.call("EXPIRE", key, 60)

return allowed
`

type TokenBucket struct {
	rdb      *redis.Client
	capacity int64
	rate     int64 // tokens per second
}

func newTokenBucket(rdb *redis.Client, capacity, rate int64) *TokenBucket {
	return &TokenBucket{rdb: rdb, capacity: capacity, rate: rate}
}

func (tb *TokenBucket) Allow(ctx context.Context, key string) bool {
	now := time.Now().UnixMilli()
	result, err := tb.rdb.Eval(ctx, rateLimitLuaScript,
		[]string{"rate_limit:" + key},
		tb.capacity, tb.rate, now, 1,
	).Int64()
	if err != nil {
		// Redis 异常时放行，避免影响正常请求
		return true
	}
	return result == 1
}

func RateLimit(rdb *redis.Client) gin.HandlerFunc {
	// 每个用户每秒最多 10 个请求，桶容量 20
	bucket := newTokenBucket(rdb, 20, 10)

	return func(c *gin.Context) {
		// 按 IP 限流（未登录），按 UserID 限流（已登录）
		key := c.ClientIP()
		if userID, exists := c.Get("user_id"); exists {
			key = "user:" + string(rune(userID.(uint64)))
		}

		if !bucket.Allow(c.Request.Context(), key) {
			response.Fail(c, http.StatusTooManyRequests, "rate limit exceeded")
			c.Abort()
			return
		}
		c.Next()
	}
}
