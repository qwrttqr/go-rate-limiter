package facade

import (
	"errors"
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
		req, err := readRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		d, err := l.Allow(r.Context(), req)
		switch {
		case errors.Is(err, algos.ErrInvalidRequest):
			http.Error(w, err.Error(), http.StatusBadRequest) // caller's mistake
		case err != nil:
			http.Error(w, "rate limiter error", http.StatusInternalServerError) // store failure
		case !d.Allowed:
			w.Header().Set("Retry-After", strconv.FormatInt(d.RetryAfter, 10))
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		}
	}
}
