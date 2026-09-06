package proxy

import (
	"encoding/json"
	"io"
	"net/http"
<<<<<<< HEAD
	"time"

	"github.com/lecodev-26/sentinelflow/internal/config"
	"github.com/lecodev-26/sentinelflow/internal/logger"
	"github.com/lecodev-26/sentinelflow/internal/rules"
=======

	"github.com/leco-dev26/sentinelflow/internal/config"
	"github.com/leco-dev26/sentinelflow/internal/logger"
	"github.com/leco-dev26/sentinelflow/internal/rules"
>>>>>>> b011fca1f048d402212fded110a6fddb45a10859
)

// Proxy es el núcleo del sistema
type Proxy struct {
	config *config.Config
	engine *rules.Engine
}

// NewProxy crea una nueva instancia del proxy
func NewProxy(configFile string) (*Proxy, error) {
<<<<<<< HEAD
=======
	// Cargar configuración
>>>>>>> b011fca1f048d402212fded110a6fddb45a10859
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return nil, err
	}

<<<<<<< HEAD
=======
	// Inicializar logger
>>>>>>> b011fca1f048d402212fded110a6fddb45a10859
	logger.Init(&struct {
		Level  string
		Format string
		Output string
	}{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
		Output: cfg.Logging.Output,
	})

<<<<<<< HEAD
	logger.Info("🛡️ SentinelFlow iniciado correctamente")
	logger.Infof("📋 Proveedores cargados: %d", len(cfg.Providers))
	logger.Infof("📋 Reglas cargadas: %d", len(cfg.Rules))

=======
	logger.Info("🛡️  SentinelFlow iniciado correctamente")
	logger.Infof("📋 Proveedores cargados: %d", len(cfg.Providers))
	logger.Infof("📋 Reglas cargadas: %d", len(cfg.Rules))

	// Crear motor de reglas
>>>>>>> b011fca1f048d402212fded110a6fddb45a10859
	engine := rules.NewEngine(cfg)

	return &Proxy{
		config: cfg,
		engine: engine,
	}, nil
}

// Handler devuelve el http.Handler principal
func (p *Proxy) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
<<<<<<< HEAD
=======
		// Leer body
>>>>>>> b011fca1f048d402212fded110a6fddb45a10859
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

<<<<<<< HEAD
		resp, status, err := p.engine.Route(r.URL.Path, r.Method, body, r.Header)
=======
		// Procesar ruta con el motor
		resp, status, err := p.engine.Route(r.URL.Path, r.Method, body, r.Header)

>>>>>>> b011fca1f048d402212fded110a6fddb45a10859
		if err != nil {
			logger.Errorf("❌ Error procesando: %v", err)
			http.Error(w, err.Error(), status)
			return
		}

<<<<<<< HEAD
=======
		// Escribir respuesta
>>>>>>> b011fca1f048d402212fded110a6fddb45a10859
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write(resp)
	})
}

// HealthCheck endpoint para verificar el estado
func (p *Proxy) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"service": "sentinel-flow",
<<<<<<< HEAD
		"version": "0.2.0",
=======
		"version": "0.1.0",
>>>>>>> b011fca1f048d402212fded110a6fddb45a10859
		"config": map[string]interface{}{
			"providers": len(p.config.Providers),
			"rules":     len(p.config.Rules),
		},
		"cache": p.engine.GetCacheStats(),
	})
}
<<<<<<< HEAD

// GetProvidersStatus devuelve el estado de los proveedores
func (p *Proxy) GetProvidersStatus(w http.ResponseWriter, r *http.Request) {
	var status []map[string]interface{}
	for _, provider := range p.config.Providers {
		status = append(status, map[string]interface{}{
			"name":   provider.Name,
			"status": "online",
			"url":    provider.URL,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// StreamLogs envía logs en tiempo real via SSE
func (p *Proxy) StreamLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logEntry := map[string]interface{}{
				"time":    time.Now().Format("15:04:05"),
				"level":   "info",
				"message": "Proxy procesando peticiones...",
			}
			data, _ := json.Marshal(logEntry)
			w.Write([]byte("data: " + string(data) + "\n\n"))
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
=======
>>>>>>> b011fca1f048d402212fded110a6fddb45a10859
