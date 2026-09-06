package cache

import (
	"sync"
	"time"
)

// Item representa un elemento en caché
type Item struct {
	Value      interface{}
	Expiration time.Time
}

// IsExpired verifica si el elemento ha expirado
func (i *Item) IsExpired() bool {
	return time.Now().After(i.Expiration)
}

// Cache es la estructura principal de caché en memoria
type Cache struct {
	items map[string]*Item
	mu    sync.RWMutex
	ttl   time.Duration
}

// NewCache crea una nueva instancia de caché
func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		items: make(map[string]*Item),
		ttl:   ttl,
	}
}

// Get obtiene un valor de la caché
func (c *Cache) Get(key string) (interface{}, bool) {
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

// Set guarda un valor en la caché
func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &Item{
		Value:      value,
		Expiration: time.Now().Add(c.ttl),
	}
}

// Delete elimina un elemento de la caché
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Clear vacía toda la caché
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*Item)
}

// Size retorna el número de elementos en caché
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// CleanExpired elimina elementos expirados
func (c *Cache) CleanExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, item := range c.items {
		if item.IsExpired() {
			delete(c.items, key)
		}
	}
<<<<<<< HEAD
}
=======
}
>>>>>>> b011fca1f048d402212fded110a6fddb45a10859
