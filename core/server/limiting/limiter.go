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
		err := algos.ValidateTokenBucketConfig(cfg)
		if err != nil {
			return nil, err
		}
		limiter := &algos.TokenBucketLimiter{Capacity: *cfg.AlgoSettings.Capacity, Rate: *cfg.AlgoSettings.Rate, StorageType: cfg.Store}
		return limiter, nil
	case "fixed_window":
		err := algos.ValidateFixedWindowConfig(cfg)
		if err != nil {
			return nil, err
		}
		limiter := &algos.FixedWindowLimiter{MaxRequests: *cfg.AlgoSettings.MaxRequests, WindowSize: *cfg.AlgoSettings.WindowSize, StorageType: cfg.Store}
		return limiter, nil
	case "rolling_window":
		err := algos.ValidateRollingWindowConfiguration(cfg)
		if err != nil {
			return nil, err
		}
		limiter := &algos.RollingWindowLimiter{MaxRequests: *cfg.AlgoSettings.MaxRequests, WindowSize: *cfg.AlgoSettings.WindowSize, StorageType: cfg.Store}
		return limiter, nil
	default:
		return nil, fmt.Errorf("unsupported rate limiting algorithm: %s", cfg.UsedAlgo)
	}
}
