package algos

import (
	"context"
	"testing"
)

func TestTokenBucket_logic(t *testing.T) {
	tests := []struct {
		name           string
		key            string
		capacity       int64
		tokensRequired *int64
		rate           float64
		disallowOn     int
	}{
		{
			name:       "single-request-allowed",
			key:        "user-123",
			capacity:   10,
			rate:       1,
			disallowOn: -1,
		},
		{
			name:           "too-expensive-request-income",
			key:            "user-123",
			capacity:       10,
			tokensRequired: int64Ptr(100),
			rate:           1,
			disallowOn:     0,
		},
		{
			name:       "request-disallowed-when-out-of-capacity",
			key:        "user-123",
			capacity:   10,
			rate:       0,
			disallowOn: 10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tokensRequired int64 = 1
			if tt.tokensRequired != nil {
				tokensRequired = *tt.tokensRequired
			}
			cache := newMockCache()
			store := &InMemoryBucketStore{Cache: cache}
			tb := &TokenBucketLimiter{
				Capacity: tt.capacity,
				Rate:     tt.rate,
				Store:    store,
			}
			tb.Configure()

			calls := max(tt.disallowOn, 0)

			for i := 0; i <= calls; i++ {
				allowed, err := tb.Store.TakeToken(context.Background(), tt.key, tb.Rate, tokensRequired, tb.Now(), tb.Capacity)
				if err != nil {
					t.Fatalf("call %d: unexpected error: %v", i, err)
				}
				wantAllowed := !(tt.disallowOn >= 0 && i == tt.disallowOn)
				if allowed != wantAllowed {
					t.Errorf("call %d: allowed=%v, want %v", i, allowed, wantAllowed)
				}
			}
		})
	}
}

func TestTokenBucket_refill(t *testing.T) {
	var fakeNow int64 = 1000
	cache := newMockCache()
	store := &InMemoryBucketStore{Cache: cache}
	tb := &TokenBucketLimiter{
		Capacity: 10,
		Rate:     1,
		Store:    store,
		Now:      func() int64 { return fakeNow },
	}
	tb.Configure()

	ctx := context.Background()
	take := func(tokens int64) (bool, error) {
		return tb.Store.TakeToken(ctx, "user-123", tb.Rate, tokens, tb.Now(), tb.Capacity)
	}

	for i := range 10 {
		allowed, err := take(1)
		if err != nil {
			t.Fatalf("call %d unexpected error: %v", i, err)
		}
		if !allowed {
			t.Fatalf("call %d unexpected disallow while draining", i)
		}
	}
	if allowed, err := take(1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if allowed {
		t.Fatal("expected disallowed when bucket is empty")
	}

	fakeNow += 5

	if allowed, err := take(5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if !allowed {
		t.Fatal("expected allowed after refill of 5 tokens")
	}

	fakeNow += 5

	for i := range 5 {
		allowed, err := take(1)
		if err != nil {
			t.Fatalf("call %d unexpected error: %v", i, err)
		}
		if !allowed {
			t.Fatalf("call %d unexpected disallow while draining", i)
		}
	}

	if allowed, err := take(5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if allowed {
		t.Fatal("expected disallowed when bucket is empty")
	}
}
