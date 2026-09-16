package rules

import (
"context"
"encoding/json"
"fmt"
"net/http"

"github.com/lecodev-26/sentinelflow/internal/config"
"github.com/lecodev-26/sentinelflow/internal/logger"
"github.com/lecodev-26/sentinelflow/internal/metrics"
"github.com/lecodev-26/sentinelflow/internal/provider"
"github.com/lecodev-26/sentinelflow/internal/provider/registry"
"github.com/lecodev-26/sentinelflow/internal/router"
)

// Engine es el punto de entrada del gateway.
// Ahora usa provider.Registry + router.SmartRouter en lugar de lógica propia.
type Engine struct {
config  *config.Config
registry *registry.Registry
router  *router.SmartRouter
}

// NewEngine crea un nuevo Engine unificado
func NewEngine(cfg *config.Config) *Engine {
// Crear registry
reg := registry.NewRegistry()

// Registrar todos los providers de la configuración
// usando el adapter HTTP genérico
for _, p := range cfg.Providers {
adapter := provider.NewHTTPProvider(
p.Name,
p.URL,
p.Headers,
p.Timeout,
nil, // modelos: se descubrirán dinámicamente
)
reg.Register(adapter)
logger.Infof("✅ Provider registrado: %s", p.Name)
}

// Crear smart router
smartRouter := router.NewSmartRouter(reg)

return &Engine{
config:   cfg,
registry: reg,
router:   smartRouter,
}
}

// Route procesa una petición HTTP y la enruta al mejor proveedor
func (e *Engine) Route(path, method string, body []byte, headers http.Header) ([]byte, int, error) {
logger.Infof("📨 %s %s", method, path)

// 1. Buscar regla para esta ruta
rule := e.config.GetRuleByPath(path, method)
if rule == nil {
logger.Warnf("⚠️ No hay regla para %s %s", method, path)
return nil, http.StatusNotFound, fmt.Errorf("no rule for %s %s", method, path)
}

// 2. Parsear body para obtener modelo
var reqBody map[string]interface{}
var model string
if len(body) > 0 {
if err := json.Unmarshal(body, &reqBody); err == nil {
if m, ok := reqBody["model"].(string); ok {
model = m
}
}
}

// 3. Construir ChatRequest normalizado
messages := []provider.Message{}
if msgs, ok := reqBody["messages"].([]interface{}); ok {
for _, m := range msgs {
if msg, ok := m.(map[string]interface{}); ok {
role, _ := msg["role"].(string)
content, _ := msg["content"].(string)
messages = append(messages, provider.Message{
Role:    role,
Content: content,
})
}
}
}

chatReq := &provider.ChatRequest{
Model:    model,
Messages: messages,
}

if temp, ok := reqBody["temperature"].(float64); ok {
t := float32(temp)
chatReq.Temperature = &t
}
if maxTok, ok := reqBody["max_tokens"].(float64); ok {
mt := int(maxTok)
chatReq.MaxTokens = &mt
}
if stream, ok := reqBody["stream"].(bool); ok {
chatReq.Stream = stream
}

// 4. Ejecutar con el SmartRouter
ctx := context.Background()
resp, err := e.router.Route(ctx, chatReq)
if err != nil {
logger.Errorf("❌ Router falló: %v", err)
metrics.RecordProviderFailure("router")
return nil, http.StatusServiceUnavailable, err
}

logger.Infof("✅ Respuesta de: %s (%.2fms)", resp.Provider, float64(resp.Latency.Microseconds())/1000.0)

// 5. Convertir respuesta a JSON
respJSON, err := json.Marshal(resp)
if err != nil {
return nil, http.StatusInternalServerError, fmt.Errorf("error marshaling response: %w", err)
}

return respJSON, http.StatusOK, nil
}

// GetCacheStats devuelve estadísticas (placeholder mientras migramos)
func (e *Engine) GetCacheStats() map[string]interface{} {
return map[string]interface{}{
"enabled": e.config.Cache.Enabled,
"ttl":     e.config.Cache.TTL.String(),
"size":    0,
}
}

// GetProviderStatus devuelve el estado de los providers
func (e *Engine) GetProviderStatus() map[string]interface{} {
status := make(map[string]interface{})
for _, p := range e.registry.GetAll() {
status[p.Name()] = map[string]interface{}{
"status": "unknown",
}
}
return status
}
