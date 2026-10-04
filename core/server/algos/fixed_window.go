package algos

import (
	"context"
	"fmt"
	"qwrttqr-rate-limiter/core/server/cache"
	"qwrttqr-rate-limiter/core/server/internal"
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

func ValidateFixedWindowConfig(cfg internal.Configuration) error {
	if err := CheckRequiredFields(cfg.AlgoSettings, []string{"window_size", "max_requests"}); err != nil {
		return err
	}
	if *cfg.AlgoSettings.MaxRequests <= 0 {
		return fmt.Errorf("max_requests must be greater than 0")
	}
	if *cfg.AlgoSettings.WindowSize <= 0 {
		return fmt.Errorf("window_size must be greater than 0")
	}
	return nil
}

func NewFixedWindowStore(cfg internal.Configuration, cacheInstance cache.Cache, redisClient *redis.Client) (FixedWindowStore, error) {
	switch cfg.Store {
	case "in_memory":
		return &InMemoryFixedWindowStore{Cache: cacheInstance}, nil
	case "redis":
		return &RedisFixedWindowStore{Client: redisClient}, nil
	default:
		return nil, fmt.Errorf("unsupported store: %s", cfg.Store)
	}
}

func (fwl *FixedWindowLimiter) Allow(ctx context.Context, req Request) (Decision, error) {
	if err := req.validate(); err != nil {
		return Decision{}, err
	}
	allowed, retryAfter, err := fwl.Store.CheckAndIncrement(
		ctx, req.Key, fwl.WindowSize, fwl.MaxRequests, fwl.Now())
	if err != nil {
		return Decision{}, err
	}
	return Decision{Allowed: allowed, RetryAfter: retryAfter}, nil
}

type InMemoryFixedWindowStore struct {
	Cache cache.Cache
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

	val, err := store.Cache.Get(key)
	if err != nil {
		val = store.Cache.LoadOrStore(key, func() any { return &FixedWindowState{StoredWindow: windowStart} })
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

type RedisFixedWindowStore struct {
	Client *redis.Client
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
	res, err := fixedWindowScript.Run(ctx, s.Client, []string{key}, windowSize, maxRequests, now).Int64Slice()
	if err != nil {
		return false, 0, err
	}
	allowed := res[0] == 1
	retryAfter := res[1]
	return allowed, retryAfter, nil
}
