package algos

import (
	"errors"
	"sync"
)

type shardedInMemoryCacheMock struct {
	mu    sync.Mutex
	calls []string
	store map[string]any
}

func newMockCache() *shardedInMemoryCacheMock {
	return &shardedInMemoryCacheMock{store: make(map[string]any)}
}

func (m *shardedInMemoryCacheMock) LoadOrStore(key string, create func() any) any {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, key)
	if v, ok := m.store[key]; ok {
		return v
	}
	v := create()
	m.store[key] = v
	return v
}

func (m *shardedInMemoryCacheMock) Store(key string, value any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, key)
	m.store[key] = value
}

func (m *shardedInMemoryCacheMock) Get(key string) (any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, key)
	v, ok := m.store[key]
	if !ok {
		return nil, errors.New("key not found")
	}
	return v, nil
}

func (m *shardedInMemoryCacheMock) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, key)
	delete(m.store, key)
}

func (m *shardedInMemoryCacheMock) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, "close")
}
