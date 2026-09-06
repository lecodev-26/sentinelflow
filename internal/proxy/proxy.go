package proxy

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Proxy es el núcleo del sistema
type Proxy struct {
	configFile string
	// TODO: añadir config, cache, logger, metrics
}

// NewProxy crea una nueva instancia del proxy
func NewProxy(configFile string) (*Proxy, error) {
	p := &Proxy{
		configFile: configFile,
	}
	
	// TODO: cargar configuración
	log.Println("✅ Proxy inicializado")
	return p, nil
}

// Handler devuelve el http.Handler principal
func (p *Proxy) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: implementar lógica de proxy
		
		// Por ahora solo responde con un mensaje de prueba
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		response := map[string]interface{}{
			"status":  "ok",
			"message": "SentinelFlow está funcionando",
			"path":    r.URL.Path,
			"method":  r.Method,
			"time":    time.Now().Format(time.RFC3339),
		}
		
		json.NewEncoder(w).Encode(response)
	})
}
