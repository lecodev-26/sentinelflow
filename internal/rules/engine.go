package rules

import (
"bytes"
"encoding/json"
"fmt"
"io"
"net/http"
"time"

"github.com/lecodev-26/sentinelflow/internal/cache"
"github.com/lecodev-26/sentinelflow/internal/config"
"github.com/lecodev-26/sentinelflow/internal/logger"
"github.com/lecodev-26/sentinelflow/internal/metrics"
)

type Engine struct {
config *config.Config
cache  *cache.Cache
client *http.Client
}

func NewEngine(cfg *config.Config) *Engine {
return &Engine{
config: cfg,
cache:  cache.NewCache(cfg.Cache.TTL),
client: &http.Client{
Timeout: 30 * time.Second,
},
}
}

type ModelRouter struct {
modelMap map[string]string
}

func NewModelRouter() *ModelRouter {
return &ModelRouter{
modelMap: map[string]string{
"gpt-3.5-turbo":   "openai",
"gpt-4":           "openai",
"gpt-4-turbo":     "openai",
"claude-3":        "anthropic",
"claude-3-sonnet": "anthropic",
"claude-3-opus":   "anthropic",
"llama3":          "local-llama",
"llama3.1":        "local-llama",
"default":         "openai",
},
}
}

func (mr *ModelRouter) GetProvider(model string) string {
if provider, ok := mr.modelMap[model]; ok {
return provider
}
return mr.modelMap["default"]
}

func (e *Engine) Route(path, method string, body []byte, headers http.Header) ([]byte, int, error) {
logger.Infof("📨 %s %s", method, path)

rule := e.config.GetRuleByPath(path, method)
if rule == nil {
logger.Warnf("⚠️ No hay regla para %s %s", method, path)
return nil, http.StatusNotFound, fmt.Errorf("no rule for %s %s", method, path)
}

var preferredProvider string
var reqBody map[string]interface{}

if len(body) > 0 {
if err := json.Unmarshal(body, &reqBody); err == nil {
if model, ok := reqBody["model"].(string); ok {
router := NewModelRouter()
preferredProvider = router.GetProvider(model)
logger.Infof("🎯 Modelo '%s' → Proveedor preferido: '%s'", model, preferredProvider)
metrics.RecordSmartRouting(model, preferredProvider)
}
} else {
logger.Warnf("⚠️ Error parseando JSON del body: %v", err)
}
}

if e.config.Cache.Enabled && rule.Cache {
cacheKey := fmt.Sprintf("%s:%s:%s", method, path, string(body))
if cached, found := e.cache.Get(cacheKey); found {
logger.Infof("✅ Caché hit: %s", cacheKey)
metrics.RecordCacheHit()
return cached.([]byte), http.StatusOK, nil
}
metrics.RecordCacheMiss()
}

providers := rule.Providers
if preferredProvider != "" {
reordered := []string{preferredProvider}
for _, p := range providers {
if p != preferredProvider {
reordered = append(reordered, p)
}
}
providers = reordered
logger.Infof("🔄 Orden de proveedores: %v", providers)
}

var lastErr error
for _, providerName := range providers {
provider := e.config.GetProviderByName(providerName)
if provider == nil {
logger.Warnf("⚠️ Proveedor %s no encontrado", providerName)
continue
}

logger.Infof("🔄 Intentando: %s", provider.Name)

resp, status, err := e.forwardRequest(provider, path, method, body, headers)
if err == nil && status < 500 {
if e.config.Cache.Enabled && rule.Cache {
cacheKey := fmt.Sprintf("%s:%s:%s", method, path, string(body))
e.cache.Set(cacheKey, resp)
logger.Infof("💾 Guardado en caché: %s", cacheKey)
}
return resp, status, nil
}

lastErr = err
logger.Warnf("❌ Falló %s: %v", provider.Name, err)
metrics.RecordProviderFailure(provider.Name)

if provider.Fallback != "" {
logger.Infof("↩️ Fallback a: %s", provider.Fallback)
metrics.RecordFallback(provider.Name, provider.Fallback)
}
}

return nil, http.StatusServiceUnavailable, fmt.Errorf("all providers failed: %v", lastErr)
}

func (e *Engine) forwardRequest(provider *config.Provider, path, method string, body []byte, headers http.Header) ([]byte, int, error) {
fullURL := provider.URL + path
req, err := http.NewRequest(method, fullURL, bytes.NewReader(body))
if err != nil {
return nil, 0, err
}

for key, value := range provider.Headers {
req.Header.Set(key, value)
}

for key, values := range headers {
if key != "Authorization" && key != "X-API-Key" {
for _, v := range values {
req.Header.Set(key, v)
}
}
}

resp, err := e.client.Do(req)
if err != nil {
return nil, 0, err
}
defer resp.Body.Close()

respBody, err := io.ReadAll(resp.Body)
if err != nil {
return nil, 0, err
}

if resp.StatusCode >= 500 {
return respBody, resp.StatusCode, fmt.Errorf("provider returned %d", resp.StatusCode)
}

return respBody, resp.StatusCode, nil
}

func (e *Engine) GetCacheStats() map[string]interface{} {
return map[string]interface{}{
"size":     e.cache.Size(),
"enabled":  e.config.Cache.Enabled,
"ttl":      e.config.Cache.TTL.String(),
"max_size": e.config.Cache.MaxSize,
}
}
