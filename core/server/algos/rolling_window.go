package algos

import (
	"context"
	"fmt"
	"qwrttqr-rate-limiter/core/config"
	"qwrttqr-rate-limiter/core/server/cache"
	"sync"
	"time"
	"uuid"

	"github.com/redis/go-redis/v9"
)

type RollingWindowStore interface {
	CheckAndIncrement(ctx context.Context, key string, windowSize, maxRequests, now int64) (bool, int64, error)
}
type RollingWindowLimiter struct {
	WindowSize  int64
	MaxRequests int64
	Store       RollingWindowStore
	Now         func() int64
}

func (rwl *RollingWindowLimiter) Configure() {
	if rwl.Now == nil {
		rwl.Now = func() int64 {
			return time.Now().Unix()
		}
	}
}

func ValidateRollingWindowConfig(cfg config.AlgoSettings) error {
	if *cfg.MaxRequests <= 0 {
		return fmt.Errorf("the max requests param should be strictly greater then 0")
	}
	return CheckRequiredFields(cfg, []string{"window_size", "max_requests"})
}

func NewRollingWindowStore(store string, cacheInstance cache.InMemoryStoreInterface, redisClient *redis.Client, redisDefaultKeyTTL int64) (RollingWindowStore, error) {
	switch store {
	case "in_memory":
		return &InMemoryRollingWindowStore{Store: cacheInstance}, nil
	case "redis":
		return &RedisRollingWindowStore{Store: cache.NewRedisStore(redisClient, redisDefaultKeyTTL)}, nil
	default:
		return nil, fmt.Errorf("unsupported store: %s", store)
	}
}

func (rwl *RollingWindowLimiter) Allow(ctx context.Context, req Request) (Decision, error) {
	allowed, retryAfter, err := rwl.Store.CheckAndIncrement(
		ctx,
		req.Key,
		rwl.WindowSize,
		rwl.MaxRequests,
		rwl.Now(),
	)
	if err != nil {
		return Decision{}, err
	}
	return Decision{Allowed: allowed, RetryAfter: retryAfter}, nil
}

type RollingWindowState struct {
	mu         sync.Mutex
	Timestamps []int64
}

type InMemoryRollingWindowStore struct {
	Store cache.InMemoryStoreInterface
}

type RedisRollingWindowStore struct {
	Store cache.RedisStoreInterface
}

func (s *InMemoryRollingWindowStore) CheckAndIncrement(
	_ context.Context,
	key string,
	windowSize,
	maxRequests,
	now int64) (bool, int64, error) {

	val, err := s.Store.Get(key)
	if err != nil {
		val = s.Store.LoadOrStore(key, func() any { return &RollingWindowState{Timestamps: make([]int64, 0, 8)} })
	}
	windowStart := now - windowSize
	window := val.(*RollingWindowState)

	window.mu.Lock()
	defer window.mu.Unlock()

	i := 0
	for ; i < len(window.Timestamps); i++ {
		if window.Timestamps[i] > windowStart {
			break
		}
	}
	if i > 0 {
		copy(window.Timestamps, window.Timestamps[i:])
		window.Timestamps = window.Timestamps[:len(window.Timestamps)-i]
	}
	if int64(len(window.Timestamps)) < maxRequests {
		window.Timestamps = append(window.Timestamps, now)
		return true, 0, nil
	}
	retryAfter := window.Timestamps[0] - windowStart

	return false, retryAfter, nil
}

var rollingWindowScript = redis.NewScript(`
local requestId = ARGV[1]
local windowSize = tonumber(ARGV[2])
local maxRequests = tonumber(ARGV[3])
local now = tonumber(ARGV[4])

local windowStart = now - windowSize

redis.call("ZREMRANGEBYSCORE", KEYS[1], "-inf", windowStart)
local count = redis.call("ZCARD", KEYS[1])

if count < maxRequests then
    redis.call("ZADD", KEYS[1], now, requestId)
    redis.call("EXPIRE", KEYS[1], windowSize)
    return {1, 0}
else
	local oldest = redis.call("ZRANGE", KEYS[1], 0, 0, "WITHSCORES")
	local retryAfter = 0
	if oldest and oldest[2] then
		retryAfter = tonumber(oldest[2]) - windowStart
	end
    return {0, retryAfter}
end
`)

func (s *RedisRollingWindowStore) CheckAndIncrement(ctx context.Context, key string, windowSize, maxRequests, now int64) (bool, int64, error) {
	requestId := uuid.New().String()
	return s.Store.RunLuaScript(ctx, rollingWindowScript, []string{key}, requestId, windowSize, maxRequests, now)
}
