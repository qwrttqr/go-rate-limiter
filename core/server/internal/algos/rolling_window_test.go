package algos

import (
	"context"
	"testing"
)

func TestRollingWindow_logic(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		windowSize  int64
		maxRequests int64
	}{
		{
			name:        "single-request-allowed",
			key:         "user-123",
			windowSize:  10,
			maxRequests: 1,
		},
		{
			name:        "request-disallowed-when-exceed",
			key:         "user-123",
			windowSize:  10,
			maxRequests: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := newMockCache()
			store := &InMemoryRollingWindowStore{Cache: cache}
			rw := &RollingWindowLimiter{
				WindowSize:  tt.windowSize,
				MaxRequests: tt.maxRequests,
				Store:       store,
			}
			rw.Configure()
			calls := int(tt.maxRequests)

			for i := 0; i <= calls; i++ {
				currentTime := rw.Now()
				windowStart := currentTime - rw.WindowSize
				allowed, err := rw.Store.CheckAndIncrement(context.Background(), tt.key, windowStart, rw.WindowSize, currentTime, rw.MaxRequests)
				if err != nil {
					t.Fatalf("call %d: unexpected error: %v", i, err)
				}
				wantAllowed := i != calls
				if allowed != wantAllowed {
					t.Errorf("call %d: allowed=%v, want %v", i, allowed, wantAllowed)
				}
			}
		})
	}
}

func TestRollingWindow_window_moved(t *testing.T) {
	var fakeNow int64 = 1000
	cache := newMockCache()
	store := &InMemoryRollingWindowStore{Cache: cache}
	rw := &RollingWindowLimiter{
		WindowSize:  10,
		MaxRequests: 5,
		Store:       store,
		Now:         func() int64 { return fakeNow },
	}
	rw.Configure()

	ctx := context.Background()
	hit := func() (bool, error) {
		currentTime := rw.Now()
		windowStart := currentTime - rw.WindowSize
		return rw.Store.CheckAndIncrement(ctx, "user-123", windowStart, rw.WindowSize, currentTime, rw.MaxRequests)
	}

	for i := range 5 {
		allowed, err := hit()
		if err != nil {
			t.Fatalf("call %d unexpected error: %v", i, err)
		}
		if !allowed {
			t.Fatalf("call %d unexpected disallow while draining", i)
		}
	}
	if allowed, err := hit(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if allowed {
		t.Fatal("expected disallowed once max requests is drained")
	}

	// move fakeNow past the window entirely — oldest timestamps should fall out
	fakeNow += 11

	if allowed, err := hit(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if !allowed {
		t.Fatal("expected allowed once window has fully slid past old requests")
	}
}

func TestRollingWindow_partial_slide(t *testing.T) {
	var fakeNow int64 = 1000
	cache := newMockCache()
	store := &InMemoryRollingWindowStore{Cache: cache}
	rw := &RollingWindowLimiter{
		WindowSize:  10,
		MaxRequests: 5,
		Store:       store,
		Now:         func() int64 { return fakeNow },
	}
	rw.Configure()

	ctx := context.Background()
	hit := func() (bool, error) {
		currentTime := rw.Now()
		windowStart := currentTime - rw.WindowSize
		return rw.Store.CheckAndIncrement(ctx, "user-123", windowStart, rw.WindowSize, currentTime, rw.MaxRequests)
	}

	for i := range 5 {
		allowed, err := hit()
		if err != nil {
			t.Fatalf("call %d unexpected error: %v", i, err)
		}
		if !allowed {
			t.Fatalf("call %d unexpected disallow while draining", i)
		}
	}

	fakeNow += 1

	if allowed, err := hit(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if allowed {
		t.Fatal("expected still disallowed — window has not slid past prior requests yet")
	}
}
