package interfaces

type Cache interface {
	Store(key string, value any)
	Get(key string) (any, error)
	LoadOrStore(key string, value any) any
	Delete(key string)
}
