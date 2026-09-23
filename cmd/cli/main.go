package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/lecodev-26/sentinelflow/internal/cache"
	"github.com/lecodev-26/sentinelflow/internal/storage/postgres"
	"github.com/lecodev-26/sentinelflow/internal/version"
)

func main() {
	showVersion := flag.Bool("version", false, "Show version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("sfctl %s\n", version.Full())
		return
	}

	args := flag.Args()
	if len(args) == 0 {
		printUsage()
		os.Exit(0)
	}

	switch args[0] {
	case "version":
		fmt.Println(version.Full())
	case "health":
		fmt.Println("Checking health... (not implemented yet)")
	case "doctor":
		if err := runDoctor(); err != nil {
			fmt.Fprintf(os.Stderr, "\n❌ doctor falló: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Printf("Unknown command: %s\n", args[0])
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("sfctl - SentinelFlow CLI")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  sfctl <command> [args]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  version       Show version")
	fmt.Println("  health        Check gateway health")
	fmt.Println("  doctor        Check environment readiness (Postgres, Redis, migrations)")
	fmt.Println()
	fmt.Println("Version:", version.Full())
}

// checkResult es el resultado de una verificación.
type checkResult struct {
	Name   string
	OK     bool
	Detail string
}

func runDoctor() error {
	fmt.Printf("🩺 SentinelFlow doctor v%s\n\n", version.Full())

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var results []checkResult

	// 1. Config: variables de entorno
	dbURL := os.Getenv("SENTINELFLOW_DATABASE_URL")
	redisURL := os.Getenv("SENTINELFLOW_REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}

	results = append(results, checkResult{
		Name:   "config.SENTINELFLOW_DATABASE_URL",
		OK:     dbURL != "",
		Detail: maskURL(dbURL),
	})
	results = append(results, checkResult{
		Name:   "config.SENTINELFLOW_REDIS_URL",
		OK:     true,
		Detail: redisURL,
	})

	// 2. Postgres
	var pgClient *postgres.Client
	if dbURL != "" {
		client, err := postgres.New(ctx, postgres.DefaultConfig(dbURL))
		if err != nil {
			results = append(results, checkResult{
				Name:   "postgres.connect",
				OK:     false,
				Detail: err.Error(),
			})
		} else {
			pgClient = client
			defer pgClient.Close()
			results = append(results, checkResult{
				Name:   "postgres.connect",
				OK:     true,
				Detail: "OK",
			})

			// 3. Migraciones
			report, err := pgClient.MigrationStatus(ctx)
			if err != nil {
				results = append(results, checkResult{
					Name:   "postgres.migrations",
					OK:     false,
					Detail: err.Error(),
				})
			} else {
				pending := 0
				for _, r := range report {
					if !r.Applied {
						pending++
					}
				}
				ok := pending == 0
				detail := fmt.Sprintf("%d applied, %d pending", len(report)-pending, pending)
				if !ok {
					detail += "  → run: sfctl migrate up"
				}
				results = append(results, checkResult{
					Name:   "postgres.migrations",
					OK:     ok,
					Detail: detail,
				})
			}

			// 4. Outbox health
			if ok, detail := checkOutboxHealth(ctx, pgClient); true {
				results = append(results, checkResult{
					Name:   "outbox.health",
					OK:     ok,
					Detail: detail,
				})
			}
		}
	}

	// 5. Redis
	redisClient, err := cache.NewRedisClientFromURL(redisURL)
	if err != nil {
		results = append(results, checkResult{
			Name:   "redis.connect",
			OK:     false,
			Detail: err.Error(),
		})
	} else {
		defer redisClient.Close()
		if err := redisClient.Ping(ctx); err != nil {
			results = append(results, checkResult{
				Name:   "redis.connect",
				OK:     false,
				Detail: err.Error(),
			})
		} else {
			info, _ := redisClient.GetClient().Info(ctx, "server").Result()
			ver := extractRedisVersion(info)
			results = append(results, checkResult{
				Name:   "redis.connect",
				OK:     true,
				Detail: "OK " + ver,
			})
		}
	}

	// 6. Providers (env vars)
	openai := os.Getenv("OPENAI_API_KEY") != ""
	anthropic := os.Getenv("ANTHROPIC_API_KEY") != ""
	provDetail := fmt.Sprintf("openai=%v anthropic=%v ollama=always", openai, anthropic)
	results = append(results, checkResult{
		Name:   "providers.configured",
		OK:     openai || anthropic,
		Detail: provDetail,
	})

	// === Print ===
	failures := 0
	for _, r := range results {
		mark := "✅"
		if !r.OK {
			mark = "❌"
			failures++
		}
		fmt.Printf("  %s  %-40s  %s\n", mark, r.Name, r.Detail)
	}

	fmt.Println()
	if failures == 0 {
		fmt.Println("✅ Entorno OK — production blockers: 0")
		return nil
	}
	fmt.Printf("⚠️  %d check(s) failed\n", failures)
	return fmt.Errorf("%d check(s) failed", failures)
}

func checkOutboxHealth(ctx context.Context, client *postgres.Client) (bool, string) {
	pool := client.Pool()
	if pool == nil {
		return false, "no pool"
	}
	var pending, failed, dead, stuck int64
	_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox_events WHERE status='pending'").Scan(&pending)
	_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox_events WHERE status='failed'").Scan(&failed)
	_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox_events WHERE status='dead'").Scan(&dead)
	_ = pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM outbox_events WHERE status='publishing' AND updated_at < NOW() - INTERVAL '10 minutes'",
	).Scan(&stuck)

	ok := dead == 0 && stuck == 0
	detail := fmt.Sprintf("pending=%d failed=%d dead=%d stuck=%d", pending, failed, dead, stuck)
	if !ok {
		detail += "  ← revisa worker/outbox"
	}
	return ok, detail
}

func maskURL(u string) string {
	if u == "" {
		return "(empty)"
	}
	if len(u) < 30 {
		return u
	}
	return u[:20] + "..." + u[len(u)-8:]
}

func extractRedisVersion(info string) string {
	// busca "redis_version:X.Y.Z" en el INFO
	const key = "redis_version:"
	idx := -1
	for i := 0; i+len(key) < len(info); i++ {
		if info[i:i+len(key)] == key {
			idx = i + len(key)
			break
		}
	}
	if idx < 0 {
		return ""
	}
	end := idx
	for end < len(info) && info[end] != '\r' && info[end] != '\n' {
		end++
	}
	return "redis=" + info[idx:end]
}

// referenciar redis para que el import no se pierda si se refactoriza
var _ = redis.Nil
