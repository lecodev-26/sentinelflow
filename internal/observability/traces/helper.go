package traces

import (
"time"
)

// Tracer facilita la creación de spans
type Tracer struct {
store   *Store
traceID string
}

// NewTracer crea un tracer ligado a un trace
func NewTracer(store *Store, traceID string) *Tracer {
return &Tracer{store: store, traceID: traceID}
}

// SpanBuilder construye un span cómodamente
type SpanBuilder struct {
tracer *Tracer
span   *Span
}

// StartSpan inicia un nuevo span
func (t *Tracer) StartSpan(name string) *SpanBuilder {
return &SpanBuilder{
tracer: t,
span: &Span{
SpanID:     generateSpanID(),
Name:       name,
StartTime:  time.Now(),
Status:     "ok",
Attributes: make(map[string]interface{}),
Events:     []SpanEvent{},
},
}
}

// WithParent asigna un parent
func (b *SpanBuilder) WithParent(parentID string) *SpanBuilder {
b.span.ParentID = parentID
return b
}

// WithAttr añade un atributo
func (b *SpanBuilder) WithAttr(key string, value interface{}) *SpanBuilder {
b.span.Attributes[key] = value
return b
}

// WithError marca el span como error
func (b *SpanBuilder) WithError(err string) *SpanBuilder {
b.span.Status = "error"
b.span.Attributes["error"] = err
return b
}

// AddEvent añade un evento al span
func (b *SpanBuilder) AddEvent(name string, attrs map[string]interface{}) *SpanBuilder {
b.span.Events = append(b.span.Events, SpanEvent{
Timestamp:  time.Now(),
Name:       name,
Attributes: attrs,
})
return b
}

// End finaliza el span
func (b *SpanBuilder) End() {
b.span.EndTime = time.Now()
b.span.Duration = b.span.EndTime.Sub(b.span.StartTime)
if b.tracer != nil && b.tracer.store != nil {
b.tracer.store.AddSpan(b.tracer.traceID, b.span)
}
}

func generateSpanID() string {
return time.Now().Format("150405.000000") + "-" + randomString(6)
}

func randomString(n int) string {
const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
b := make([]byte, n)
for i := range b {
b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
}
return string(b)
}
