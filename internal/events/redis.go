package events

import (
	"context"
	"sync"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// RedisBus implementa EventBus sobre Redis Pub/Sub.
//
// Uso en producción:
//   - Múltiples gateways comparten el mismo Redis.
//   - Cada gateway publica y todos los que estén suscritos reciben.
//   - Es un bus "fan-out": todos los subscribers del mismo canal reciben el evento.
//
// Limitaciones:
//   - Sin persistencia: si un subscriber está caído, pierde el evento.
//     Para entrega garantizada, el outbox cubre el lado del publisher;
//     el consumer debe ser idempotente + persistir lo que recibe.
//   - Redis Pub/Sub no es un stream (no hay replay). Para eso, usar Redis Streams
//     o NATS JetStream (V4).
type RedisBus struct {
	client *redis.Client

	mu          sync.RWMutex
	subscribers map[string][]Consumer
	started     bool

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewRedisBus crea un bus Redis. El client se puede obtener con
// cache.RedisClient.GetClient() o construyendo uno con redis.ParseURL.
func NewRedisBus(client *redis.Client) *RedisBus {
	ctx, cancel := context.WithCancel(context.Background())
	return &RedisBus{
		client:      client,
		subscribers: make(map[string][]Consumer),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start comprueba la conexión con Redis.
func (b *RedisBus) Start(ctx context.Context) error {
	if err := b.client.Ping(ctx).Err(); err != nil {
		return err
	}
	b.mu.Lock()
	b.started = true
	b.mu.Unlock()
	logrus.Info("📡 RedisBus iniciado")
	return nil
}

// Stop cierra todas las suscripciones.
func (b *RedisBus) Stop() error {
	b.mu.Lock()
	if !b.started {
		b.mu.Unlock()
		return nil
	}
	b.started = false
	b.mu.Unlock()

	b.cancel()
	b.wg.Wait()
	logrus.Info("📡 RedisBus detenido")
	return b.client.Close()
}

// Publish serializa el evento y lo publica al canal con el nombre del Type.
func (b *RedisBus) Publish(ctx context.Context, event Event) error {
	data, err := event.Marshal()
	if err != nil {
		return err
	}
	return b.client.Publish(ctx, event.Type, data).Err()
}

// Subscribe registra un consumer y arranca una goroutine que escucha el canal.
func (b *RedisBus) Subscribe(eventType string, consumer Consumer) error {
	b.mu.Lock()
	b.subscribers[eventType] = append(b.subscribers[eventType], consumer)
	b.mu.Unlock()

	b.wg.Add(1)
	go b.listen(eventType, consumer)

	logrus.Infof("📡 [Redis] Consumer '%s' subscribed to '%s'", consumer.Name, eventType)
	return nil
}

// Unsubscribe elimina los consumers registrados para un tipo (no cancela el listen).
func (b *RedisBus) Unsubscribe(eventType string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.subscribers, eventType)
}

// listen mantiene una suscripción al canal eventType y reenvía al consumer.
func (b *RedisBus) listen(eventType string, consumer Consumer) {
	defer b.wg.Done()

	pubsub := b.client.Subscribe(b.ctx, eventType)
	defer pubsub.Close()

	ch := pubsub.Channel()
	logrus.Infof("📡 [Redis] listening on '%s' for '%s'", eventType, consumer.Name)

	for {
		select {
		case <-b.ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			ev, err := UnmarshalEvent([]byte(msg.Payload))
			if err != nil {
				logrus.Errorf("📡 [Redis] unmarshal failed on '%s': %v", eventType, err)
				continue
			}
			b.dispatch(*ev, consumer)
		}
	}
}

// dispatch ejecuta el handler con recover para no tumbar el bus.
func (b *RedisBus) dispatch(ev Event, consumer Consumer) {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("❌ Panic in consumer %s: %v", consumer.Name, r)
		}
	}()
	consumer.Handler(b.ctx, ev)
}
