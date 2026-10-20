package middleware

import (
	"log"
	"net/http"
	"qwrttqr-rate-limiter/core/server/algos"
	"qwrttqr-rate-limiter/core/server/facade"
	"strconv"
)

func NewHTTPMiddleware(
	algo algos.Limiter,
	keyDer HTTPKeyDerivationFunction,
	costDer HTTPCostDerivationFunction,
	failOpen bool,
) func(http.Handler) http.Handler {
	if algo == nil || keyDer == nil {
		panic("middleware: limiter and key function are required")
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fail := func(stage string, err error) {
				log.Printf("limiter error on %s: %v", stage, err)
				if failOpen {
					next.ServeHTTP(w, r)
					return
				}
				http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			}

			tokens := 1.0
			if costDer != nil {
				t, err := costDer(r)
				if err != nil {
					fail("cost derivation", err)
					return
				}
				tokens = t
			}

			key, err := keyDer(r)
			if err != nil {
				fail("key derivation", err)
				return
			}

			req, err := facade.NewRequest(key, tokens)
			if err != nil {
				fail("request creation", err)
				return
			}

			decision, err := algo.Allow(r.Context(), req)
			if err != nil {
				fail("allow", err)
				return
			}
			if !decision.Allowed {
				w.Header().Set("Retry-After", strconv.FormatInt(decision.RetryAfter, 10))
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
