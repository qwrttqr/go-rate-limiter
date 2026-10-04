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

type FixedWindowStore interface {
	CheckAndIncrement(ctx context.Context, key string, windowSize, maxRequests, now int64) (bool, int64, error)
}
type FixedWindowLimiter struct {
	WindowSize  int64
	MaxRequests int64
	Store       FixedWindowStore
	Now         func() int64
}

func (fwl *FixedWindowLimiter) Configure() {
	if fwl.Now == nil {
		fwl.Now = func() int64 {
			return time.Now().Unix()
		}
	}
}

func ValidateFixedWindowConfig(cfg config.Configuration) error {
	if err := CheckRequiredFields(cfg.AlgoSettings, []string{"window_size", "max_requests"}); err != nil {
		return err
	}
	if *cfg.AlgoSettings.MaxRequests <= 0 {
		return fmt.Errorf("max_requests must be greater than 0")
	}
	if *cfg.AlgoSettings.WindowSize <= 0 { // assuming WindowSize is a pointer like MaxRequests
		return fmt.Errorf("window_size must be greater than 0")
	}
	return nil
}

func NewFixedWindowStore(cfg config.Configuration, cacheInstance cache.Cache, redisClient *redis.Client) (FixedWindowStore, error) {
	switch cfg.Store {
	case "in_memory":
		return &InMemoryFixedWindowStore{Cache: cacheInstance}, nil
	case "redis":
		return &RedisFixedWindowStore{Client: redisClient}, nil
	default:
		return nil, fmt.Errorf("unsupported store: %s", cfg.Store)
	}
}

func (fwl *FixedWindowLimiter) LimitHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := ReadIncomingHeader(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	now := fwl.Now()

	allowed, retryAfter, err := fwl.Store.CheckAndIncrement(r.Context(), body.ClientKey, fwl.WindowSize, fwl.MaxRequests, now)
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

type InMemoryFixedWindowStore struct {
	Cache cache.Cache
}
type FixedWindowState struct {
	mu           sync.Mutex
	Count        int64
	StoredWindow int64
}

func (s *InMemoryFixedWindowStore) CheckAndIncrement(
	_ context.Context, key string,
	windowSize,
	maxRequests,
	now int64,
) (bool, int64, error) {
	windowStart := (now / windowSize) * windowSize

	val, err := s.Cache.Get(key)
	if err != nil {
		val = s.Cache.LoadOrStore(key, func() any { return &FixedWindowState{StoredWindow: windowStart} })
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

	timeToNextWindow := (window.StoredWindow + windowSize) - now
	if timeToNextWindow < 0 {
		timeToNextWindow = 0
	}

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
