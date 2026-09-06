package ratelimit

import (
	"sync"
	"time"
)

// Limiter controla el rate limiting por IP
type Limiter struct {
	mu       sync.RWMutex
	visitors map[string]*Visitor
	limit    int           // Peticiones permitidas
	window   time.Duration // Ventana de tiempo
}

// Visitor representa un cliente individual
type Visitor struct {
	mu       sync.Mutex
	count    int
	lastSeen time.Time
	reset    time.Time
}

// NewLimiter crea un nuevo rate limiter
func NewLimiter(limit int, window time.Duration) *Limiter {
	l := &Limiter{
		visitors: make(map[string]*Visitor),
		limit:    limit,
		window:   window,
	}
	go l.cleanup()
	return l
}

// Allow verifica si una IP puede hacer una petición
func (l *Limiter) Allow(ip string) bool {
	l.mu.Lock()
	v, exists := l.visitors[ip]
	if !exists {
		v = &Visitor{
			count:    1,
			lastSeen: time.Now(),
			reset:    time.Now().Add(l.window),
		}
		l.visitors[ip] = v
		l.mu.Unlock()
		return true
	}
	l.mu.Unlock()

	v.mu.Lock()
	defer v.mu.Unlock()

	// Si pasó la ventana, resetear
	if time.Now().After(v.reset) {
		v.count = 1
		v.reset = time.Now().Add(l.window)
		return true
	}

	// Verificar límite
	if v.count >= l.limit {
		return false
	}

	v.count++
	v.lastSeen = time.Now()
	return true
}

// cleanup elimina visitantes antiguos
func (l *Limiter) cleanup() {
	for {
		time.Sleep(10 * time.Minute)
		l.mu.Lock()
		for ip, v := range l.visitors {
			v.mu.Lock()
			if time.Since(v.lastSeen) > 30*time.Minute {
				delete(l.visitors, ip)
			}
			v.mu.Unlock()
		}
		l.mu.Unlock()
	}
}