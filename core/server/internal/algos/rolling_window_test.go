package algos

import (
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
			rw := &RollingWindowLimiter{
				WindowSize:  tt.windowSize,
				MaxRequests: tt.maxRequests,
				Cache:       cache,
				StorageType: "in_memory",
			}
			rw.Configure()
			calls := int(tt.maxRequests)

			for i := 0; i <= calls; i++ {
				allowed := rw.limitingHook(tt.key)
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
	fw := &FixedWindowLimiter{
		WindowSize:  10,
		MaxRequests: 5,
		Cache:       cache,
		StorageType: "in_memory",
		Now:         func() int64 { return fakeNow },
	}
	fw.Configure()

	if !fw.limitingHook("user-123") {
		t.Fatal("expected disallowed when max requests is drained")
	}

	fakeNow += 1

	for i := range 4 {
		if !fw.limitingHook("user-123") {
			t.Fatalf("call %d unexpected disallow while draining", i)
		}
	}
	if fw.limitingHook("user-123") {
		t.Fatal("expected disallowed when max requests is drained")
	}

	fakeNow += 10

	if !fw.limitingHook("user-123") {
		t.Fatal("expected disallowed when max requests is drained")
	}
}

func TestRollingWindow_same_window(t *testing.T) {
	var fakeNow int64 = 1000
	cache := newMockCache()
	fw := &FixedWindowLimiter{
		WindowSize:  10,
		MaxRequests: 5,
		Cache:       cache,
		StorageType: "in_memory",
		Now:         func() int64 { return fakeNow },
	}
	fw.Configure()

	for i := range 5 {
		if !fw.limitingHook("user-123") {
			t.Fatalf("call %d unexpected disallow while draining", i)
		}
	}
	if fw.limitingHook("user-123") {
		t.Fatal("expected disallowed when max requests is drained")
	}

	fakeNow += 1

	for i := range 5 {
		if fw.limitingHook("user-123") {
			t.Fatalf("call %d unexpected disallow while draining", i)
		}
	}
}
