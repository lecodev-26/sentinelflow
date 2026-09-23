package audit_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/lecodev-26/sentinelflow/internal/audit"
	"github.com/lecodev-26/sentinelflow/internal/events"
)

// TestAudit_EndToEnd emite un evento audit al outbox, arranca publisher + consumer,
// y verifica que audit_log recibe el registro.
func TestAudit_EndToEnd(t *testing.T) {
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

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("pg pool: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}

	ropt, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatalf("redis url: %v", err)
	}
	rdb := redis.NewClient(ropt)
	defer rdb.Close()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis unavailable: %v", err)
	}

	bus := events.NewRedisBus(rdb)
	if err := bus.Start(ctx); err != nil {
		t.Fatalf("bus start: %v", err)
	}
	defer bus.Stop()

	consumer := audit.NewConsumer(pool)
	if err := consumer.Register(bus); err != nil {
		t.Fatalf("register: %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	outbox := events.NewOutbox(pool, bus, events.DefaultOutboxConfig())

	// Emitir evento audit dentro de una TX
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	action := events.EventAPIKeyCreated
	apiKeyID := "key_test_" + time.Now().Format("150405")
	err = audit.Log(ctx, tx, outbox, audit.Entry{
		Action:       action,
		TenantID:     "tenant-audit-test",
		ActorID:      "user-42",
		ActorEmail:   "test@example.com",
		ResourceType: "api_key",
		ResourceID:   apiKeyID,
		IP:           "127.0.0.1",
	})
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("audit.Log: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	// Arrancar publisher
	pubCtx, pubCancel := context.WithCancel(ctx)
	defer pubCancel()
	go outbox.Run(pubCtx)

	// Esperar a que se persista en audit_log (max 8s)
	deadline := time.Now().Add(8 * time.Second)
	var count int
	for time.Now().Before(deadline) {
		err := pool.QueryRow(ctx,
			"SELECT COUNT(*) FROM audit_log WHERE resource_id = $1", apiKeyID,
		).Scan(&count)
		if err == nil && count > 0 {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	if count == 0 {
		t.Fatal("timeout: audit_log no recibió el registro")
	}

	var got struct {
		action   string
		actorID  string
		resource string
	}
	err = pool.QueryRow(ctx,
		"SELECT action, actor_id, resource_type FROM audit_log WHERE resource_id = $1", apiKeyID,
	).Scan(&got.action, &got.actorID, &got.resource)
	if err != nil {
		t.Fatalf("query audit_log: %v", err)
	}
	if got.action != action {
		t.Errorf("action: got %q want %q", got.action, action)
	}
	if got.actorID != "user-42" {
		t.Errorf("actor: got %q want %q", got.actorID, "user-42")
	}
	if got.resource != "api_key" {
		t.Errorf("resource: got %q want %q", got.resource, "api_key")
	}
	t.Logf("✅ audit_log OK: action=%s actor=%s resource=%s", got.action, got.actorID, got.resource)
}
