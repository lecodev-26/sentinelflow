package cache

import (
"context"
"encoding/json"
"time"

"github.com/redis/go-redis/v9"
)

// L2Redis es la caché Redis
type L2Redis struct {
client *redis.Client
prefix string
}

// NewL2Redis crea una nueva caché L2
func NewL2Redis(client *redis.Client, prefix string) *L2Redis {
if prefix == "" {
prefix = "sf:cache"
}
return &L2Redis{client: client, prefix: prefix}
}

func (l *L2Redis) Name() string { return "L2-redis" }

func (l *L2Redis) key(k Key) string {
return l.prefix + ":" + k.String()
}

func (l *L2Redis) Get(ctx context.Context, key Key) (*Value, bool, error) {
data, err := l.client.Get(ctx, l.key(key)).Bytes()
if err == redis.Nil {
return nil, false, nil
}
if err != nil {
return nil, false, err
}

var v Value
if err := json.Unmarshal(data, &v); err != nil {
return nil, false, err
}
if v.IsExpired() {
return nil, false, nil
}
return &v, true, nil
}

func (l *L2Redis) Set(ctx context.Context, key Key, value *Value, ttl time.Duration) error {
if value.CreatedAt.IsZero() {
value.CreatedAt = time.Now()
}
value.ExpiresAt = time.Now().Add(ttl)

data, err := json.Marshal(value)
if err != nil {
return err
}
return l.client.Set(ctx, l.key(key), data, ttl).Err()
}

func (l *L2Redis) Delete(ctx context.Context, key Key) error {
return l.client.Del(ctx, l.key(key)).Err()
}

func (l *L2Redis) Clear(ctx context.Context, prefix string) error {
pattern := l.prefix + ":" + prefix + "*"
iter := l.client.Scan(ctx, 0, pattern, 100).Iterator()
for iter.Next(ctx) {
if err := l.client.Del(ctx, iter.Val()).Err(); err != nil {
return err
}
}
return iter.Err()
}
