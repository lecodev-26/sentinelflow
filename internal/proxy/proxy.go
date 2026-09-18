package proxy

import (
"context"
"encoding/json"
"io"
"net/http"
"os"
"time"

"github.com/lecodev-26/sentinelflow/internal/config"
"github.com/lecodev-26/sentinelflow/internal/gateway"
gwcontext "github.com/lecodev-26/sentinelflow/internal/gateway/context"
"github.com/lecodev-26/sentinelflow/internal/logger"
"github.com/lecodev-26/sentinelflow/internal/provider"
"github.com/lecodev-26/sentinelflow/internal/provider/health"
"github.com/lecodev-26/sentinelflow/internal/provider/model"
"github.com/lecodev-26/sentinelflow/internal/provider/registry"
"github.com/lecodev-26/sentinelflow/internal/router"
"github.com/lecodev-26/sentinelflow/internal/router/scoring"
)

type Proxy struct {
config        *config.Config
registry      *registry.Registry
router        *router.IntelligentRouter
executor      *gateway.Executor
healthMonitor *health.Monitor
modelRegistry *model.Registry
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

// Provider registry
reg := registry.NewRegistry()
for _, p := range cfg.Providers {
adapter := provider.NewHTTPProvider(p.Name, p.URL, p.Headers, p.Timeout, nil)
reg.Register(adapter)
logger.Infof("✅ Provider registrado: %s", p.Name)
}

// Model registry
modelReg := model.NewRegistry()
for _, m := range model.DefaultCatalog() {
modelReg.Register(m)
}
logger.Infof("📦 Catálogo por defecto: %d modelos", len(modelReg.List()))

// Discovery en background
go discoverModels(modelReg)

// Health monitor
healthMonitor := health.NewMonitor(30*time.Second, 5*time.Second)
for _, p := range reg.GetAll() {
healthMonitor.Register(p)
}
healthMonitor.Start(context.Background())
logger.Info("❤️ Health monitor iniciado")

// Intelligent router
cbCfg := router.DefaultCBConfig()
weights := scoring.DefaultWeights()
intelligentRouter := router.NewIntelligentRouter(reg, modelReg, healthMonitor, weights, cbCfg)
logger.Info("🧠 Intelligent router iniciado")

// Executor
executor := gateway.NewExecutor(intelligentRouter)

return &Proxy{
config:        cfg,
registry:      reg,
router:        intelligentRouter,
executor:      executor,
healthMonitor: healthMonitor,
modelRegistry: modelReg,
}, nil
}

func discoverModels(reg *model.Registry) {
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

discovery := model.NewDiscovery()

if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
if models, err := discovery.DiscoverOpenAI(ctx, apiKey, ""); err == nil {
for _, m := range models {
if existing, ok := reg.Get("openai", m.ID); ok {
m.Pricing = existing.Pricing
m.Limits = existing.Limits
m.Capabilities = existing.Capabilities
}
reg.Register(m)
}
logger.Infof("🔍 OpenAI discovery: %d modelos", len(models))
}
}

if models, err := discovery.DiscoverOllama(ctx, "http://localhost:11434"); err == nil {
for _, m := range models {
reg.Register(m)
}
logger.Infof("🔍 Ollama discovery: %d modelos", len(models))
}
}

func (p *Proxy) Handler() http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
body, err := io.ReadAll(r.Body)
if err != nil {
http.Error(w, "Error reading body", http.StatusBadRequest)
return
}
defer r.Body.Close()

rc, ok := gwcontext.FromContext(r.Context())
if !ok {
rc = gwcontext.New(r.Context())
rc.Method = r.Method
rc.Path = r.URL.Path
rc.IP = r.RemoteAddr
}

resp, status, err := p.executor.Execute(r.Context(), rc, body)
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
"version": "2.0.0",
"config": map[string]interface{}{
"providers": len(p.config.Providers),
"rules":     len(p.config.Rules),
"models":    len(p.modelRegistry.List()),
},
"cache":            p.GetCacheStats(),
"providers":        p.GetProvidersStatusMap(),
"circuit_breakers": p.router.GetCircuitBreakerStatus(),
})
}

func (p *Proxy) GetProvidersStatus(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(p.GetProvidersStatusMap())
}

func (p *Proxy) GetProvidersStatusMap() map[string]interface{} {
status := make(map[string]interface{})
for _, h := range p.healthMonitor.GetAll() {
status[h.Name] = map[string]interface{}{
"status":            string(h.Status),
"last_check":        h.LastCheck,
"last_error":        h.LastError,
"consecutive_fails": h.ConsecutiveFails,
"latency_ms":        h.Latency.Milliseconds(),
}
}
return status
}

func (p *Proxy) GetCircuitBreakerStatus() map[string]string {
return p.router.GetCircuitBreakerStatus()
}

func (p *Proxy) GetCacheStats() map[string]interface{} {
return map[string]interface{}{
"enabled": p.config.Cache.Enabled,
"ttl":     p.config.Cache.TTL.String(),
"size":    0,
}
}

func (p *Proxy) GetModels() []*model.Model {
return p.modelRegistry.List()
}

func (p *Proxy) GetModelsByCapability(cap model.Capability) []*model.Model {
return p.modelRegistry.FindByCapability(cap)
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

func (p *Proxy) Stop() {
if p.healthMonitor != nil {
p.healthMonitor.Stop()
}
}

// GetModelRegistry devuelve el registry de modelos
func (p *Proxy) GetModelRegistry() *model.Registry {
return p.modelRegistry
}
