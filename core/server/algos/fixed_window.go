package algos

import (
	"context"
	"fmt"
	"qwrttqr-rate-limiter/core/config"
	"qwrttqr-rate-limiter/core/server/cache"

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
		return &InMemoryFixedWindowStore{Backend: cacheInstance}, nil
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
	Backend cache.InMemoryStoreInterface
}

type RedisFixedWindowStore struct {
	Store cache.RedisStoreInterface
}
type FixedWindowState struct {
	Count        int64
	StoredWindow int64
}

func (fws *FixedWindowState) check(
	windowStart,
	windowSize,
	maxRequests,
	now int64,
) (bool, int64) {
	if fws.StoredWindow < windowStart {
		fws.Count = 0
		fws.StoredWindow = windowStart
	}
	if fws.Count < maxRequests {
		fws.Count++
		return true, 0
	}

	timeToNextWindow := max((fws.StoredWindow+windowSize)-now, 0)

	return false, timeToNextWindow
}

func (store *InMemoryFixedWindowStore) CheckAndIncrement(
	_ context.Context,
	key string,
	windowSize,
	maxRequests,
	now int64,
) (allowed bool, retryAfter int64, err error) {
	windowStart := (now / windowSize) * windowSize
	err = store.Backend.Update(
		key,
		func() any { return &FixedWindowState{StoredWindow: windowStart} },
		func(v any) error {
			st, ok := v.(*FixedWindowState)
			if !ok {
				return fmt.Errorf("unexpected state type %T for key %q", v, key)
			}
			allowed, retryAfter = st.check(windowStart, windowSize, maxRequests, now)
			return nil
		})
	return allowed, retryAfter, err
}

var fixedWindowScript = redis.NewScript(`
local stored = redis.call("HMGET", KEYS[1], "window", "count")
local storedWindow = stored[1] and tonumber(stored[1]) 
local count = stored[2] and tonumber(stored[2]) or 0
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
