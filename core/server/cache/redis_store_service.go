package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client     *redis.Client
	defaultTtl int64
}

func NewRedisStore(client *redis.Client, defaultTtl int64) *RedisStore {
	return &RedisStore{client: client, defaultTtl: defaultTtl}
}

// RunLuaScript runs a Lua script that returns {allowed(0|1), retryAfter}.
func (r *RedisStore) RunLuaScript(
	ctx context.Context,
	script *redis.Script,
	keys []string,
	args ...any,
) (bool, int64, error) {
	res, err := script.Run(ctx, r.client, keys, args...).Int64Slice()
	if err != nil {
		return false, 0, err
	}
	if len(res) != 2 {
		return false, 0, fmt.Errorf("unexpected length of script output")
	}
	return res[0] == 1, res[1], nil
}
