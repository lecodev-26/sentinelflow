package proxy

import (
"encoding/json"
"io"
"net/http"
"time"

"github.com/lecodev-26/sentinelflow/internal/config"
"github.com/lecodev-26/sentinelflow/internal/logger"
"github.com/lecodev-26/sentinelflow/internal/rules"
)

type Proxy struct {
config *config.Config
engine *rules.Engine
}

func NewProxy(configFile string) (*Proxy, error) {
cfg, err := config.LoadConfig(configFile)
if err != nil {
return nil, err
}

logger.Init(&struct {
Level  string
Format string
Output string
}{
Level:  cfg.Logging.Level,
Format: cfg.Logging.Format,
Output: cfg.Logging.Output,
})

logger.Info("🛡️ SentinelFlow iniciado correctamente")
logger.Infof("📋 Proveedores cargados: %d", len(cfg.Providers))
logger.Infof("📋 Reglas cargadas: %d", len(cfg.Rules))

engine := rules.NewEngine(cfg)

return &Proxy{
config: cfg,
engine: engine,
}, nil
}

func (p *Proxy) Handler() http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
body, err := io.ReadAll(r.Body)
if err != nil {
http.Error(w, "Error reading body", http.StatusBadRequest)
return
}
defer r.Body.Close()

resp, status, err := p.engine.Route(r.URL.Path, r.Method, body, r.Header)
if err != nil {
logger.Errorf("❌ Error procesando: %v", err)
http.Error(w, err.Error(), status)
return
}

w.Header().Set("Content-Type", "application/json")
w.WriteHeader(status)
w.Write(resp)
})
}

func (p *Proxy) HealthCheck(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
json.NewEncoder(w).Encode(map[string]interface{}{
"status":  "ok",
"service": "sentinel-flow",
"version": "0.2.0",
"config": map[string]interface{}{
"providers": len(p.config.Providers),
"rules":     len(p.config.Rules),
},
"cache": p.engine.GetCacheStats(),
})
}

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
