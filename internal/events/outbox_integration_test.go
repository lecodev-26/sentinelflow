package events

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// TestOutbox_EndToEnd prueba el flujo completo:
//
//	EnqueueTx (dentro de una TX) -> Outbox.Run -> RedisBus.Publish -> Consumer recibe
//
// Requiere Supabase + Redis locales corriendo.
func TestOutbox_EndToEnd(t *testing.T) {
	dbURL := os.Getenv("SENTINELFLOW_DATABASE_URL")
	if dbURL == "" {
		t.Skip("SENTINELFLOW_DATABASE_URL not set")
	}
	redisURL := os.Getenv("SENTINELFLOW_REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// --- Pool Postgres ---
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("pg pool: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}

	// --- Redis + Bus ---
	ropt, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatalf("redis url: %v", err)
	}
	rdb := redis.NewClient(ropt)
	defer rdb.Close()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis unavailable: %v", err)
	}

	bus := NewRedisBus(rdb)
	if err := bus.Start(ctx); err != nil {
		t.Fatalf("bus start: %v", err)
	}
	defer bus.Stop()

	// --- Consumer ---
	received := make(chan Event, 1)
	bus.Subscribe("usage.recorded", NewConsumer("test-consumer", func(_ context.Context, ev Event) {
		select {
		case received <- ev:
		default:
		}
	}))
	time.Sleep(200 * time.Millisecond) // dar tiempo a la suscripción

	// --- Enqueue dentro de una TX ---
	ev := NewEvent("usage.recorded").
		WithTenant("tenant-e2e").
		WithProject("proj-e2e").
		WithPayload("tokens", float64(42))

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	outbox := NewOutbox(pool, bus, DefaultOutboxConfig())
	if err := outbox.EnqueueTx(ctx, tx, ev); err != nil {
		tx.Rollback(ctx)
		t.Fatalf("enqueue: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	// --- Arrancar publisher ---
	pubCtx, pubCancel := context.WithCancel(ctx)
	defer pubCancel()
	go outbox.Run(pubCtx)

	// --- Esperar evento ---
	select {
	case got := <-received:
		if got.ID != ev.ID {
			t.Errorf("id mismatch: got %s want %s", got.ID, ev.ID)
		}
		if got.TenantID != "tenant-e2e" {
			t.Errorf("tenant mismatch: got %q", got.TenantID)
		}
		if got.ProjectID != "proj-e2e" {
			t.Errorf("project mismatch: got %q", got.ProjectID)
		}
		t.Logf("✅ evento recibido: id=%s type=%s tenant=%s", got.ID, got.Type, got.TenantID)
	case <-time.After(8 * time.Second):
		t.Fatal("timeout: evento no llegó al consumer")
	}

	// --- Verificar estado en DB ---
	time.Sleep(300 * time.Millisecond)
	var status string
	if err := pool.QueryRow(ctx,
		"SELECT status FROM outbox_events WHERE event_id = $1", ev.ID,
	).Scan(&status); err != nil {
		t.Fatalf("query status: %v", err)
	}
	if status != "published" {
		t.Errorf("status en DB: got %q want %q", status, "published")
	}
	t.Logf("✅ estado final en DB: %s", status)
}

// TestOutbox_TxRollbackNoEvent verifica que si la TX hace rollback,
// el evento NO se persiste (garantía del outbox).
func TestOutbox_TxRollbackNoEvent(t *testing.T) {
	dbURL := os.Getenv("SENTINELFLOW_DATABASE_URL")
	if dbURL == "" {
		t.Skip("SENTINELFLOW_DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("pg pool: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}

	ev := NewEvent("usage.recorded").WithTenant("tenant-rollback")
	bus := NewInMemoryBus()
	outbox := NewOutbox(pool, bus, DefaultOutboxConfig())

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := outbox.EnqueueTx(ctx, tx, ev); err != nil {
		tx.Rollback(ctx)
		t.Fatalf("enqueue: %v", err)
	}
	tx.Rollback(ctx) // rollback deliberado

	// Verificar que NO existe
	var count int
	if err := pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM outbox_events WHERE event_id = $1", ev.ID,
	).Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 0 {
		t.Errorf("evento persistido tras rollback: count=%d", count)
	}
	t.Logf("✅ rollback correcto: 0 filas")
}
