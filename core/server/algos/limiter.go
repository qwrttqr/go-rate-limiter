package algos

import (
	"context"
)

type Request struct {
	Key            string
	RequiredTokens float64
}

type Decision struct {
	Allowed    bool
	RetryAfter int64
}

type Limiter interface {
	Allow(ctx context.Context, req Request) (Decision, error)
}
