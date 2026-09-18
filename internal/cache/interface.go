package cache

import (
"context"
"time"
)

// Key representa una clave de caché estructurada
type Key struct {
Version   string
TenantID  string
ProjectID string
Model     string
Provider  string
Hash      string
}

// String devuelve la clave serializada
func (k Key) String() string {
return k.Version + ":" + k.TenantID + ":" + k.ProjectID + ":" +
k.Model + ":" + k.Provider + ":" + k.Hash
}

// Value es el valor cacheado
type Value struct {
Data      []byte    `json:"data"`
Provider  string    `json:"provider"`
Model     string    `json:"model"`
ExpiresAt time.Time `json:"expires_at"`
CreatedAt time.Time `json:"created_at"`
}

// IsExpired verifica si el valor expiró
func (v *Value) IsExpired() bool {
return time.Now().After(v.ExpiresAt)
}

// Layer es una capa individual de caché
type Layer interface {
Name() string
Get(ctx context.Context, key Key) (*Value, bool, error)
Set(ctx context.Context, key Key, value *Value, ttl time.Duration) error
Delete(ctx context.Context, key Key) error
Clear(ctx context.Context, prefix string) error
}

// Interface es la fachada multi-capa
type Interface interface {
Get(ctx context.Context, key Key) (*Value, string, bool)
Set(ctx context.Context, key Key, value *Value, ttl time.Duration)
Delete(ctx context.Context, key Key)
}
