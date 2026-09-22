package gateway

import (
	"net/http"

	gwcontext "github.com/lecodev-26/sentinelflow/internal/gateway/context"
)

// Middleware es una función que envuelve un handler
type Middleware func(http.Handler) http.Handler

// Pipeline es una cadena de middlewares
type Pipeline struct {
	middlewares []Middleware
}

// NewPipeline crea un nuevo pipeline
func NewPipeline() *Pipeline {
	return &Pipeline{
		middlewares: []Middleware{},
	}
}

// Use añade un middleware al pipeline
func (p *Pipeline) Use(mw Middleware) *Pipeline {
	p.middlewares = append(p.middlewares, mw)
	return p
}

// Then construye el handler final aplicando todos los middlewares
func (p *Pipeline) Then(final http.Handler) http.Handler {
	// Aplicar en orden inverso para que el primero sea el más externo
	handler := final
	for i := len(p.middlewares) - 1; i >= 0; i-- {
		handler = p.middlewares[i](handler)
	}
	return handler
}

// ContextMiddleware crea el RequestContext para cada petición
func ContextMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rc := gwcontext.New(r.Context())
			rc.Method = r.Method
			rc.Path = r.URL.Path
			rc.IP = r.RemoteAddr
			rc.UserAgent = r.UserAgent()

			// Añadir al contexto
			ctx := gwcontext.WithContext(r.Context(), rc)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)

			rc.Finish()
		})
	}
}
