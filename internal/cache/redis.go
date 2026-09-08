package cache

import (
"context"
"encoding/json"
"time"

"github.com/redis/go-redis/v9"
)

// RedisClient es un wrapper para Redis
type RedisClient struct {
client *redis.Client
}

// NewRedisClient crea un nuevo cliente de Redis
func NewRedisClient(addr, password string, db int) *RedisClient {
client := redis.NewClient(&redis.Options{
Addr:     addr,
Password: password,
DB:       db,
})

return &RedisClient{client: client}
}

// NewRedisClientFromURL crea un cliente desde una URL
func NewRedisClientFromURL(url string) (*RedisClient, error) {
opt, err := redis.ParseURL(url)
if err != nil {
return nil, err
}

client := redis.NewClient(opt)
return &RedisClient{client: client}, nil
}

// Set guarda un valor en Redis
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
data, err := json.Marshal(value)
if err != nil {
return err
}

return r.client.Set(ctx, key, data, ttl).Err()
}

// Get obtiene un valor de Redis
func (r *RedisClient) Get(ctx context.Context, key string, dest interface{}) error {
data, err := r.client.Get(ctx, key).Bytes()
if err != nil {
return err
}

return json.Unmarshal(data, dest)
}

// Delete elimina una clave de Redis
func (r *RedisClient) Delete(ctx context.Context, key string) error {
return r.client.Del(ctx, key).Err()
}

// Exists verifica si una clave existe
func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
result, err := r.client.Exists(ctx, key).Result()
if err != nil {
return false, err
}
return result > 0, nil
}

// Incr incrementa un contador en Redis
func (r *RedisClient) Incr(ctx context.Context, key string) (int64, error) {
return r.client.Incr(ctx, key).Result()
}

// IncrBy incrementa un contador en Redis por un valor específico
func (r *RedisClient) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
return r.client.IncrBy(ctx, key, value).Result()
}

// Expire establece TTL en una clave
func (r *RedisClient) Expire(ctx context.Context, key string, ttl time.Duration) error {
return r.client.Expire(ctx, key, ttl).Err()
}

// Ping verifica la conexión a Redis
func (r *RedisClient) Ping(ctx context.Context) error {
return r.client.Ping(ctx).Err()
}

// Close cierra la conexión a Redis
func (r *RedisClient) Close() error {
return r.client.Close()
}

// GetClient devuelve el cliente raw de Redis
func (r *RedisClient) GetClient() *redis.Client {
return r.client
}
