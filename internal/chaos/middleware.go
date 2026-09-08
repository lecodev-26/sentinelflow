package chaos

import (
"net/http"
)

// Middleware es un middleware de chaos engineering
type Middleware struct {
injector *Injector
}

// NewMiddleware crea un nuevo middleware de chaos
func NewMiddleware(injector *Injector) *Middleware {
return &Middleware{injector: injector}
}

// Handler envuelve un handler con chaos
func (m *Middleware) Handler(next http.HandlerFunc) http.HandlerFunc {
return func(w http.ResponseWriter, r *http.Request) {
if m.injector.ShouldFail() {
failure := m.injector.GetFailure()
_ = m.injector.SimulateFailure(failure)
http.Error(w, "Chaos injected", http.StatusInternalServerError)
return
}
next(w, r)
}
}
