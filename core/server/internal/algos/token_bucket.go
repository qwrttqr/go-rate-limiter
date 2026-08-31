package algos

import (
	"fmt" // Imported to handle or print errors if needed
	"qwrttqr-rate-limiter/core/server/internal/config"
	"qwrttqr-rate-limiter/core/server/internal/utils"
)

type TokenBucketLimiter struct {
	Capacity int
	Rate     int
}

func ValidateTokenBucket(cfg config.Configuration) error {
	requiredFields := []string{"capacity", "rate"}
	err := utils.CheckRequiredFields(cfg.AlgoSettings, requiredFields)
	if err != nil {
		return fmt.Errorf("token bucket validation failed: %w", err)
	}

	return nil
}

func (l *TokenBucketLimiter) Limit() {

}
