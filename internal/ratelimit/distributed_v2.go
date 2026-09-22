package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// DistributedLimiterV2 usa Redis con sliding window
type DistributedLimiterV2 struct {
	client *redis.Client
	prefix string
}

// NewDistributedLimiterV2 crea un limiter distribuido
func NewDistributedLimiterV2(client *redis.Client, prefix string) *DistributedLimiterV2 {
	if prefix == "" {
		prefix = "sf:ratelimit"
	}
	return &DistributedLimiterV2{client: client, prefix: prefix}
}

// Allow verifica si se permite una petición
// Implementa sliding window con Redis sorted sets
func (l *DistributedLimiterV2) Allow(ctx context.Context, identifier string, limit int, window time.Duration) (bool, int, error) {
	key := fmt.Sprintf("%s:%s", l.prefix, identifier)
	now := time.Now().UnixNano()
	windowStart := now - window.Nanoseconds()

	// Pipeline: eliminar antiguos + contar + añadir actual
	pipe := l.client.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))
	countCmd := pipe.ZCard(ctx, key)
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})
	pipe.Expire(ctx, key, window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, 0, err
	}

	count := countCmd.Val()
	if count >= int64(limit) {
		return false, int(count), nil
	}
	return true, int(count), nil
}

// Reset limpia el contador de un identificador
func (l *DistributedLimiterV2) Reset(ctx context.Context, identifier string) error {
	key := fmt.Sprintf("%s:%s", l.prefix, identifier)
	return l.client.Del(ctx, key).Err()
}
