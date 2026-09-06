package algos

import (
	"context"
	"testing"
)

func TestFixedWindow_logic(t *testing.T) {
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
			store := &InMemoryFixedWindowStore{Cache: cache}
			fw := &FixedWindowLimiter{
				WindowSize:  tt.windowSize,
				MaxRequests: tt.maxRequests,
				Store:       store,
			}
			fw.Configure()
			calls := int(tt.maxRequests)

			for i := 0; i <= calls; i++ {
				currentTime := fw.Now()
				currentWindow := currentTime / fw.WindowSize
				windowStart := currentWindow * fw.WindowSize

				allowed, err := fw.Store.CheckAndIncrement(context.Background(), tt.key, windowStart, fw.WindowSize, fw.MaxRequests)
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

func TestFixedWindow_new_window(t *testing.T) {
	var fakeNow int64 = 1000
	cache := newMockCache()
	store := &InMemoryFixedWindowStore{Cache: cache}
	fw := &FixedWindowLimiter{
		WindowSize:  10,
		MaxRequests: 5,
		Store:       store,
		Now:         func() int64 { return fakeNow },
	}
	fw.Configure()

	ctx := context.Background()
	hit := func() (bool, error) {
		currentTime := fw.Now()
		currentWindow := currentTime / fw.WindowSize
		windowStart := currentWindow * fw.WindowSize
		return fw.Store.CheckAndIncrement(ctx, "user-123", windowStart, fw.WindowSize, fw.MaxRequests)
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
		t.Fatal("expected disallowed when max requests is drained")
	}

	fakeNow += 10 // moves into the next 10s-aligned bucket entirely

	for i := range 5 {
		allowed, err := hit()
		if err != nil {
			t.Fatalf("call %d unexpected error: %v", i, err)
		}
		if !allowed {
			t.Fatalf("call %d unexpected disallow in new window", i)
		}
	}
}

func TestFixedWindow_same_window(t *testing.T) {
	var fakeNow int64 = 1000
	cache := newMockCache()
	store := &InMemoryFixedWindowStore{Cache: cache}
	fw := &FixedWindowLimiter{
		WindowSize:  10,
		MaxRequests: 5,
		Store:       store,
		Now:         func() int64 { return fakeNow },
	}
	fw.Configure()

	ctx := context.Background()
	hit := func() (bool, error) {
		currentTime := fw.Now()
		currentWindow := currentTime / fw.WindowSize
		windowStart := currentWindow * fw.WindowSize
		return fw.Store.CheckAndIncrement(ctx, "user-123", windowStart, fw.WindowSize, fw.MaxRequests)
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
		t.Fatal("expected disallowed when max requests is drained")
	}

	fakeNow += 1 // still within the same 10s-aligned bucket (1000-1009)

	for i := range 5 {
		allowed, err := hit()
		if err != nil {
			t.Fatalf("call %d unexpected error: %v", i, err)
		}
		if allowed {
			t.Fatalf("call %d unexpected allow — still same window, should stay disallowed", i)
		}
	}
}
