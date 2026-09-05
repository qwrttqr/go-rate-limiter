package algos

import (
	"net/http"
	"qwrttqr-rate-limiter/core/server/internal/config"
	"qwrttqr-rate-limiter/core/server/internal/interfaces"
	"qwrttqr-rate-limiter/core/server/internal/utils"
	"sync"
	"time"
)

type TokenBucketLimiter struct {
	mu           sync.Mutex
	Capacity     int64
	Rate         float64
	StorageType  string
	limitingHook func(key string, tokensRequired int64) bool
	Cache        interfaces.Cache
	Now          func() int64
}

type BucketState struct {
	mu         sync.Mutex
	Tokens     int64
	LastRefill int64
}

func (tbl *TokenBucketLimiter) Configure() {
	if tbl.Now == nil {
		tbl.Now = func() int64 {
			return time.Now().Unix()
		}
	}
	switch tbl.StorageType {
	case "in_memory":
		tbl.limitingHook = func(key string, tokensRequired int64) bool {
			val := tbl.Cache.LoadOrStore(key, &BucketState{
				Tokens:     tbl.Capacity,
				LastRefill: tbl.Now(),
			})

			bucket := val.(*BucketState)

			bucket.mu.Lock()
			defer bucket.mu.Unlock()

			tokens := bucket.Tokens
			lastRefill := bucket.LastRefill
			currentTime := tbl.Now()

			timePassed := currentTime - lastRefill
			refillTokens := float64(timePassed) * tbl.Rate
			newTokens := min(tbl.Capacity, tokens+int64(refillTokens))
			lastRefill = tbl.Now()

			if newTokens >= tokensRequired {
				bucket.Tokens = newTokens - tokensRequired
				bucket.LastRefill = lastRefill
				return true
			}

			return false
		}
	}
}

func ValidateTokenBucketConfig(cfg config.Configuration) error {
	return utils.CheckRequiredFields(cfg.AlgoSettings, []string{"capacity", "rate"})
}

func (tbl *TokenBucketLimiter) LimitHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := utils.ReadIncomingBody(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var requiredTokens int64 = 1
	if body.RequiredTokens != nil {
		requiredTokens = *body.RequiredTokens
	}
	allowed := tbl.limitingHook(body.ClientKey, requiredTokens)
	if !allowed {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}
}
