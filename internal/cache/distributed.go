package cache

import (
"context"
"crypto/sha256"
"encoding/hex"
"time"
)

type DistributedCache struct {
redis  *RedisClient
prefix string
}

func NewDistributedCache(redis *RedisClient, prefix string) *DistributedCache {
return &DistributedCache{
redis:  redis,
prefix: prefix,
}
}

func (c *DistributedCache) key(k string) string {
hash := sha256.Sum256([]byte(k))
return c.prefix + ":" + hex.EncodeToString(hash[:16])
}

func (c *DistributedCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
return c.redis.Set(ctx, c.key(key), value, ttl)
}

func (c *DistributedCache) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
err := c.redis.Get(ctx, c.key(key), dest)
if err != nil {
if err.Error() == "redis: nil" {
return false, nil
}
return false, err
}
return true, nil
}

func (c *DistributedCache) Delete(ctx context.Context, key string) error {
return c.redis.Delete(ctx, c.key(key))
}

func (c *DistributedCache) Exists(ctx context.Context, key string) (bool, error) {
return c.redis.Exists(ctx, c.key(key))
}

func (c *DistributedCache) Incr(ctx context.Context, key string) (int64, error) {
return c.redis.Incr(ctx, c.key(key))
}

// Key devuelve la clave completa (para expirar)
func (c *DistributedCache) Key(k string) string {
return c.key(k)
}

// Expire establece TTL en una clave
func (c *DistributedCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
return c.redis.Expire(ctx, c.key(key), ttl)
}
