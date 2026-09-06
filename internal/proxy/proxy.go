package proxy

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/leco-dev26/sentinelflow/internal/config"
	"github.com/leco-dev26/sentinelflow/internal/logger"
	"github.com/leco-dev26/sentinelflow/internal/rules"
)

// Proxy es el núcleo del sistema
type Proxy struct {
	config *config.Config
	engine *rules.Engine
}

// NewProxy crea una nueva instancia del proxy
func NewProxy(configFile string) (*Proxy, error) {
	// Cargar configuración
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return nil, err
	}

	// Inicializar logger
	logger.Init(&struct {
		Level  string
		Format string
		Output string
	}{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
		Output: cfg.Logging.Output,
	})

	logger.Info("🛡️  SentinelFlow iniciado correctamente")
	logger.Infof("📋 Proveedores cargados: %d", len(cfg.Providers))
	logger.Infof("📋 Reglas cargadas: %d", len(cfg.Rules))

	// Crear motor de reglas
	engine := rules.NewEngine(cfg)

	return &Proxy{
		config: cfg,
		engine: engine,
	}, nil
}

// Handler devuelve el http.Handler principal
func (p *Proxy) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Leer body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Procesar ruta con el motor
		resp, status, err := p.engine.Route(r.URL.Path, r.Method, body, r.Header)

		if err != nil {
			logger.Errorf("❌ Error procesando: %v", err)
			http.Error(w, err.Error(), status)
			return
		}

		// Escribir respuesta
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
		"version": "0.1.0",
		"config": map[string]interface{}{
			"providers": len(p.config.Providers),
			"rules":     len(p.config.Rules),
		},
		"cache": p.engine.GetCacheStats(),
	})
}
