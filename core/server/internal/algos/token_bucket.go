package algos

import (
	"context"
	"fmt"
	"net/http"
	"qwrttqr-rate-limiter/core/server/internal/config"
	"qwrttqr-rate-limiter/core/server/internal/interfaces"
	"qwrttqr-rate-limiter/core/server/internal/utils"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type TokenBucketStore interface {
	TakeToken(ctx context.Context, key string, rate float64, tokensRequired, now, capacity int64) (bool, error)
}
type TokenBucketLimiter struct {
	Capacity int64
	Rate     float64
	Store    TokenBucketStore
	Now      func() int64
}

func (tbl *TokenBucketLimiter) Configure() {
	if tbl.Now == nil {
		tbl.Now = func() int64 {
			return time.Now().Unix()
		}
	}
}

func ValidateTokenBucketConfig(cfg config.Configuration) error {
	return utils.CheckRequiredFields(cfg.AlgoSettings, []string{"capacity", "rate"})
}

func NewTokenBucketStore(cfg config.Configuration, cacheInstance interfaces.Cache, redisClient *redis.Client) (TokenBucketStore, error) {
	switch cfg.Store {
	case "in_memory":
		return &InMemoryBucketStore{Cache: cacheInstance}, nil
	case "redis":
		return &RedisBucketStore{Client: redisClient, DefaultTtl: cfg.Backends.Redis.DefaultTtl}, nil
	default:
		return nil, fmt.Errorf("unsupported store: %s", cfg.Store)
	}
}

func (tbl *TokenBucketLimiter) LimitHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := utils.ReadIncomingBody(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var requiredTokens int64 = 1
	if body.RequiredTokens != nil {
		requiredTokens = *body.RequiredTokens
	}
	allowed, err := tbl.Store.TakeToken(r.Context(), body.ClientKey, tbl.Rate, requiredTokens, tbl.Now(), tbl.Capacity)
	if err != nil {
		http.Error(w, "rate limiter error", http.StatusInternalServerError)
		return
	}
	if !allowed {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}
}

type InMemoryBucketStore struct {
	Cache interfaces.Cache
}

type BucketState struct {
	mu         sync.Mutex
	Tokens     int64
	LastRefill int64
}

func (s *InMemoryBucketStore) TakeToken(ctx context.Context, key string, rate float64, tokensRequired, now, capacity int64) (bool, error) {
	val := s.Cache.LoadOrStore(key, &BucketState{
		Tokens:     capacity,
		LastRefill: now,
	})

	bucket := val.(*BucketState)

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	tokens := bucket.Tokens
	lastRefill := bucket.LastRefill
	currentTime := now

	timePassed := currentTime - lastRefill
	refillTokens := float64(timePassed) * rate
	newTokens := min(capacity, tokens+int64(refillTokens))

	if newTokens >= tokensRequired {
		bucket.Tokens = newTokens - tokensRequired
		bucket.LastRefill = now
		return true, nil
	}

	return false, nil
}

type RedisBucketStore struct {
	Client     *redis.Client
	DefaultTtl int64
}

var tokenBucketScript = redis.NewScript(`
local capacity = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local tokensRequired = tonumber(ARGV[3])
local now = tonumber(ARGV[4])
local ttl = tonumber(ARGV[5])

local stored = redis.call("HMGET", KEYS[1], "tokens", "last_refill")
local tokens = tonumber(stored[1])
local lastRefill = tonumber(stored[2])

if tokens == nil then
	tokens = capacity
	lastRefill = now
end

local timePassed = now - lastRefill
local refill = timePassed * rate
local newTokens = math.min(capacity, tokens + refill)

if newTokens >= tokensRequired then
	newTokens = newTokens - tokensRequired
	redis.call("HMSET", KEYS[1], "tokens", newTokens, "last_refill", now)
	redis.call("EXPIRE", KEYS[1], ttl)
	return 1
else 
	return 0
end
`)

func (s *RedisBucketStore) TakeToken(ctx context.Context, key string, rate float64, tokensRequired, now, capacity int64) (bool, error) {
	res, err := tokenBucketScript.Run(ctx, s.Client, []string{key}, capacity, rate, tokensRequired, now, s.DefaultTtl).Int()
	if err != nil {
		return false, err
	}
	return res == 1, nil
}
