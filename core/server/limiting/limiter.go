package limiting

import (
	"fmt"
	"qwrttqr-rate-limiter/core/server/internal/algos"
	"qwrttqr-rate-limiter/core/server/internal/config"
	"qwrttqr-rate-limiter/core/server/internal/interfaces"
)

func NewRateLimiter() (interfaces.RateLimiterInterface, error) {
	cfg, _ := config.ReadConfig()
	switch cfg.UsedAlgo {
	case "token_bucket":
		limiter := &algos.TokenBucketLimiter{Capacity: *cfg.AlgoSettings.Capacity, Rate: *cfg.AlgoSettings.Rate, StorageType: cfg.Store}
		err := limiter.ValidateAlgoSettings(cfg)
		if err != nil {
			return nil, err
		}
		return limiter, nil
	default:
		return nil, fmt.Errorf("unsupported rate limiting algorithm: %s", cfg.UsedAlgo)
	}
}
