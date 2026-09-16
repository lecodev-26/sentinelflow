package gateway

import (
"fmt"
"net/http"
"strings"
)

type Limits struct {
MaxBodySize    int64
MaxHeaderSize  int64
MaxMessages    int
MaxTokens      int
MaxTimeoutSecs int
}

func DefaultLimits() Limits {
return Limits{
MaxBodySize:    1 * 1024 * 1024,
MaxHeaderSize:  16 * 1024,
MaxMessages:    100,
MaxTokens:      128000,
MaxTimeoutSecs: 120,
}
}

type LimitMiddleware struct {
limits Limits
}

func NewLimitMiddleware(limits Limits) *LimitMiddleware {
return &LimitMiddleware{limits: limits}
}

// Handler devuelve un Middleware compatible con el pipeline
func (m *LimitMiddleware) Handler(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
if r.ContentLength > m.limits.MaxBodySize {
WriteError(w, NewInvalidRequestError(fmt.Sprintf("request body too large (max %d bytes)", m.limits.MaxBodySize)))
return
}

r.Body = http.MaxBytesReader(w, r.Body, m.limits.MaxBodySize)

var headerSize int64
for k, v := range r.Header {
headerSize += int64(len(k))
for _, val := range v {
headerSize += int64(len(val))
}
}
if headerSize > m.limits.MaxHeaderSize {
WriteError(w, NewInvalidRequestError("request headers too large"))
return
}

switch r.Method {
case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions:
default:
WriteError(w, NewInvalidRequestError("method not allowed"))
return
}

if strings.Contains(r.URL.Path, "..") {
WriteError(w, NewInvalidRequestError("invalid path"))
return
}

next.ServeHTTP(w, r)
})
}
