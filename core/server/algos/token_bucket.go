package algos

import (
	"context"
	"fmt"
	"math"
	"qwrttqr-rate-limiter/core/config"
	"qwrttqr-rate-limiter/core/server/cache"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type TokenBucketStore interface {
	TakeToken(ctx context.Context, key string, now int64, rate, tokensRequired, capacity float64) (bool, int64, error)
}
type TokenBucketLimiter struct {
	Capacity float64
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

func ValidateTokenBucketConfig(cfg config.AlgoSettings) error {
	if *cfg.Rate <= 0 {
		return fmt.Errorf("the rate param should be strictly greater then 0")
	}
	return CheckRequiredFields(cfg, []string{"capacity", "rate"})
}

func NewTokenBucketStore(store string, cacheInstance cache.InMemoryStoreInterface, redisClient *redis.Client, redisDefaultKeyTTL int64) (TokenBucketStore, error) {
	switch store {
	case "in_memory":
		return &InMemoryTokenBucketStore{Store: cacheInstance}, nil
	case "redis":
		return &RedisTokenBucketStore{Store: cache.NewRedisStore(redisClient, redisDefaultKeyTTL)}, nil
	default:
		return nil, fmt.Errorf("unsupported store: %s", store)
	}
}

func (tbl *TokenBucketLimiter) Allow(ctx context.Context, req Request) (Decision, error) {
	allowed, retryAfter, err := tbl.Store.TakeToken(
		ctx,
		req.Key,
		tbl.Now(),
		tbl.Rate,
		req.RequiredTokens,
		tbl.Capacity,
	)
	if err != nil {
		return Decision{}, err
	}
	return Decision{Allowed: allowed, RetryAfter: retryAfter}, nil
}

type InMemoryTokenBucketStore struct {
	Store cache.InMemoryStoreInterface
}
type RedisTokenBucketStore struct {
	Store cache.RedisStoreInterface
}

type BucketState struct {
	mu         sync.Mutex
	Tokens     float64
	LastRefill int64
}

func (s *InMemoryTokenBucketStore) TakeToken(
	_ context.Context,
	key string,
	now int64,
	rate,
	requiredTokens,
	capacity float64) (bool, int64, error) {
	val, err := s.Store.Get(key)
	if err != nil {
		val = s.Store.LoadOrStore(key, func() any {
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
	newTokens := min(capacity, tokens+refillTokens)

	bucket.LastRefill = now

	if newTokens >= requiredTokens {
		bucket.Tokens = newTokens
		bucket.Tokens -= requiredTokens
		return true, 0, nil
	}
	bucket.Tokens = newTokens
	retryAfter := math.Ceil((requiredTokens - bucket.Tokens) / rate)
	return false, int64(retryAfter), nil
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

func (s *RedisTokenBucketStore) TakeToken(
	ctx context.Context,
	key string,
	now int64,
	rate,
	tokensRequired,
	capacity float64,
) (bool, int64, error) {
	return s.Store.RunLuaScript(ctx, tokenBucketScript, []string{key}, rate, tokensRequired, capacity, now, s.Store.DefaultTtl())
}
