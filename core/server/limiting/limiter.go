package limiting

import (
	"fmt"
	"qwrttqr-rate-limiter/core/server/internal/algos"
	"qwrttqr-rate-limiter/core/server/internal/cache"
	"qwrttqr-rate-limiter/core/server/internal/config"
	"qwrttqr-rate-limiter/core/server/internal/interfaces"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewRateLimiter() (interfaces.RateLimiterInterface, error) {
	cfg, err := config.ReadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cacheInstance *cache.InMemoryCache
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
		if err := algos.ValidateTokenBucketConfig(cfg); err != nil {
			return nil, err
		}
		store, err := algos.NewTokenBucketStore(cfg, cacheInstance, redisClient)
		if err != nil {
			return nil, err
		}
		limiter := &algos.TokenBucketLimiter{
			Capacity: *cfg.AlgoSettings.Capacity,
			Rate:     *cfg.AlgoSettings.Rate,
			Store:    store,
		}
		return limiter, nil
	case "fixed_window":
		if err := algos.ValidateFixedWindowConfig(cfg); err != nil {
			return nil, err
		}
		store, err := algos.NewFixedWindowStore(cfg, cacheInstance, redisClient)
		if err != nil {
			return nil, err
		}
		limiter := &algos.FixedWindowLimiter{
			MaxRequests: *cfg.AlgoSettings.MaxRequests,
			WindowSize:  *cfg.AlgoSettings.WindowSize,
			Store:       store,
		}
		return limiter, nil

	case "rolling_window":
		if err := algos.ValidateRollingWindowConfiguration(cfg); err != nil {
			return nil, err
		}
		store, err := algos.NewRollingWindowStore(cfg, cacheInstance, redisClient)
		if err != nil {
			return nil, err
		}
		limiter := &algos.RollingWindowLimiter{
			MaxRequests: *cfg.AlgoSettings.MaxRequests,
			WindowSize:  *cfg.AlgoSettings.WindowSize,
			Store:       store,
		}
		return limiter, nil

	default:
		return nil, fmt.Errorf("unsupported rate limiting algorithm: %s", cfg.UsedAlgo)
	}
}
