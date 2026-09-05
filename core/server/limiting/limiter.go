package limiting

import (
	"fmt"
	"qwrttqr-rate-limiter/core/server/internal/algos"
	"qwrttqr-rate-limiter/core/server/internal/cache"
	"qwrttqr-rate-limiter/core/server/internal/config"
	"qwrttqr-rate-limiter/core/server/internal/interfaces"
	"time"
)

func NewRateLimiter() (interfaces.RateLimiterInterface, error) {
	cfg, err := config.ReadConfig() // Good practice: Handle this error too
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// 1. Declare the pointer OUTSIDE the if-block scope so it propagates down
	var cacheInstance *cache.InMemoryCache

	// 2. Initialize it only if in_memory storage is requested
	if cfg.Store == "in_memory" {
		cacheInstance = cache.NewCache(
			*cfg.CacheSettings.ExpirationTime,
			time.Duration(*cfg.CacheSettings.EvictionTime)*time.Second,
		)
	}

	switch cfg.UsedAlgo {
	case "token_bucket":
		if err := algos.ValidateTokenBucketConfig(cfg); err != nil {
			return nil, err
		}

		limiter := &algos.TokenBucketLimiter{
			Capacity:    *cfg.AlgoSettings.Capacity,
			Rate:        *cfg.AlgoSettings.Rate,
			StorageType: cfg.Store,
			Cache:       cacheInstance,
		}
		return limiter, nil

	case "fixed_window":
		if err := algos.ValidateFixedWindowConfig(cfg); err != nil {
			return nil, err
		}
		limiter := &algos.FixedWindowLimiter{
			MaxRequests: *cfg.AlgoSettings.MaxRequests,
			WindowSize:  *cfg.AlgoSettings.WindowSize,
			StorageType: cfg.Store,
			Cache:       cacheInstance,
		}
		return limiter, nil

	case "rolling_window":
		if err := algos.ValidateRollingWindowConfiguration(cfg); err != nil {
			return nil, err
		}
		limiter := &algos.RollingWindowLimiter{
			MaxRequests: *cfg.AlgoSettings.MaxRequests,
			WindowSize:  *cfg.AlgoSettings.WindowSize,
			StorageType: cfg.Store,
			Cache:       cacheInstance,
		}
		return limiter, nil

	default:
		return nil, fmt.Errorf("unsupported rate limiting algorithm: %s", cfg.UsedAlgo)
	}
}
