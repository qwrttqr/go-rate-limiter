package algos

import (
	"net/http"
	"qwrttqr-rate-limiter/core/server/internal/config"
	"qwrttqr-rate-limiter/core/server/internal/utils"
	"sync"
	"time"
)

type TokenBucketLimiter struct {
	mu           sync.Mutex
	Capacity     int64
	Rate         float64
	StorageType  string
	inMemoryMap  sync.Map
	limitingHook func(key string, tokensRequired int64) bool
}

type BucketState struct {
	mu         sync.Mutex
	Tokens     int64
	LastRefill int64 // Unix timestamp in seconds
}

func (tbl *TokenBucketLimiter) Configure() {
	switch tbl.StorageType {
	case "in_memory":
		tbl.limitingHook = func(key string, tokensRequired int64) bool {
			val, _ := tbl.inMemoryMap.LoadOrStore(key, &BucketState{
				Tokens:     tbl.Capacity,
				LastRefill: time.Now().Unix(),
			})

			bucket := val.(*BucketState)

			bucket.mu.Lock()
			defer bucket.mu.Unlock()

			tokens := bucket.Tokens
			lastRefill := bucket.LastRefill
			currentTime := time.Now().Unix()

			timePassed := currentTime - lastRefill
			refillTokens := float64(timePassed) * tbl.Rate
			newTokens := min(tbl.Capacity, tokens+int64(refillTokens))
			lastRefill = time.Now().Unix()

			if newTokens >= tokensRequired {
				bucket.Tokens = newTokens - tokensRequired
				bucket.LastRefill = lastRefill
				return true
			} else {
				return false
			}
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
