package interfaces

import (
	"net/http"
	"qwrttqr-rate-limiter/core/server/internal/config"
)

type IncomingBody struct {
	ClientKey      string `json:"client_key"`
	RequiredTokens *int64 `json:"required_tokens"`
}

type RateLimiterInterface interface {
	ValidateAlgoSettings(config.Configuration) error
	Configure()
	LimitHTTP(w http.ResponseWriter, r *http.Request)
}
