package main

import (
"context"
"flag"
"log"
"net/http"
"os"
"os/signal"
"syscall"
"time"

"github.com/gorilla/mux"
gwcontext "github.com/lecodev-26/sentinelflow/internal/gateway/context"
"github.com/lecodev-26/sentinelflow/internal/logger"
"github.com/lecodev-26/sentinelflow/internal/version"
)

func main() {
configFile := flag.String("config", "configs/rules.yaml", "Config file")
flag.Parse()

log.Printf("🛡️ SentinelFlow Gateway v%s", version.Full())
log.Printf("   Config: %s", *configFile)

// Iniciar logger
logger.Init(&struct {
Level  string
Format string
Output string
}{
Level:  "info",
Format: "json",
Output: "stdout",
})

// Router básico
r := mux.NewRouter()

// Health check
r.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
w.Write([]byte(`{"status":"ok","service":"sentinelflow-gateway","version":"` + version.String() + `"}`))
}).Methods("GET")

// Version
r.HandleFunc("/version", func(w http.ResponseWriter, req *http.Request) {
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
info := version.Get()
w.Write([]byte(`{"version":"` + info.Version + `","commit":"` + info.Commit + `","go":"` + info.GoVersion + `"}`))
}).Methods("GET")

// Placeholder para V3.2 (Gateway Core)
r.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, req *http.Request) {
ctx, _ := gwcontext.FromContext(req.Context())
if ctx == nil {
ctx = gwcontext.New(req.Context())
}
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusNotImplemented)
w.Write([]byte(`{"error":{"type":"not_implemented","message":"V3.2 in progress"}}`))
}).Methods("POST")

// Servidor
srv := &http.Server{
Addr:         ":8080",
Handler:      r,
ReadTimeout:  30 * time.Second,
WriteTimeout: 30 * time.Second,
IdleTimeout:  60 * time.Second,
}

stop := make(chan os.Signal, 1)
signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

go func() {
log.Printf("✅ Gateway en http://localhost:8080")
if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
log.Fatalf("❌ Gateway error: %v", err)
}
}()

<-stop
log.Println("🔄 Apagando...")

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

srv.Shutdown(ctx)
log.Println("✅ Gateway detenido")
}
