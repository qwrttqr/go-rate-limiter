package cache

type Cache interface {
	Store(key string, value any)
	Get(key string) (any, error)
	LoadOrStore(key string, create func() any) any
	Delete(key string)
	Close() error
}
