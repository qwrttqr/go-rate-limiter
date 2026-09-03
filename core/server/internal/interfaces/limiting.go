package interfaces

import (
	"net/http"
)

type IncomingBody struct {
	ClientKey      string `json:"client_key"`
	RequiredTokens *int64 `json:"required_tokens"`
}

type RateLimiterInterface interface {
	Configure()
	LimitHTTP(w http.ResponseWriter, r *http.Request)
}
