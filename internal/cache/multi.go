package cache

import (
"context"
"time"
)

// MultiCache combina L1, L2 y L3
type MultiCache struct {
l1 *L1Memory
l2 *L2Redis
l3 *L3Semantic
}

// NewMultiCache crea la caché multi-capa
func NewMultiCache(l1 *L1Memory, l2 *L2Redis, l3 *L3Semantic) *MultiCache {
return &MultiCache{l1: l1, l2: l2, l3: l3}
}

// Get busca en L1 → L2 (popula L1 si HIT)
func (m *MultiCache) Get(ctx context.Context, key Key) (*Value, string, bool) {
// L1
if m.l1 != nil {
if v, ok, _ := m.l1.Get(ctx, key); ok {
return v, "L1", true
}
}
// L2
if m.l2 != nil {
if v, ok, _ := m.l2.Get(ctx, key); ok {
// Poblar L1
if m.l1 != nil {
_ = m.l1.Set(ctx, key, v, time.Until(v.ExpiresAt))
}
return v, "L2", true
}
}
return nil, "", false
}

// GetSemantic busca en L3
func (m *MultiCache) GetSemantic(ctx context.Context, text string) (*Value, bool) {
if m.l3 == nil {
return nil, false
}
return m.l3.GetByText(ctx, text)
}

// Set guarda en L1 y L2
func (m *MultiCache) Set(ctx context.Context, key Key, value *Value, ttl time.Duration) {
if m.l1 != nil {
_ = m.l1.Set(ctx, key, value, ttl)
}
if m.l2 != nil {
_ = m.l2.Set(ctx, key, value, ttl)
}
}

// SetSemantic guarda en L3
func (m *MultiCache) SetSemantic(ctx context.Context, text string, value *Value, ttl time.Duration) {
if m.l3 != nil {
_ = m.l3.SetByText(ctx, text, value, ttl)
}
}

// Delete elimina de todas las capas
func (m *MultiCache) Delete(ctx context.Context, key Key) {
if m.l1 != nil {
_ = m.l1.Delete(ctx, key)
}
if m.l2 != nil {
_ = m.l2.Delete(ctx, key)
}
}

// Stats devuelve estadísticas
func (m *MultiCache) Stats() map[string]interface{} {
stats := map[string]interface{}{}
if m.l1 != nil {
stats["l1_size"] = m.l1.Size()
}
if m.l3 != nil {
stats["l3_size"] = m.l3.Size()
}
return stats
}
