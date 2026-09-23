package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lecodev-26/sentinelflow/internal/analytics"
	"github.com/lecodev-26/sentinelflow/internal/audit"
	"github.com/lecodev-26/sentinelflow/internal/cache"
	"github.com/lecodev-26/sentinelflow/internal/events"
	"github.com/lecodev-26/sentinelflow/internal/logger"
	"github.com/lecodev-26/sentinelflow/internal/security"
	"github.com/lecodev-26/sentinelflow/internal/storage/postgres"
	"github.com/lecodev-26/sentinelflow/internal/version"
)

func main() {
	log.Printf("🛡️ SentinelFlow Worker v%s", version.Full())

	logger.Init(&struct {
		Level  string
		Format string
		Output string
	}{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	})

	// === Configuración ===
	dbURL := os.Getenv("SENTINELFLOW_DATABASE_URL")
	if dbURL == "" {
		log.Fatalf("❌ SENTINELFLOW_DATABASE_URL is required")
	}
	redisURL := os.Getenv("SENTINELFLOW_REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
		log.Printf("⚠️ SENTINELFLOW_REDIS_URL not set, using default %s", redisURL)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// === PostgreSQL ===
	pgClient, err := postgres.New(ctx, postgres.DefaultConfig(dbURL))
	if err != nil {
		log.Fatalf("❌ Error conectando a PostgreSQL: %v", err)
	}
	defer pgClient.Close()
	log.Printf("✅ PostgreSQL conectado")

	// === Redis ===
	redisClient, err := cache.NewRedisClientFromURL(redisURL)
	if err != nil {
		log.Fatalf("❌ Error conectando a Redis: %v", err)
	}
	defer redisClient.Close()
	if err := redisClient.Ping(ctx); err != nil {
		log.Fatalf("❌ Redis ping failed: %v", err)
	}
	log.Printf("✅ Redis conectado")

	// === EventBus (RedisBus en prod) ===
	bus := events.NewRedisBus(redisClient.GetClient())
	if err := bus.Start(ctx); err != nil {
		log.Fatalf("❌ EventBus start failed: %v", err)
	}
	defer bus.Stop()
	log.Printf("✅ EventBus iniciado (RedisBus)")

	// === Outbox Publisher ===
	pool := pgClient.Pool()
	if pool == nil {
		log.Fatalf("❌ pgClient.Pool() returned nil; revisar internal/storage/postgres")
	}
	outbox := events.NewOutbox(pool, bus, events.DefaultOutboxConfig())

	// === Consumers ===
	analyticsConsumer := analytics.NewConsumer(pool)
	if err := analyticsConsumer.Register(bus); err != nil {
		log.Fatalf("❌ analytics consumer register failed: %v", err)
	}
	log.Printf("📊 Analytics consumer registrado")

	auditConsumer := audit.NewConsumer(pool)
	if err := auditConsumer.Register(bus); err != nil {
		log.Fatalf("❌ audit consumer register failed: %v", err)
	}
	log.Printf("📜 Audit consumer registrado")

	securityConsumer := security.NewConsumer(pool)
	if err := securityConsumer.Register(bus); err != nil {
		log.Fatalf("❌ security consumer register failed: %v", err)
	}
	log.Printf("🛡️ Security consumer registrado")

	// === Arrancar publisher en goroutine ===
	pubCtx, pubCancel := context.WithCancel(ctx)
	defer pubCancel()
	go func() {
		if err := outbox.Run(pubCtx); err != nil && err != context.Canceled {
			log.Printf("⚠️ Outbox.Run terminó: %v", err)
		}
	}()
	log.Printf("📤 OutboxPublisher activo")

	logger.Info("⚙️ Worker listo (event consumers)")

	// === Esperar señal de parada ===
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("🔄 Apagando worker...")
	pubCancel()
	cancel()
	time.Sleep(500 * time.Millisecond)
	log.Println("✅ Worker detenido")
}
