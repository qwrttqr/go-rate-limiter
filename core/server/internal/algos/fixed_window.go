package algos

import (
	"context"
	"fmt"
	"net/http"
	"qwrttqr-rate-limiter/core/server/internal/config"
	"qwrttqr-rate-limiter/core/server/internal/interfaces"
	"qwrttqr-rate-limiter/core/server/internal/utils"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type FixedWindowStore interface {
	CheckAndIncrement(ctx context.Context, key string, windowStart, windowSize, maxRequests int64, now int64) (bool, int64, error)
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
	return utils.CheckRequiredFields(cfg.AlgoSettings, []string{"window_size", "max_requests"})
}

func NewFixedWindowStore(cfg config.Configuration, cacheInstance interfaces.Cache, redisClient *redis.Client) (FixedWindowStore, error) {
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
	body, err := utils.ReadIncomingBody(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	now := fwl.Now()
	currentWindow := now / fwl.WindowSize
	windowStart := currentWindow * fwl.WindowSize

	allowed, retryAfter, err := fwl.Store.CheckAndIncrement(r.Context(), body.ClientKey, windowStart, fwl.WindowSize, fwl.MaxRequests, now)
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
	Cache interfaces.Cache
}
type FixedWindowState struct {
	mu           sync.Mutex
	Count        int64
	StoredWindow int64
}

func (s *InMemoryFixedWindowStore) CheckAndIncrement(
	ctx context.Context, key string,
	windowStart,
	windowSize,
	maxRequests,
	now int64,
) (bool, int64, error) {

	val := s.Cache.LoadOrStore(key, &FixedWindowState{
		Count:        0,
		StoredWindow: windowStart,
	})
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
local windowStart = tonumber(ARGV[1])
local windowSize = tonumber(ARGV[2])
local maxRequests = tonumber(ARGV[3])
local now = tonumber(ARGV[4])

if storedWindow == nil or storedWindow < windowStart then
	count = 0
	storedWindow = windowStart
end
local retryAfter = (windowStart + windowSize) - now 
if count < maxRequests then
	count = count + 1
	redis.call("HMSET", KEYS[1], "window", storedWindow, "count", count)
	redis.call("EXPIRE", KEYS[1], windowSize)
	return {1, 0}
else 
	return {0, retryAfter}
end
`)

func (s *RedisFixedWindowStore) CheckAndIncrement(ctx context.Context, key string, windowStart, windowSize, maxRequests, now int64) (bool, int64, error) {
	res, err := fixedWindowScript.Run(ctx, s.Client, []string{key}, windowStart, windowSize, maxRequests, now).Int64Slice()
	if err != nil {
		return false, 0, err
	}
	allowed := res[0] == 1
	retryAfter := res[1]
	return allowed, retryAfter, nil
}
