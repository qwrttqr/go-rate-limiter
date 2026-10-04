package bench

import (
	"context"
	"math/rand/v2"
	"strconv"
	"testing"
	"time"

	"qwrttqr-rate-limiter/core/server/algos"
	"qwrttqr-rate-limiter/core/server/cache"
)

func benchFixedWindow(b *testing.B, store algos.FixedWindowStore, nKeys int, zipf bool, maxRequests int64, wantAllowed bool) {
	keys := make([]string, nKeys)
	for i := range keys {
		keys[i] = "client-" + strconv.Itoa(i)
	}
	seq := keySequence(nKeys, zipf)

	ctx := context.Background()
	now := time.Now().Unix()

	for _, k := range keys {
		allowed, _, err := store.CheckAndIncrement(ctx, k, windowSize, maxRequests, now)
		if err != nil || allowed != wantAllowed {
			b.Fatalf("wrong path: allowed=%v err=%v", allowed, err)
		}
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		pos := rand.IntN(seqLen)
		for pb.Next() {
			k := keys[seq[pos&(seqLen-1)]] // same as keys[seq[pos % seqLen]]]
			store.CheckAndIncrement(ctx, k, windowSize, maxRequests, now)
			pos++
		}
	})
}

func BenchmarkFixedWindow(b *testing.B) {
	workloads := []struct {
		name  string
		nKeys int
		zipf  bool
	}{
		{"single_key", 1, false},
		{"uniform_100k", 100_000, false},
		{"zipf_100k", 100_000, true},
	}
	paths := []struct {
		name        string
		maxRequests int64
		wantAllowed bool
	}{
		{"allow", 2 << 30, true},
		{"deny", 0, false},
	}

	for _, w := range workloads {
		for _, p := range paths {
			b.Run(w.name+"/"+p.name, func(b *testing.B) {
				c := cache.NewCache(60, time.Minute, 256)
				b.Cleanup(func() { _ = c.Close() })
				store := &algos.InMemoryFixedWindowStore{Cache: c}
				benchFixedWindow(b, store, w.nKeys, w.zipf, p.maxRequests, p.wantAllowed)
			})
		}
	}
}
