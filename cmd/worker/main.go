package main

import (
"context"
"log"
"os"
"os/signal"
"syscall"
"time"

"github.com/lecodev-26/sentinelflow/internal/logger"
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

logger.Info("⚙️ Worker iniciado (event consumers)")
logger.Info("   Consumidores: accounting, analytics, audit, billing, alerts, webhooks")

// Por ahora solo espera señales
// V3.6+ añadirá los consumers reales

stop := make(chan os.Signal, 1)
signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

<-stop
log.Println("🔄 Apagando worker...")

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

_ = ctx
log.Println("✅ Worker detenido")
}
