package algos

import (
	"context"
	"fmt"
	"log"
	"qwrttqr-rate-limiter/core/config"
	"qwrttqr-rate-limiter/core/server/cache"
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

type InMemoryRollingWindowStore struct {
	Backend cache.InMemoryStoreInterface
}

type RedisRollingWindowStore struct {
	Backend cache.RedisStoreInterface
}

func NewRollingWindowStore(store string, cacheInstance cache.InMemoryStoreInterface, redisClient *redis.Client, redisDefaultKeyTTL int64) (RollingWindowStore, error) {
	switch store {
	case "in_memory":
		return &InMemoryRollingWindowStore{Backend: cacheInstance}, nil
	case "redis":
		return &RedisRollingWindowStore{Backend: cache.NewRedisStore(redisClient, redisDefaultKeyTTL)}, nil
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
		log.Printf("rate limiter error while allow %s", err)
		return Decision{}, err
	}
	return Decision{Allowed: allowed, RetryAfter: retryAfter}, nil
}

type RollingWindowState struct {
	Timestamps []int64
}

func (rws *RollingWindowState) check(
	windowSize,
	maxRequests,
	now int64,
) (bool, int64) {
	windowStart := now - windowSize
	i := 0
	for ; i < len(rws.Timestamps); i++ {
		if rws.Timestamps[i] > windowStart {
			break
		}
	}
	if i > 0 {
		copy(rws.Timestamps, rws.Timestamps[i:])
		rws.Timestamps = rws.Timestamps[:len(rws.Timestamps)-i]
	}
	if int64(len(rws.Timestamps)) < maxRequests {
		rws.Timestamps = append(rws.Timestamps, now)
		return true, 0
	}
	retryAfter := rws.Timestamps[0] - windowStart
	return false, retryAfter
}

func (store *InMemoryRollingWindowStore) CheckAndIncrement(
	_ context.Context,
	key string,
	windowSize,
	maxRequests,
	now int64,
) (allowed bool, retryAfter int64, err error) {
	err = store.Backend.Update(
		key,
		func() any { return &RollingWindowState{Timestamps: make([]int64, 0, 32)} },
		func(v any) error {
			st, ok := v.(*RollingWindowState)
			if !ok {
				return fmt.Errorf("unexpected state type %T for key %q", v, key)
			}
			allowed, retryAfter = st.check(windowSize, maxRequests, now)
			return nil
		})
	return allowed, retryAfter, err
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
	return s.Backend.RunLuaScript(ctx, rollingWindowScript, []string{key}, requestId, windowSize, maxRequests, now)
}
