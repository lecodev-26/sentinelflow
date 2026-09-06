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

<<<<<<< HEAD
	"github.com/gorilla/mux"
	"github.com/lecodev-26/sentinelflow/internal/metrics"
	"github.com/lecodev-26/sentinelflow/internal/proxy"
	"github.com/lecodev-26/sentinelflow/internal/ratelimit"
)

func main() {
	port := flag.String("port", "8080", "Puerto del proxy")
	metricsPort := flag.String("metrics-port", "9090", "Puerto para métricas")
	configFile := flag.String("config", "configs/rules.yaml", "Archivo de configuración")
	flag.Parse()

	log.Printf("🛡️ SentinelFlow iniciando en puerto %s", *port)
	log.Printf("📋 Configuración: %s", *configFile)

=======
	"github.com/lecodev-26/sentinelflow/internal/proxy"


func main() {
	// Flags de línea de comandos
	port := flag.String("port", "8080", "Puerto del proxy")
	configFile := flag.String("config", "configs/rules.yaml", "Archivo de configuración")
	flag.Parse()

	log.Printf("🛡️  SentinelFlow iniciando en puerto %s", *port)
	log.Printf("📋 Configuración: %s", *configFile)

	// Crear proxy
>>>>>>> b011fca1f048d402212fded110a6fddb45a10859
	p, err := proxy.NewProxy(*configFile)
	if err != nil {
		log.Fatalf("❌ Error creando proxy: %v", err)
	}

<<<<<<< HEAD
	// Rate Limiter: 100 peticiones por minuto por IP
	limiter := ratelimit.NewLimiter(100, time.Minute)

	// Router principal
	r := mux.NewRouter()

	// Middleware de métricas
	r.Use(metrics.MetricsMiddleware)

	// Middleware de Rate Limiting
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if !limiter.Allow(ip) {
				http.Error(w, "Too many requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// Rutas principales
	r.Handle("/", p.Handler())
	r.HandleFunc("/health", p.HealthCheck)

	// Dashboard
	r.PathPrefix("/dashboard").Handler(
		http.StripPrefix("/dashboard", http.FileServer(http.Dir("./web/dashboard"))),
	)

	// API para dashboard
	r.HandleFunc("/api/providers", p.GetProvidersStatus).Methods("GET")
	r.HandleFunc("/api/logs/stream", p.StreamLogs).Methods("GET")

	srv := &http.Server{
		Addr:         ":" + *port,
		Handler:      r,
=======
	// Configurar rutas
	mux := http.NewServeMux()
	mux.Handle("/", p.Handler())
	mux.HandleFunc("/health", p.HealthCheck)

	// Servidor HTTP
	srv := &http.Server{
		Addr:         ":" + *port,
		Handler:      mux,
>>>>>>> b011fca1f048d402212fded110a6fddb45a10859
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

<<<<<<< HEAD
	metricsSrv := &http.Server{
		Addr:    ":" + *metricsPort,
		Handler: metrics.Handler(),
	}

=======
	// Manejar señales para graceful shutdown
>>>>>>> b011fca1f048d402212fded110a6fddb45a10859
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
<<<<<<< HEAD
		log.Printf("✅ Proxy en http://localhost:%s", *port)
		log.Printf("📊 Dashboard en http://localhost:%s/dashboard", *port)
		log.Printf("📈 Métricas en http://localhost:%s/metrics", *metricsPort)
		log.Printf("🔒 Rate Limiting activo: 100 req/min por IP")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Error: %v", err)
		}
	}()

	go func() {
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("⚠️ Error en métricas: %v", err)
		}
	}()

	<-stop
	log.Println("🔄 Apagando...")
=======
		log.Printf("✅ Proxy escuchando en http://localhost:%s", *port)
		log.Printf("📊 Health check: http://localhost:%s/health", *port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Error en el servidor: %v", err)
		}
	}()

	// Esperar señal de terminación
	<-stop
	log.Println("🔄 Apagando servidor...")
>>>>>>> b011fca1f048d402212fded110a6fddb45a10859

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

<<<<<<< HEAD
	srv.Shutdown(ctx)
	metricsSrv.Shutdown(ctx)
=======
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("❌ Error en shutdown: %v", err)
	}
>>>>>>> b011fca1f048d402212fded110a6fddb45a10859

	log.Println("✅ SentinelFlow detenido correctamente")
}
