package algos

import (
	"net/http"
)

type IncomingHeaders struct {
	ClientKey      string
	RequiredTokens int64
}

type RateLimiterInterface interface {
	Configure()
	LimitHTTP(w http.ResponseWriter, r *http.Request)
}
