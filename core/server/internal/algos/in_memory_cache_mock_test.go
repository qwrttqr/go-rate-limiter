package algos

import (
	"errors"
	"sync"
)

type inMemoryCacheMock struct {
	mu    sync.Mutex
	calls []string
	store map[string]any
}

func int64Ptr(v int64) *int64 { return &v }

func newMockCache() *inMemoryCacheMock {
	return &inMemoryCacheMock{store: make(map[string]any)}
}

func (m *inMemoryCacheMock) LoadOrStore(key string, value any) any {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, key)
	if v, ok := m.store[key]; ok {
		return v
	}
	m.store[key] = value
	return value
}

func (m *inMemoryCacheMock) Store(key string, value any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, key)
	m.store[key] = value
}

func (m *inMemoryCacheMock) Get(key string) (any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, key)
	v, ok := m.store[key]
	if !ok {
		return nil, errors.New("key not found")
	}
	return v, nil
}

func (m *inMemoryCacheMock) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, key)
	delete(m.store, key)
}
