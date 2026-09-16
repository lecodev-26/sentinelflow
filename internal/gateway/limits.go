package gateway

import (
"fmt"
"net/http"
"strings"
)

// Limits define los límites de una petición
type Limits struct {
MaxBodySize    int64 // bytes
MaxHeaderSize  int64
MaxMessages    int
MaxTokens      int
MaxTimeoutSecs int
}

// DefaultLimits devuelve límites razonables por defecto
func DefaultLimits() Limits {
return Limits{
MaxBodySize:    1 * 1024 * 1024, // 1 MB
MaxHeaderSize:  16 * 1024,        // 16 KB
MaxMessages:    100,
MaxTokens:      128000,
MaxTimeoutSecs: 120,
}
}

// LimitMiddleware aplica límites a las peticiones
type LimitMiddleware struct {
limits Limits
}

func NewLimitMiddleware(limits Limits) *LimitMiddleware {
return &LimitMiddleware{limits: limits}
}

// Handler envuelve un handler con límites
func (m *LimitMiddleware) Handler(next http.HandlerFunc) http.HandlerFunc {
return func(w http.ResponseWriter, r *http.Request) {
// Verificar tamaño del body
if r.ContentLength > m.limits.MaxBodySize {
http.Error(w, fmt.Sprintf("Request body too large (max %d bytes)", m.limits.MaxBodySize), http.StatusRequestEntityTooLarge)
return
}

// Limitar body a nivel de lector
r.Body = http.MaxBytesReader(w, r.Body, m.limits.MaxBodySize)

// Verificar tamaño de headers
var headerSize int64
for k, v := range r.Header {
headerSize += int64(len(k))
for _, val := range v {
headerSize += int64(len(val))
}
}
if headerSize > m.limits.MaxHeaderSize {
http.Error(w, "Request headers too large", http.StatusRequestHeaderFieldsTooLarge)
return
}

// Bloquear métodos raros
switch r.Method {
case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions:
// OK
default:
http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
return
}

// Bloquear paths sospechosos
if strings.Contains(r.URL.Path, "..") {
http.Error(w, "Invalid path", http.StatusBadRequest)
return
}

next(w, r)
}
}
