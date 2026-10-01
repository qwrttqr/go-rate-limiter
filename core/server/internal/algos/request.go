package algos

import (
	"errors"
	"net/http"
	"strconv"
)

func ReadIncomingHeader(r *http.Request) (*IncomingHeaders, error) {
	defer r.Body.Close()
	clientKey := r.Header.Get("X-Client-Key")
	requiredTokensStr := r.Header.Get("X-Required-Tokens")
	if clientKey == "" {
		return nil, errors.New("missing mandatory X-Client-Key header")
	}
	var requiredTokens int64 = 1
	if requiredTokensStr != "" {
		parsedTokens, err := strconv.ParseInt(requiredTokensStr, 10, 64)
		if err != nil {
			return nil, errors.New("invalid X-Required-Tokens header: must be a valid integer")
		}
		if parsedTokens <= 0 {
			return nil, errors.New("invalid X-Required-Tokens header: must be greater than 0")
		}
		requiredTokens = parsedTokens
	}
	return &IncomingHeaders{ClientKey: clientKey, RequiredTokens: requiredTokens}, nil
}
