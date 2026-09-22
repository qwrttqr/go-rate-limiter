package algos

import (
	"context"
	"fmt"
	"net/http"
	"qwrttqr-rate-limiter/core/server/internal/cache"
	"qwrttqr-rate-limiter/core/server/internal/config"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type RollingWindowStore interface {
	CheckAndIncrement(ctx context.Context, key string, windowStart, windowSize, currentTime, maxRequests int64) (bool, int64, error)
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
	body, err := ReadIncomingBody(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	currentTime := rwl.Now()
	windowStart := currentTime - rwl.WindowSize
	allowed, retryAfter, err := rwl.Store.CheckAndIncrement(r.Context(), body.ClientKey, windowStart, rwl.WindowSize, currentTime, rwl.MaxRequests)
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

func (s *InMemoryRollingWindowStore) CheckAndIncrement(ctx context.Context, key string, windowStart, windowSize, currentTime, maxRequests int64) (bool, int64, error) {
	val := s.Cache.LoadOrStore(key, &RollingWindowState{
		Timestamps: make([]int64, 0, maxRequests*2),
	})
	window := val.(*RollingWindowState)

	window.mu.Lock()
	defer window.mu.Unlock()

	i := 0
	for ; i < len(window.Timestamps); i++ {
		if window.Timestamps[i] > windowStart {
			break
		}
	}
	window.Timestamps = window.Timestamps[i:]

	if int64(len(window.Timestamps)) < maxRequests {
		window.Timestamps = append(window.Timestamps, currentTime)
		return true, 0, nil
	}
	retryAfter := window.Timestamps[0] - windowStart

	return false, retryAfter, nil
}

type RedisRollingWindowStore struct {
	Client *redis.Client
}

var rollingWindowScript = redis.NewScript(`
local windowStart = tonumber(ARGV[1])
local windowSize = tonumber(ARGV[2])
local currentTime = tonumber(ARGV[3])
local maxRequests = tonumber(ARGV[4])

redis.call("ZREMRANGEBYSCORE", KEYS[1], "-inf", windowStart)
local count = redis.call("ZCARD", KEYS[1])

if count < maxRequests then
    redis.call("ZADD", KEYS[1], currentTime, currentTime)
    redis.call("EXPIRE", KEYS[1], windowSize)
    return {1, 0}
else
	local oldest = redis.call("ZRANGE", KEYS[1], 0, 0, "WITHSCORES")
	local retryAfter = tonumber(oldest[2]) - windowStart
    return {0, retryAfter}
end
`)

func (s *RedisRollingWindowStore) CheckAndIncrement(ctx context.Context, key string, windowStart, windowSize, currentTime, maxRequests int64) (bool, int64, error) {
	res, err := rollingWindowScript.Run(ctx, s.Client, []string{key}, windowStart, windowSize, currentTime, maxRequests).Int64Slice()
	if err != nil {
		return false, 0, err
	}
	allowed := res[0] == 1
	retryAfter := res[1]
	return allowed, retryAfter, nil
}
