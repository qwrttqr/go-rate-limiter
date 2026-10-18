package metrics

import (
	"context"
	"qwrttqr-rate-limiter/core/server/algos"
	"time"
)

type instrumented struct {
	next algos.Limiter
	rec  Recorder
}

func Instrument(next algos.Limiter, rec Recorder) algos.Limiter {
	return &instrumented{next: next, rec: rec}
}

func (i *instrumented) Allow(ctx context.Context, req algos.Request) (algos.Decision, error) {
	start := time.Now()
	dec, err := i.next.Allow(ctx, req)
	elapsed := time.Since(start)

	switch {
	case err != nil:
		i.rec.ObserveDecision(Errored, elapsed)
	case dec.Allowed:
		i.rec.ObserveDecision(Allowed, elapsed)
	default:
		i.rec.ObserveDecision(Denied, elapsed)
	}
	return dec, err
}
