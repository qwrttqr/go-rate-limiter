package algos

import (
	"context"
	"fmt"
	"qwrttqr-rate-limiter/core/config"
	"qwrttqr-rate-limiter/core/server/cache"
	"sync"

	"github.com/redis/go-redis/v9"
)

type FixedWindowStore interface {
	CheckAndIncrement(ctx context.Context, key string, windowSize, maxRequests, now int64) (bool, int64, error)
}
type FixedWindowLimiter struct {
	WindowSize  int64
	MaxRequests int64
	Store       FixedWindowStore
	Now         func() int64
}

func ValidateFixedWindowConfig(cfg config.AlgoSettings) error {
	if err := CheckRequiredFields(cfg, []string{"window_size", "max_requests"}); err != nil {
		return err
	}
	if *cfg.MaxRequests <= 0 {
		return fmt.Errorf("max_requests must be greater than 0")
	}
	if *cfg.WindowSize <= 0 {
		return fmt.Errorf("window_size must be greater than 0")
	}
	return nil
}

func NewFixedWindowStore(store string, cacheInstance cache.InMemoryStoreInterface, redisClient *redis.Client, redisDefaultKeyTTL int64) (FixedWindowStore, error) {
	switch store {
	case "in_memory":
		return &InMemoryFixedWindowStore{Store: cacheInstance}, nil
	case "redis":
		return &RedisFixedWindowStore{Store: cache.NewRedisStore(redisClient, redisDefaultKeyTTL)}, nil
	default:
		return nil, fmt.Errorf("unsupported store: %s", store)
	}
}

func (fwl *FixedWindowLimiter) Allow(ctx context.Context, req Request) (Decision, error) {
	allowed, retryAfter, err := fwl.Store.CheckAndIncrement(
		ctx, req.Key, fwl.WindowSize, fwl.MaxRequests, fwl.Now())
	if err != nil {
		return Decision{}, err
	}
	return Decision{Allowed: allowed, RetryAfter: retryAfter}, nil
}

type InMemoryFixedWindowStore struct {
	Store cache.InMemoryStoreInterface
}

type RedisFixedWindowStore struct {
	Store cache.RedisStoreInterface
}
type FixedWindowState struct {
	mu           sync.Mutex
	Count        int64
	StoredWindow int64
}

func (store *InMemoryFixedWindowStore) CheckAndIncrement(
	_ context.Context, key string,
	windowSize,
	maxRequests,
	now int64,
) (bool, int64, error) {
	windowStart := (now / windowSize) * windowSize

	val, err := store.Store.Get(key)
	if err != nil {
		val = store.Store.LoadOrStore(key, func() any { return &FixedWindowState{StoredWindow: windowStart} })
	}
	window := val.(*FixedWindowState)

	window.mu.Lock()
	defer window.mu.Unlock()

	if window.StoredWindow < windowStart {
		window.Count = 0
		window.StoredWindow = windowStart
	}
	if window.Count < maxRequests {
		window.Count++
		return true, 0, nil
	}

	timeToNextWindow := max((window.StoredWindow+windowSize)-now, 0)

	return false, timeToNextWindow, nil
}

var fixedWindowScript = redis.NewScript(`
local stored = redis.call("HMGET", KEYS[1], "window", "count")
local storedWindow = tonumber(stored[1])
local count = tonumber(stored[2]) or 0
local windowSize = tonumber(ARGV[1])
local maxRequests = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

local windowStart = math.floor(now / windowSize) * windowSize

if storedWindow == nil or storedWindow < windowStart then
    count = 0
    storedWindow = windowStart
end

if count < maxRequests then
    redis.call("HSET", KEYS[1], "window", storedWindow, "count", count + 1)
    redis.call("EXPIRE", KEYS[1], windowSize)
    return {1, 0}
end
return {0, windowStart + windowSize - now}
`)

func (s *RedisFixedWindowStore) CheckAndIncrement(ctx context.Context, key string, windowSize, maxRequests, now int64) (bool, int64, error) {
	return s.Store.RunLuaScript(ctx, fixedWindowScript, []string{key}, windowSize, maxRequests, now)
}
