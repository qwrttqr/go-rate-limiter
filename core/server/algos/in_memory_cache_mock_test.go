package algos

import (
	"sync"
)

type shardedInMemoryCacheMock struct {
	mu      sync.Mutex
	entries map[string]any
}

func newMockCache() *shardedInMemoryCacheMock {
	return &shardedInMemoryCacheMock{entries: make(map[string]any)}
}

func (m *shardedInMemoryCacheMock) Update(key string, createFn func() any, fn func(v any) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, present := m.entries[key]
	if !present {
		value = createFn()
		m.entries[key] = value
	}
	return fn(value)
}

func (m *shardedInMemoryCacheMock) Close() error {
	return nil
}
