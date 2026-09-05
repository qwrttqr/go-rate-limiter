package algos

import (
	"net/http"
	"qwrttqr-rate-limiter/core/server/internal/config"
	"qwrttqr-rate-limiter/core/server/internal/interfaces"
	"qwrttqr-rate-limiter/core/server/internal/utils"
	"sync"
	"time"
)

type RollingWindowLimiter struct {
	WindowSize   int64
	MaxRequests  int64
	StorageType  string
	limitingHook func(key string) bool
	Cache        interfaces.Cache
	Now          func() int64
}

type RollingWindowState struct {
	mu         sync.Mutex
	Timestamps []int64
}

func (rwl *RollingWindowLimiter) Configure() {
	if rwl.Now == nil {
		rwl.Now = func() int64 {
			return time.Now().Unix()
		}
	}
	switch rwl.StorageType {
	case "in_memory":
		rwl.limitingHook = func(key string) bool {

			currentTime := rwl.Now()
			windowStart := currentTime - rwl.WindowSize
			val := rwl.Cache.LoadOrStore(key, &RollingWindowState{
				Timestamps: make([]int64, 0, rwl.MaxRequests*2),
			})
			window := val.(*RollingWindowState)

			window.mu.Lock()
			defer window.mu.Unlock()

			i := 0
			for ; i < len(window.Timestamps); i++ {
				if window.Timestamps[i] > windowStart {
					break
				}
			}
			window.Timestamps = window.Timestamps[i:]

			if int64(len(window.Timestamps)) < rwl.MaxRequests {
				window.Timestamps = append(window.Timestamps, currentTime)
				return true
			}
			return false
		}
	}
}

func ValidateRollingWindowConfiguration(cfg config.Configuration) error {
	return utils.CheckRequiredFields(cfg.AlgoSettings, []string{"window_size", "max_requests"})
}

func (rwl *RollingWindowLimiter) LimitHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := utils.ReadIncomingBody(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	allowed := rwl.limitingHook(body.ClientKey)
	if !allowed {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}
}
