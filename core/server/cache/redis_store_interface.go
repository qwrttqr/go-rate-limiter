package cache

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type RedisStoreInterface interface {
	RunLuaScript(ctx context.Context, script *redis.Script, keys []string, args ...any) (bool, int64, error)
	DefaultTtl() int64
}
