package algos

import (
	"fmt" // Imported to handle or print errors if needed
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
	Tokens     int64
	LastRefill int64 // Unix timestamp in seconds or milliseconds
}

func (tbl *TokenBucketLimiter) Configure() {
	switch tbl.StorageType {
	case "in_memory":
		tbl.limitingHook = func(key string, tokensRequired int64) bool {

			val, ok := tbl.inMemoryMap.Load(key)
			if !ok {
				state := BucketState{
					Tokens:     tbl.Capacity,
					LastRefill: time.Now().Unix(),
				}
				actual, _ := tbl.inMemoryMap.LoadOrStore(key, state)
				val = actual
			}

			bucket := val.(BucketState)
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
				tbl.inMemoryMap.Store(key, bucket)
				return true
			} else {
				return false
			}
		}
	}
}

func (tbl *TokenBucketLimiter) ValidateAlgoSettings(cfg config.Configuration) error {
	requiredFields := []string{"capacity", "rate"}
	err := utils.CheckRequiredFields(cfg.AlgoSettings, requiredFields)
	if err != nil {
		return fmt.Errorf("token bucket validation failed: %w", err)
	}

	return nil
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
