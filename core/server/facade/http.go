package facade

import (
	"errors"
	"net/http"
	"strconv"

	"qwrttqr-rate-limiter/core/server/algos"
)

const maxKeyLen = 128

func readRequest(r *http.Request) (algos.Request, error) {
	key := r.Header.Get("X-Client-Key")
	if key == "" {
		return algos.Request{}, errors.New("missing mandatory X-Client-Key header")
	}
	if len(key) > maxKeyLen {
		return algos.Request{}, errors.New("X-Client-Key header is too long")
	}

	tokens := int64(1)
	if s := r.Header.Get("X-Required-Tokens"); s != "" {
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return algos.Request{}, errors.New("invalid X-Required-Tokens header: must be a valid integer")
		}
		if n <= 0 {
			return algos.Request{}, errors.New("invalid X-Required-Tokens header: must be greater than 0")
		}
		tokens = n
	}
	return algos.Request{Key: key, RequiredTokens: tokens}, nil
}
