package limiting

import (
	"fmt"
	"io"
	"qwrttqr-rate-limiter/core/server/algos"
	"qwrttqr-rate-limiter/core/server/cache"
	"qwrttqr-rate-limiter/core/server/internal"
	"time"

	"github.com/redis/go-redis/v9"
)

func realClock() int64 { return time.Now().Unix() }

type RateLimiter struct {
	algos.Limiter
	io.Closer
}

func NewRateLimiter() (_ *RateLimiter, err error) {
	var cfg internal.Configuration
	cfg, err = internal.ReadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var (
		mem    cache.Cache
		rdb    *redis.Client
		closer io.Closer
	)
	switch cfg.Store {
	case "in_memory":
		cacheInstance := cache.NewCache(
			cfg.Backends.InMemory.DefaultTtl,
			time.Duration(cfg.Backends.InMemory.EvictionTime)*time.Second,
			cfg.Backends.InMemory.ShardsCount,
		)
		mem, closer = cacheInstance, cacheInstance
	case "redis":
		redisClient := redis.NewClient(&redis.Options{
			Addr:     cfg.Backends.Redis.Addr,
			Password: cfg.Backends.Redis.Password,
			DB:       cfg.Backends.Redis.Db,
		})
		rdb, closer = redisClient, redisClient
	default:
		return nil, fmt.Errorf("unsupported store: %q", cfg.Store)
	}

	defer func() {
		if err != nil {
			_ = closer.Close()
		}
	}()
	var limiter algos.Limiter

	switch cfg.UsedAlgo {
	case "token_bucket":
		if err := algos.ValidateTokenBucketConfig(cfg); err != nil {
			return nil, err
		}
		store, err := algos.NewTokenBucketStore(cfg, mem, rdb)
		if err != nil {
			return nil, err
		}
		limiter = &algos.TokenBucketLimiter{
			Capacity: *cfg.AlgoSettings.Capacity,
			Rate:     *cfg.AlgoSettings.Rate,
			Store:    store,
			Now:      realClock,
		}
	case "fixed_window":
		if err := algos.ValidateFixedWindowConfig(cfg); err != nil {
			return nil, err
		}
		store, err := algos.NewFixedWindowStore(cfg, mem, rdb)
		if err != nil {
			return nil, err
		}
		limiter = &algos.FixedWindowLimiter{
			MaxRequests: *cfg.AlgoSettings.MaxRequests,
			WindowSize:  *cfg.AlgoSettings.WindowSize,
			Store:       store,
			Now:         realClock,
		}
	case "rolling_window":
		if err := algos.ValidateRollingWindowConfiguration(cfg); err != nil {
			return nil, err
		}
		store, err := algos.NewRollingWindowStore(cfg, mem, rdb)
		if err != nil {
			return nil, err
		}
		limiter = &algos.RollingWindowLimiter{
			MaxRequests: *cfg.AlgoSettings.MaxRequests,
			WindowSize:  *cfg.AlgoSettings.WindowSize,
			Store:       store,
			Now:         realClock,
		}

	default:
		return nil, fmt.Errorf("unsupported rate limiting algorithm: %s", cfg.UsedAlgo)
	}
	return &RateLimiter{Limiter: limiter, Closer: closer}, nil
}
