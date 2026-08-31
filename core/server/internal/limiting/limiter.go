package limiting

import (
	"qwrttqr-rate-limiter/core/server/internal/config"
)

type Limiter interface {
	ValidateAlgoSettings(config.Configuration) error
	Limit()
}

type RateLimiter struct {
}

func (limiter *RateLimiter) ValidateAlgoSettings(cfg config.Configuration) bool {

}
