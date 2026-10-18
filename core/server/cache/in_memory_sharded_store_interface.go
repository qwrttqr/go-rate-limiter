package cache

type InMemoryStoreInterface interface {
	Update(key string, newFn func() any, fn func(v any) error) error
	Close() error
}
