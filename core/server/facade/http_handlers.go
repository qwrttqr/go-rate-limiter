package facade

import (
	"net/http"
	"qwrttqr-rate-limiter/core/server/algos"
	"strconv"
)

func HealthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func LimitHandler(l algos.Limiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-Client-Key")
		tokens := 1.0
		if s := r.Header.Get("X-Required-Tokens"); s != "" {
			n, err := strconv.ParseFloat(s, 64)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			tokens = n
		}
		req, err := NewRequest(key, tokens)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		d, err := l.Allow(r.Context(), req)
		switch {
		case err != nil:
			http.Error(w, "rate limiter error", http.StatusInternalServerError) // store failure
		case !d.Allowed:
			w.Header().Set("Retry-After", strconv.FormatInt(d.RetryAfter, 10))
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		}
	}
}
