package observability

import (
"context"
"fmt"
"time"

"go.opentelemetry.io/otel"
"go.opentelemetry.io/otel/attribute"
"go.opentelemetry.io/otel/codes"
"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
"go.opentelemetry.io/otel/sdk/resource"
sdktrace "go.opentelemetry.io/otel/sdk/trace"
semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

// InitTracing inicializa OpenTelemetry
func InitTracing(serviceName, endpoint string) error {
if endpoint == "" {
// Si no hay endpoint, usamos un tracer no-op
tracer = otel.Tracer(serviceName)
return nil
}

// Crear exporter OTLP
exporter, err := otlptracegrpc.New(
context.Background(),
otlptracegrpc.WithEndpoint(endpoint),
otlptracegrpc.WithInsecure(),
)
if err != nil {
return fmt.Errorf("failed to create exporter: %w", err)
}

// Crear resource
res, err := resource.New(
context.Background(),
resource.WithAttributes(
semconv.ServiceNameKey.String(serviceName),
attribute.String("service.version", "0.2.0"),
attribute.String("environment", "production"),
),
)
if err != nil {
return fmt.Errorf("failed to create resource: %w", err)
}

// Crear tracer provider
provider := sdktrace.NewTracerProvider(
sdktrace.WithBatcher(exporter),
sdktrace.WithResource(res),
)

// Establecer como global
otel.SetTracerProvider(provider)

// Crear tracer
tracer = otel.Tracer(serviceName)

return nil
}

// GetTracer devuelve el tracer global
func GetTracer() trace.Tracer {
if tracer == nil {
tracer = otel.Tracer("sentinelflow")
}
return tracer
}

// SpanWrapper es un wrapper para facilitar el uso de spans
type SpanWrapper struct {
ctx  context.Context
span trace.Span
}

// StartSpan inicia un nuevo span
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) *SpanWrapper {
ctx, span := GetTracer().Start(ctx, name, opts...)
return &SpanWrapper{
ctx:  ctx,
span: span,
}
}

// End finaliza el span
func (s *SpanWrapper) End() {
s.span.End()
}

// Context devuelve el contexto con el span
func (s *SpanWrapper) Context() context.Context {
return s.ctx
}

// SetAttribute añade un atributo al span
func (s *SpanWrapper) SetAttribute(key string, value interface{}) {
switch v := value.(type) {
case string:
s.span.SetAttributes(attribute.String(key, v))
case int:
s.span.SetAttributes(attribute.Int(key, v))
case int64:
s.span.SetAttributes(attribute.Int64(key, v))
case float64:
s.span.SetAttributes(attribute.Float64(key, v))
case bool:
s.span.SetAttributes(attribute.Bool(key, v))
case time.Duration:
s.span.SetAttributes(attribute.String(key, v.String()))
default:
s.span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", v)))
}
}

// RecordError registra un error en el span
func (s *SpanWrapper) RecordError(err error) {
s.span.RecordError(err)
s.span.SetStatus(codes.Error, err.Error())
}

// SetStatus establece el estado del span
func (s *SpanWrapper) SetStatus(code codes.Code, description string) {
s.span.SetStatus(code, description)
}
