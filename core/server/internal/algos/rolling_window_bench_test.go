package algos

import (
	"context"
	"qwrttqr-rate-limiter/core/server/internal/cache"
	"strconv"
	"testing"
	"time"
)

func BenchmarkCheckAndIncrement(b *testing.B) {
	store := &InMemoryRollingWindowStore{Cache: cache.NewCache(60, time.Minute)}
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := strconv.Itoa(i % 1000)
			store.CheckAndIncrement(context.Background(), key, 0, 60, time.Now().Unix(), 50)
			i++
		}
	})
}
