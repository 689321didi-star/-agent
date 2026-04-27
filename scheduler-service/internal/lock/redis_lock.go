package lock

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var ErrLockFailed = errors.New("failed to acquire lock")

// RedisLock 基于 Redis 的分布式锁
// ─────────────────────────────────────────────────────────
// 面试考点全在这里：
//   1. 为什么用 SETNX + EXPIRE 不够安全？
//      → 两条命令非原子，中间宕机会死锁
//      → 解决：SET key value NX EX seconds（原子操作）
//
//   2. value 为什么要用随机 UUID？
//      → 防止释放别人的锁（自己超时后别人拿到锁，自己再来删就删错了）
//      → 解锁用 Lua 脚本保证"查+删"原子性
//
//   3. 锁过期但任务还没完成怎么办？
//      → watchdog：后台 goroutine 每隔 1/3 过期时间续期
//      → 任务完成后主动停止 watchdog

const unlockScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
else
    return 0
end
`

const renewScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("EXPIRE", KEYS[1], ARGV[2])
else
    return 0
end
`

type RedisLock struct {
	rdb        *redis.Client
	key        string
	value      string        // 随机 UUID，用于防止误删
	expiry     time.Duration
	stopRenew  chan struct{}  // 停止 watchdog
}

func NewRedisLock(rdb *redis.Client, key string, expiry time.Duration) *RedisLock {
	return &RedisLock{
		rdb:    rdb,
		key:    "lock:" + key,
		value:  uuid.New().String(),
		expiry: expiry,
	}
}

// Lock 加锁，带重试
func (l *RedisLock) Lock(ctx context.Context) error {
	return l.LockWithRetry(ctx, 1, 0)
}

// LockWithRetry 加锁（可重试）
func (l *RedisLock) LockWithRetry(ctx context.Context, maxRetries int, retryInterval time.Duration) error {
	for i := 0; i <= maxRetries; i++ {
		ok, err := l.rdb.SetNX(ctx, l.key, l.value, l.expiry).Result()
		if err != nil {
			return fmt.Errorf("redis setnx: %w", err)
		}
		if ok {
			// 加锁成功，启动 watchdog
			l.startWatchdog()
			return nil
		}

		if i < maxRetries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryInterval):
			}
		}
	}
	return ErrLockFailed
}

// Unlock 解锁（Lua 脚本保证原子性）
func (l *RedisLock) Unlock(ctx context.Context) error {
	// 停止 watchdog
	l.stopWatchdog()

	result, err := l.rdb.Eval(ctx, unlockScript, []string{l.key}, l.value).Int64()
	if err != nil {
		return fmt.Errorf("unlock: %w", err)
	}
	if result == 0 {
		return fmt.Errorf("lock already expired or owned by others")
	}
	return nil
}

// startWatchdog 启动自动续期 goroutine
func (l *RedisLock) startWatchdog() {
	l.stopRenew = make(chan struct{})
	renewInterval := l.expiry / 3 // 每 1/3 过期时间续期一次

	go func() {
		ticker := time.NewTicker(renewInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				l.rdb.Eval(ctx, renewScript,
					[]string{l.key}, l.value, int(l.expiry.Seconds()),
				)
				cancel()
			case <-l.stopRenew:
				return
			}
		}
	}()
}

func (l *RedisLock) stopWatchdog() {
	if l.stopRenew != nil {
		close(l.stopRenew)
	}
}
