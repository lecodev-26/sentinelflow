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

	"github.com/TU-USUARIO/sentinelflow/internal/proxy"
)

func main() {
	// Flags de línea de comandos
	port := flag.String("port", "8080", "Puerto del proxy")
	configFile := flag.String("config", "configs/rules.yaml", "Archivo de configuración")
	flag.Parse()

	log.Printf("🛡️  SentinelFlow iniciando en puerto %s", *port)
	log.Printf("📋 Configuración: %s", *configFile)

	// Crear proxy
	p, err := proxy.NewProxy(*configFile)
	if err != nil {
		log.Fatalf("❌ Error creando proxy: %v", err)
	}

	// Servidor HTTP
	srv := &http.Server{
		Addr:         ":" + *port,
		Handler:      p.Handler(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Manejar señales para graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("✅ Proxy escuchando en http://localhost:%s", *port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Error en el servidor: %v", err)
		}
	}()

	// Esperar señal de terminación
	<-stop
	log.Println("🔄 Apagando servidor...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("❌ Error en shutdown: %v", err)
	}

	log.Println("✅ SentinelFlow detenido correctamente")
}
