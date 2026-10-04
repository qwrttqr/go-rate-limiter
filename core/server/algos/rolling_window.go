package algos

import (
	"context"
	"fmt"
	"net/http"
	"qwrttqr-rate-limiter/core/server/cache"
	"qwrttqr-rate-limiter/core/server/internal/config"
	"strconv"
	"sync"
	"time"

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

func ValidateRollingWindowConfiguration(cfg config.Configuration) error {
	if *cfg.AlgoSettings.MaxRequests <= 0 {
		return fmt.Errorf("the max requests param should be strictly greater then 0")
	}
	return CheckRequiredFields(cfg.AlgoSettings, []string{"window_size", "max_requests"})
}

func NewRollingWindowStore(cfg config.Configuration, cacheInstance cache.Cache, redisClient *redis.Client) (RollingWindowStore, error) {
	switch cfg.Store {
	case "in_memory":
		return &InMemoryRollingWindowStore{Cache: cacheInstance}, nil
	case "redis":
		return &RedisRollingWindowStore{Client: redisClient}, nil
	default:
		return nil, fmt.Errorf("unsupported store: %s", cfg.Store)
	}
}

func (rwl *RollingWindowLimiter) LimitHTTP(w http.ResponseWriter, r *http.Request) {
	parsedHeaders, err := ReadIncomingHeader(r)
	if err != nil {
		http.Error(w, "rate limiter error", http.StatusInternalServerError)
		return
	}

	now := rwl.Now()

	allowed, retryAfter, err := rwl.Store.CheckAndIncrement(r.Context(), parsedHeaders.ClientKey, rwl.WindowSize, rwl.MaxRequests, now)
	if err != nil {
		http.Error(w, "rate limiter error", http.StatusInternalServerError)
		return
	}
	if !allowed {
		w.Header().Set("Retry-After", strconv.FormatInt(retryAfter, 10))
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}
}

type RollingWindowState struct {
	mu         sync.Mutex
	Timestamps []int64
}

type InMemoryRollingWindowStore struct {
	Cache cache.Cache
}

func (s *InMemoryRollingWindowStore) CheckAndIncrement(
	_ context.Context,
	key string,
	windowSize,
	maxRequests,
	now int64) (bool, int64, error) {

	val, err := s.Cache.Get(key)
	if err != nil {
		val = s.Cache.LoadOrStore(key, func() any { return &RollingWindowState{Timestamps: make([]int64, 0, 8)} })
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

type RedisRollingWindowStore struct {
	Client *redis.Client
}

var rollingWindowScript = redis.NewScript(`
local windowSize = tonumber(ARGV[1])
local now = tonumber(ARGV[2])
local maxRequests = tonumber(ARGV[3])

local windowStart = now - windowSize

redis.call("ZREMRANGEBYSCORE", KEYS[1], "-inf", windowStart)
local count = redis.call("ZCARD", KEYS[1])

if count < maxRequests then
    redis.call("ZADD", KEYS[1], now, now)
    redis.call("EXPIRE", KEYS[1], windowSize)
    return {1, 0}
else
	local oldest = redis.call("ZRANGE", KEYS[1], 0, 0, "WITHSCORES")
	local retryAfter = tonumber(oldest[2]) - windowStart
    return {0, retryAfter}
end
`)

func (s *RedisRollingWindowStore) CheckAndIncrement(ctx context.Context, key string, windowSize, maxRequests, now int64) (bool, int64, error) {
	res, err := rollingWindowScript.Run(ctx, s.Client, []string{key}, windowSize, maxRequests, now).Int64Slice()
	if err != nil {
		return false, 0, err
	}
	allowed := res[0] == 1
	retryAfter := res[1]
	return allowed, retryAfter, nil
}
