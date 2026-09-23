// Command migrator aplica las migraciones pendientes de SentinelFlow.
//
// Uso:
//
//SENTINELFLOW_DATABASE_URL=... migrator            # aplica pendientes
//SENTINELFLOW_DATABASE_URL=... migrator status     # muestra estado sin aplicar
//
// Diseñado para CI/CD: no arranca HTTP, no abre Redis, solo Postgres.
package main

import (
"context"
"fmt"
"log"
"os"
"time"

"github.com/lecodev-26/sentinelflow/internal/storage/postgres"
"github.com/lecodev-26/sentinelflow/internal/version"
)

func main() {
cmd := "up"
if len(os.Args) > 1 {
cmd = os.Args[1]
}

dbURL := os.Getenv("SENTINELFLOW_DATABASE_URL")
if dbURL == "" {
log.Fatalf("❌ SENTINELFLOW_DATABASE_URL is required")
}

ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

client, err := postgres.New(ctx, postgres.DefaultConfig(dbURL))
if err != nil {
log.Fatalf("❌ Error conectando a PostgreSQL: %v", err)
}
defer client.Close()
fmt.Printf("✅ Conectado a PostgreSQL\n")

switch cmd {
case "up", "migrate":
if err := client.Migrate(ctx); err != nil {
log.Fatalf("❌ Migración falló: %v", err)
}
fmt.Printf("✅ Migraciones aplicadas\n")

case "status":
report, err := client.MigrationStatus(ctx)
if err != nil {
log.Fatalf("❌ Error leyendo estado: %v", err)
}
fmt.Printf("SentinelFlow v%s — migration status\n", version.Full())
fmt.Printf("%-6s  %-40s  %s\n", "VER", "NAME", "STATUS")
for _, r := range report {
status := "⏳ pending"
if r.Applied {
status = "✅ applied"
}
fmt.Printf("%-6d  %-40s  %s\n", r.Version, r.Name, status)
}

case "version":
fmt.Println(version.Full())

default:
fmt.Fprintf(os.Stderr, "uso: migrator [up|status|version]\n")
os.Exit(2)
}
}
