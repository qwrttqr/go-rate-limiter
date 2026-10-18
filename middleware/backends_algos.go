package middleware

import (
	"qwrttqr-rate-limiter/core/config"
	"qwrttqr-rate-limiter/core/server/algos"
	"qwrttqr-rate-limiter/core/server/cache"
	"time"

	"github.com/redis/go-redis/v9"
)

var realClock = func() int64 { return time.Now().Unix() }

type Backend interface {
	TokenBucket(cfg config.AlgoSettings) (algos.Limiter, error)
	FixedWindow(cfg config.AlgoSettings) (algos.Limiter, error)
	RollingWindow(cfg config.AlgoSettings) (algos.Limiter, error)
}

type memoryBackend struct {
	mem cache.InMemoryStoreInterface
}

func NewInMemoryBackend(cfg config.InMemoryBackendSettings) Backend {
	return &memoryBackend{mem: cache.NewCache(
		cfg.DefaultTtl,
		time.Duration(cfg.EvictionTime)*time.Second,
		cfg.ShardsCount),
	}
}

type redisBackend struct {
	redis cache.RedisStoreInterface
}

func NewRedisBackend(client *redis.Client, redisDefaultKeyTTL int64) Backend {
	return &redisBackend{redis: cache.NewRedisStore(client, redisDefaultKeyTTL)}
}

func newTokenBucket(cfg config.AlgoSettings, store algos.TokenBucketStore) (algos.Limiter, error) {
	if err := algos.ValidateTokenBucketConfig(cfg); err != nil {
		return nil, err
	}
	return &algos.TokenBucketLimiter{
		Capacity: *cfg.Capacity, Rate: *cfg.Rate, Store: store, Now: realClock,
	}, nil
}

func newFixedWindow(cfg config.AlgoSettings, store algos.FixedWindowStore) (algos.Limiter, error) {
	if err := algos.ValidateFixedWindowConfig(cfg); err != nil {
		return nil, err
	}
	return &algos.FixedWindowLimiter{
		WindowSize: *cfg.WindowSize, MaxRequests: *cfg.MaxRequests, Store: store, Now: realClock,
	}, nil
}

func newRollingWindow(cfg config.AlgoSettings, store algos.RollingWindowStore) (algos.Limiter, error) {
	if err := algos.ValidateRollingWindowConfig(cfg); err != nil {
		return nil, err
	}
	return &algos.RollingWindowLimiter{
		WindowSize: *cfg.WindowSize, MaxRequests: *cfg.MaxRequests, Store: store, Now: realClock,
	}, nil
}

func (b *memoryBackend) TokenBucket(cfg config.AlgoSettings) (algos.Limiter, error) {
	return newTokenBucket(cfg, &algos.InMemoryTokenBucketStore{Backend: b.mem})
}

func (b *memoryBackend) FixedWindow(cfg config.AlgoSettings) (algos.Limiter, error) {
	return newFixedWindow(cfg, &algos.InMemoryFixedWindowStore{Backend: b.mem})
}

func (b *memoryBackend) RollingWindow(cfg config.AlgoSettings) (algos.Limiter, error) {
	return newRollingWindow(cfg, &algos.InMemoryRollingWindowStore{Backend: b.mem})
}

func (b *redisBackend) TokenBucket(cfg config.AlgoSettings) (algos.Limiter, error) {
	return newTokenBucket(cfg, &algos.RedisTokenBucketStore{Backend: b.redis})
}

func (b *redisBackend) FixedWindow(cfg config.AlgoSettings) (algos.Limiter, error) {
	return newFixedWindow(cfg, &algos.RedisFixedWindowStore{Store: b.redis})
}

func (b *redisBackend) RollingWindow(cfg config.AlgoSettings) (algos.Limiter, error) {
	return newRollingWindow(cfg, &algos.RedisRollingWindowStore{Backend: b.redis})
}

var limiterBuilders = map[string]func(Backend, config.AlgoSettings) (algos.Limiter, error){
	"token_bucket":   Backend.TokenBucket,
	"fixed_window":   Backend.FixedWindow,
	"rolling_window": Backend.RollingWindow,
}
