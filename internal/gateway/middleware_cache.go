package gateway

import (
"bytes"
"encoding/json"
"io"
"net/http"
"time"

gwcontext "github.com/lecodev-26/sentinelflow/internal/gateway/context"
"github.com/lecodev-26/sentinelflow/internal/cache"
"github.com/lecodev-26/sentinelflow/internal/logger"
)

// CacheMiddleware cachea respuestas de chat
type CacheMiddleware struct {
cache   *cache.Cache
enabled bool
ttl     time.Duration
}

// NewCacheMiddleware crea un nuevo middleware de caché
func NewCacheMiddleware(enabled bool, ttl time.Duration) *CacheMiddleware {
if ttl == 0 {
ttl = 5 * time.Minute
}
return &CacheMiddleware{
cache:   cache.NewCache(ttl),
enabled: enabled,
ttl:     ttl,
}
}

func (m *CacheMiddleware) Handler(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
if !m.enabled {
next.ServeHTTP(w, r)
return
}

// Solo cachear POST a /chat/completions
if r.Method != http.MethodPost {
next.ServeHTTP(w, r)
return
}

rc, ok := gwcontext.FromContext(r.Context())
if !ok {
WriteError(w, NewInternalError("missing request context", nil))
return
}

// Leer body
body, err := io.ReadAll(r.Body)
if err != nil {
WriteError(w, NewInvalidRequestError("error reading body"))
return
}
defer r.Body.Close()

// Parsear modelo y stream
var reqBody map[string]interface{}
_ = json.Unmarshal(body, &reqBody)

if model, ok := reqBody["model"].(string); ok {
rc.Model = model
}
if stream, ok := reqBody["stream"].(bool); ok {
rc.Stream = stream
}

// No cachear streaming
if rc.Stream {
r.Body = io.NopCloser(bytes.NewReader(body))
next.ServeHTTP(w, r)
return
}

// Generar cache key
cacheKey := cache.NewCacheKey(
rc.TenantID,
rc.ProjectID,
rc.Model,
"", // provider se rellena después
reqBody,
)
keyStr := cacheKey.String()

// Intentar leer de caché
if cached, found := m.cache.Get(keyStr); found {
logger.Infof("✅ Cache HIT: %s", keyStr)
rc.CacheHit = true
w.Header().Set("X-Cache", "HIT")
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
w.Write(cached.([]byte))
return
}

logger.Infof("❌ Cache MISS: %s", keyStr)
w.Header().Set("X-Cache", "MISS")

// Capturar respuesta
recorder := &responseRecorder{
ResponseWriter: w,
statusCode:     http.StatusOK,
buffer:         &bytes.Buffer{},
}

// Restaurar body
r.Body = io.NopCloser(bytes.NewReader(body))

// Ejecutar siguiente
next.ServeHTTP(recorder, r)

// Guardar en caché si fue exitoso
if recorder.statusCode == http.StatusOK {
m.cache.Set(keyStr, recorder.buffer.Bytes())
logger.Infof("💾 Guardado en caché: %s", keyStr)
}
})
}

// responseRecorder captura la respuesta
type responseRecorder struct {
http.ResponseWriter
statusCode int
buffer     *bytes.Buffer
}

func (r *responseRecorder) WriteHeader(code int) {
r.statusCode = code
r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
r.buffer.Write(b)
return r.ResponseWriter.Write(b)
}
