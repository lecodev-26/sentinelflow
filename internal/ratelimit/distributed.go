package ratelimit

import (
"context"
"time"

"github.com/lecodev-26/sentinelflow/internal/cache"
)

type DistributedLimiter struct {
cache  *cache.DistributedCache
limit  int
window time.Duration
prefix string
}

func NewDistributedLimiter(c *cache.DistributedCache, limit int, window time.Duration, prefix string) *DistributedLimiter {
return &DistributedLimiter{
cache:  c,
limit:  limit,
window: window,
prefix: prefix,
}
}

func (l *DistributedLimiter) Allow(ctx context.Context, ip string) (bool, error) {
key := l.prefix + ":ratelimit:" + ip

count, err := l.cache.Incr(ctx, key)
if err != nil {
return false, err
}

if count == 1 {
err = l.cache.Expire(ctx, key, l.window)
if err != nil {
return false, err
}
}

if count > int64(l.limit) {
return false, nil
}

return true, nil
}
