package middleware

import (
	"qwrttqr-rate-limiter/core/config"
)

func NewTokenBucketConfig(capacity, rate float64) config.AlgoSettings {
	return config.AlgoSettings{Capacity: &capacity, Rate: &rate}
}

func NewFixedWindowConfig(windowSize, maxRequests int64) config.AlgoSettings {
	return config.AlgoSettings{WindowSize: &windowSize, MaxRequests: &maxRequests}
}

func NewSlidingWindowConfig(windowSize, maxRequests int64) config.AlgoSettings {
	return config.AlgoSettings{WindowSize: &windowSize, MaxRequests: &maxRequests}
}
