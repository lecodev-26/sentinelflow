package cache

import (
"context"
"sync"
"time"
)

type Item struct {
Value      interface{}
Expiration time.Time
}

func (i *Item) IsExpired() bool {
return time.Now().After(i.Expiration)
}

type Cache struct {
items map[string]*Item
mu    sync.RWMutex
ttl   time.Duration
redis *DistributedCache
}

func NewCache(ttl time.Duration) *Cache {
return &Cache{
items: make(map[string]*Item),
ttl:   ttl,
}
}

func NewCacheWithRedis(redis *DistributedCache, ttl time.Duration) *Cache {
return &Cache{
items: make(map[string]*Item),
ttl:   ttl,
redis: redis,
}
}

func (c *Cache) Get(key string) (interface{}, bool) {
if c.redis != nil {
var value interface{}
ctx := context.Background()
found, err := c.redis.Get(ctx, key, &value)
if err == nil && found {
return value, true
}
}

c.mu.RLock()
defer c.mu.RUnlock()

item, exists := c.items[key]
if !exists {
return nil, false
}

if item.IsExpired() {
delete(c.items, key)
return nil, false
}

return item.Value, true
}

func (c *Cache) Set(key string, value interface{}) {
if c.redis != nil {
ctx := context.Background()
_ = c.redis.Set(ctx, key, value, c.ttl)
}

c.mu.Lock()
defer c.mu.Unlock()

c.items[key] = &Item{
Value:      value,
Expiration: time.Now().Add(c.ttl),
}
}

func (c *Cache) Delete(key string) {
if c.redis != nil {
ctx := context.Background()
_ = c.redis.Delete(ctx, key)
}

c.mu.Lock()
defer c.mu.Unlock()
delete(c.items, key)
}

func (c *Cache) Clear() {
c.mu.Lock()
defer c.mu.Unlock()
c.items = make(map[string]*Item)
}

func (c *Cache) Size() int {
c.mu.RLock()
defer c.mu.RUnlock()
return len(c.items)
}

func (c *Cache) CleanExpired() {
c.mu.Lock()
defer c.mu.Unlock()

for key, item := range c.items {
if item.IsExpired() {
delete(c.items, key)
}
}
}
