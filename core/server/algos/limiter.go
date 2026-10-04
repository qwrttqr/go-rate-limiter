package algos

import (
	"context"
	"errors"
	"fmt"
)

type Request struct {
	Key            string
	RequiredTokens int64
}

type Decision struct {
	Allowed    bool
	RetryAfter int64
}

type Limiter interface {
	Allow(ctx context.Context, req Request) (Decision, error)
}

var ErrInvalidRequest = errors.New("invalid request")

func (r Request) validate() error {
	if r.Key == "" {
		return fmt.Errorf("%w: empty key", ErrInvalidRequest)
	}
	return nil
}
