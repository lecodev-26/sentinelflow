package events

import (
	"context"
	"sync"

	"github.com/lecodev-26/sentinelflow/internal/logger"
)

// InMemoryBus es un EventBus in-process para desarrollo y tests.
//
// Limitaciones:
//   - Eventos no persisten entre reinicios
//   - No funciona con múltiples gateways (cada uno tiene su bus)
//   - Sin garantía de entrega
//
// Para producción, usa RedisBus o NATSBus.
type InMemoryBus struct {
	mu          sync.RWMutex
	subscribers map[string][]Consumer
	started     bool
	stopCh      chan struct{}
}

// NewInMemoryBus crea un nuevo bus in-memory
func NewInMemoryBus() *InMemoryBus {
	return &InMemoryBus{
		subscribers: make(map[string][]Consumer),
		stopCh:      make(chan struct{}),
	}
}

// Start arranca el bus (no-op para InMemory)
func (b *InMemoryBus) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.started = true
	logger.Info("📡 InMemoryBus iniciado")
	return nil
}

// Stop cierra el bus
func (b *InMemoryBus) Stop() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.started {
		return nil
	}

	close(b.stopCh)
	b.started = false
	logger.Info("📡 InMemoryBus detenido")
	return nil
}

// Publish emite un evento a todos los subscribers de su tipo
func (b *InMemoryBus) Publish(ctx context.Context, event Event) error {
	b.mu.RLock()
	consumers := b.subscribers[event.Type]
	b.mu.RUnlock()

	if len(consumers) == 0 {
		// Sin subscribers: silencioso
		return nil
	}

	// Invocar cada handler en su propia goroutine
	for _, consumer := range consumers {
		c := consumer // captura
		go func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Errorf("❌ Panic in consumer %s: %v", c.Name, r)
				}
			}()
			c.Handler(ctx, event)
		}()
	}

	return nil
}

// Subscribe registra un consumidor para un tipo de evento
func (b *InMemoryBus) Subscribe(eventType string, consumer Consumer) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers[eventType] = append(b.subscribers[eventType], consumer)
	logger.Infof("📡 [InMemory] Consumer '%s' subscribed to '%s'", consumer.Name, eventType)
	return nil
}

// Unsubscribe elimina todos los handlers de un tipo de evento
func (b *InMemoryBus) Unsubscribe(eventType string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.subscribers, eventType)
}

// SubscriberCount devuelve el número de subscribers de un tipo de evento
func (b *InMemoryBus) SubscriberCount(eventType string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers[eventType])
}

// TypeCount devuelve el número de tipos de eventos con subscribers
func (b *InMemoryBus) TypeCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}
