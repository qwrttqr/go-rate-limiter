package algos

import (
	"context"
	"fmt"
	"math"
	"qwrttqr-rate-limiter/core/server/cache"
	"qwrttqr-rate-limiter/core/server/internal"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type TokenBucketStore interface {
	TakeToken(ctx context.Context, key string, rate float64, tokensRequired, now, capacity int64) (bool, int64, error)
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

func ValidateTokenBucketConfig(cfg internal.Configuration) error {
	if *cfg.AlgoSettings.Rate <= 0 {
		return fmt.Errorf("the rate param should be strictly greater then 0")
	}
	return CheckRequiredFields(cfg.AlgoSettings, []string{"capacity", "rate"})
}

func NewTokenBucketStore(cfg internal.Configuration, cacheInstance cache.Cache, redisClient *redis.Client) (TokenBucketStore, error) {
	switch cfg.Store {
	case "in_memory":
		return &InMemoryBucketStore{Cache: cacheInstance}, nil
	case "redis":
		return &RedisBucketStore{Client: redisClient, DefaultTtl: cfg.Backends.Redis.DefaultTtl}, nil
	default:
		return nil, fmt.Errorf("unsupported store: %s", cfg.Store)
	}
}

func (tbl *TokenBucketLimiter) Allow(ctx context.Context, req Request) (Decision, error) {
	if err := req.validate(); err != nil {
		return Decision{}, err
	}
	allowed, retryAfter, err := tbl.Store.TakeToken(
		ctx,
		req.Key,
		tbl.Rate,
		req.RequiredTokens,
		tbl.Now(),
		tbl.Capacity,
	)
	if err != nil {
		return Decision{}, err
	}
	return Decision{Allowed: allowed, RetryAfter: retryAfter}, nil
}

type InMemoryBucketStore struct {
	Cache cache.Cache
}

type BucketState struct {
	mu         sync.Mutex
	Tokens     int64
	LastRefill int64
}

func (s *InMemoryBucketStore) TakeToken(
	_ context.Context,
	key string,
	rate float64,
	tokensRequired,
	now,
	capacity int64) (bool, int64, error) {
	val, err := s.Cache.Get(key)
	if err != nil {
		val = s.Cache.LoadOrStore(key, func() any {
			return &BucketState{
				Tokens:     capacity,
				LastRefill: now,
			}
		})
	}

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
		if int64(refillTokens) > 0 {
			bucket.LastRefill = now
		}
		return true, 0, nil
	}
	retryAfter := math.Ceil((float64(tokensRequired) - float64(newTokens)) / rate)
	return false, int64(retryAfter), nil
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
    return {1, 0}
else
    local retryAfter = math.ceil((tokensRequired - newTokens) / rate)
    redis.call("HMSET", KEYS[1], "tokens", newTokens, "last_refill", now)
    redis.call("EXPIRE", KEYS[1], ttl)
    return {0, retryAfter}
end
`)

func (s *RedisBucketStore) TakeToken(ctx context.Context, key string, rate float64, tokensRequired, now, capacity int64) (bool, int64, error) {
	res, err := tokenBucketScript.Run(ctx, s.Client, []string{key}, capacity, rate, tokensRequired, now, s.DefaultTtl).Int64Slice()
	if err != nil {
		return false, 0, err
	}
	allowed := res[0] == 1
	retryAfter := res[1]
	return allowed, retryAfter, nil
}
