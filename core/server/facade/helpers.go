package facade

import (
	"errors"
	"qwrttqr-rate-limiter/core/server/algos"
)

func newRequest(key string, tokens float64) (algos.Request, error) {
	if key == "" {
		return algos.Request{}, errors.New("missing mandatory X-Client-Key header")
	}
	if tokens <= 0 {
		return algos.Request{}, errors.New("invalid X-Required-Tokens header: must be greater than 0")
	}
	return algos.Request{Key: key, RequiredTokens: tokens}, nil
}
