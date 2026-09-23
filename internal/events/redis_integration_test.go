package events

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func skipIfNoRedis(t *testing.T) *redis.Client {
	t.Helper()
	url := os.Getenv("SENTINELFLOW_REDIS_URL")
	if url == "" {
		url = "redis://localhost:6379/0"
	}
	opt, err := redis.ParseURL(url)
	if err != nil {
		t.Skipf("bad redis url: %v", err)
	}
	client := redis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not available: %v", err)
	}
	return client
}

func TestRedisBus_PublishSubscribe(t *testing.T) {
	client := skipIfNoRedis(t)
	defer client.Close()

	bus := NewRedisBus(client)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := bus.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer bus.Stop()

	received := make(chan Event, 1)
	bus.Subscribe("test.event", NewConsumer("test", func(_ context.Context, ev Event) {
		received <- ev
	}))

	// dar tiempo a que la suscripción esté activa
	time.Sleep(200 * time.Millisecond)

	ev := NewEvent("test.event").
		WithTenant("tenant-1").
		WithProject("proj-1").
		WithPayload("hello", "world")

	if err := bus.Publish(ctx, ev); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case got := <-received:
		if got.ID != ev.ID {
			t.Errorf("id mismatch: got %s want %s", got.ID, ev.ID)
		}
		if got.TenantID != "tenant-1" {
			t.Errorf("tenant mismatch: got %s", got.TenantID)
		}
		if got.ProjectID != "proj-1" {
			t.Errorf("project mismatch: got %s", got.ProjectID)
		}
		if got.Payload["hello"] != "world" {
			t.Errorf("payload mismatch: %+v", got.Payload)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout: no event received")
	}
}
