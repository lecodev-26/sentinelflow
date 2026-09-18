package cache

import (
"context"
"sync"
"time"
)

// L1Memory es la caché en memoria
type L1Memory struct {
mu    sync.RWMutex
items map[string]*Value
}

// NewL1Memory crea una nueva caché L1
func NewL1Memory() *L1Memory {
l1 := &L1Memory{
items: make(map[string]*Value),
}
go l1.cleanup()
return l1
}

func (l *L1Memory) Name() string { return "L1-memory" }

func (l *L1Memory) Get(ctx context.Context, key Key) (*Value, bool, error) {
l.mu.RLock()
v, exists := l.items[key.String()]
l.mu.RUnlock()

if !exists {
return nil, false, nil
}
if v.IsExpired() {
l.Delete(ctx, key)
return nil, false, nil
}
return v, true, nil
}

func (l *L1Memory) Set(ctx context.Context, key Key, value *Value, ttl time.Duration) error {
l.mu.Lock()
defer l.mu.Unlock()

if value.CreatedAt.IsZero() {
value.CreatedAt = time.Now()
}
value.ExpiresAt = time.Now().Add(ttl)
l.items[key.String()] = value
return nil
}

func (l *L1Memory) Delete(ctx context.Context, key Key) error {
l.mu.Lock()
defer l.mu.Unlock()
delete(l.items, key.String())
return nil
}

func (l *L1Memory) Clear(ctx context.Context, prefix string) error {
l.mu.Lock()
defer l.mu.Unlock()

for k := range l.items {
if len(prefix) == 0 || (len(k) >= len(prefix) && k[:len(prefix)] == prefix) {
delete(l.items, k)
}
}
return nil
}

// Size devuelve el número de entradas
func (l *L1Memory) Size() int {
l.mu.RLock()
defer l.mu.RUnlock()
return len(l.items)
}

// cleanup elimina entradas expiradas cada 5 minutos
func (l *L1Memory) cleanup() {
ticker := time.NewTicker(5 * time.Minute)
defer ticker.Stop()

for range ticker.C {
l.mu.Lock()
for k, v := range l.items {
if v.IsExpired() {
delete(l.items, k)
}
}
l.mu.Unlock()
}
}
