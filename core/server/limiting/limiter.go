package limiting

import (
	"fmt"
	algos2 "qwrttqr-rate-limiter/core/server/algos"
	"qwrttqr-rate-limiter/core/server/cache"
	"qwrttqr-rate-limiter/core/server/internal/config"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewRateLimiter() (algos2.RateLimiterInterface, error) {
	cfg, err := config.ReadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cacheInstance *cache.ShardedInMemoryCache
	var redisClient *redis.Client
	switch cfg.Store {
	case "in_memory":
		cacheInstance = cache.NewCache(
			cfg.Backends.InMemory.DefaultTtl,
			time.Duration(cfg.Backends.InMemory.EvictionTime)*time.Second,
		)
	case "redis":
		redisClient = redis.NewClient(&redis.Options{
			Addr:     cfg.Backends.Redis.Addr,
			Password: cfg.Backends.Redis.Password,
			DB:       cfg.Backends.Redis.Db,
		})
	}
	switch cfg.UsedAlgo {
	case "token_bucket":
		if err := algos2.ValidateTokenBucketConfig(cfg); err != nil {
			return nil, err
		}
		store, err := algos2.NewTokenBucketStore(cfg, cacheInstance, redisClient)
		if err != nil {
			return nil, err
		}
		limiter := &algos2.TokenBucketLimiter{
			Capacity: *cfg.AlgoSettings.Capacity,
			Rate:     *cfg.AlgoSettings.Rate,
			Store:    store,
		}
		return limiter, nil
	case "fixed_window":
		if err := algos2.ValidateFixedWindowConfig(cfg); err != nil {
			return nil, err
		}
		store, err := algos2.NewFixedWindowStore(cfg, cacheInstance, redisClient)
		if err != nil {
			return nil, err
		}
		limiter := &algos2.FixedWindowLimiter{
			MaxRequests: *cfg.AlgoSettings.MaxRequests,
			WindowSize:  *cfg.AlgoSettings.WindowSize,
			Store:       store,
		}
		return limiter, nil

	case "rolling_window":
		if err := algos2.ValidateRollingWindowConfiguration(cfg); err != nil {
			return nil, err
		}
		store, err := algos2.NewRollingWindowStore(cfg, cacheInstance, redisClient)
		if err != nil {
			return nil, err
		}
		limiter := &algos2.RollingWindowLimiter{
			MaxRequests: *cfg.AlgoSettings.MaxRequests,
			WindowSize:  *cfg.AlgoSettings.WindowSize,
			Store:       store,
		}
		return limiter, nil

	default:
		return nil, fmt.Errorf("unsupported rate limiting algorithm: %s", cfg.UsedAlgo)
	}
}
