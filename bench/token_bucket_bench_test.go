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

func benchTokenBucket(
	b *testing.B,
	store algos.TokenBucketStore,
	nKeys int,
	zipf bool,
	rate float64,
	capacity int64,
	tokensRequired int64,
	wantAllowed bool,
) {
	keys := make([]string, nKeys)
	for i := range keys {
		keys[i] = "client-" + strconv.Itoa(i)
	}
	seq := keySequence(nKeys, zipf)

	ctx := context.Background()
	now := time.Now().Unix()

	if !wantAllowed {
		for _, k := range keys {
			drain := capacity / tokensRequired
			for range drain {
				store.TakeToken(ctx, k, rate, tokensRequired, now, capacity)
			}
		}
	}

	for _, k := range keys {
		allowed, _, err := store.TakeToken(ctx, k, rate, tokensRequired, now, capacity)
		if err != nil || allowed != wantAllowed {
			b.Fatalf("wrong path: allowed=%v err=%v", allowed, err)
		}
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		pos := rand.IntN(seqLen)
		for pb.Next() {
			k := keys[seq[pos&(seqLen-1)]]
			store.TakeToken(ctx, k, rate, tokensRequired, now, capacity)
			pos++
		}
	})
}

func BenchmarkTokenBucket(b *testing.B) {
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
		name           string
		rate           float64
		capacity       int64
		tokensRequired int64
		wantAllowed    bool
	}{
		{"allow", 0, 2 << 30, 1, true},
		{"deny", 0, 10, 1, false},
	}

	for _, w := range workloads {
		for _, p := range paths {
			b.Run(w.name+"/"+p.name, func(b *testing.B) {
				c := cache.NewCache(60, time.Minute, 256)
				b.Cleanup(func() { _ = c.Close() })
				store := &algos.InMemoryBucketStore{Cache: c}
				benchTokenBucket(b, store, w.nKeys, w.zipf, p.rate, p.capacity, p.tokensRequired, p.wantAllowed)
			})
		}
	}
}
