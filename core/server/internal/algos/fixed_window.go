package algos

import (
	"net/http"
	"qwrttqr-rate-limiter/core/server/internal/config"
	"qwrttqr-rate-limiter/core/server/internal/interfaces"
	"qwrttqr-rate-limiter/core/server/internal/utils"
	"sync"
	"time"
)

type FixedWindowLimiter struct {
	WindowSize   int64
	MaxRequests  int64
	StorageType  string
	limitingHook func(key string) bool
	Cache        interfaces.Cache
	Now          func() int64
}

type FixedWindowState struct {
	mu           sync.Mutex
	Count        int64
	StoredWindow int64
}

func (fwl *FixedWindowLimiter) Configure() {
	if fwl.Now == nil {
		fwl.Now = func() int64 {
			return time.Now().Unix()
		}
	}
	switch fwl.StorageType {
	case "in_memory":
		fwl.limitingHook = func(key string) bool {

			currentTime := fwl.Now()
			currentWindow := currentTime / fwl.WindowSize
			windowStart := currentWindow * fwl.WindowSize
			val := fwl.Cache.LoadOrStore(key, &FixedWindowState{
				Count:        0,
				StoredWindow: windowStart,
			})

			window := val.(*FixedWindowState)

			window.mu.Lock()
			defer window.mu.Unlock()

			if window.StoredWindow < windowStart {
				window.Count = 0
				window.StoredWindow = windowStart
			}
			if window.Count < fwl.MaxRequests {
				window.Count++
				return true
			}
			return false
		}
	}
}

func ValidateFixedWindowConfig(cfg config.Configuration) error {
	return utils.CheckRequiredFields(cfg.AlgoSettings, []string{"window_size", "max_requests"})
}

func (fwl *FixedWindowLimiter) LimitHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := utils.ReadIncomingBody(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	allowed := fwl.limitingHook(body.ClientKey)
	if !allowed {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}
}
