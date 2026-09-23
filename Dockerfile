# === BUILD STAGE ===
FROM golang:1.27-alpine AS builder

ARG SERVICE=gateway

WORKDIR /build

# Instalar dependencias del sistema
RUN apk add --no-cache git ca-certificates tzdata

# Copiar go.mod y go.sum primero (cache de capas)
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copiar código fuente
COPY . .

# Compilar binario estático del servicio solicitado
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-s -w -extldflags '-static' \
      -X github.com/lecodev-26/sentinelflow/internal/version.Version=3.0.0 \
      -X github.com/lecodev-26/sentinelflow/internal/version.Commit=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown') \
      -X github.com/lecodev-26/sentinelflow/internal/version.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -trimpath \
    -o /out/service \
    ./cmd/${SERVICE}

# === RUNTIME STAGE ===
FROM gcr.io/distroless/static-debian12:nonroot

# Metadatos
LABEL org.opencontainers.image.title="SentinelFlow"
LABEL org.opencontainers.image.description="AI Gateway & Control Plane"
LABEL org.opencontainers.image.version="3.0.0"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.source="https://github.com/lecodev-26/sentinelflow"

# Copiar binario
COPY --from=builder /out/service /service

# Copiar web (para gateway)
COPY --from=builder /build/web /web

# Exponer puertos (gateway: 8080, controlplane: 8081, metrics: 9090)
EXPOSE 8080 8081 9090

# Health check genérico
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/service", "-health"]

# Usuario no-root (distroless ya viene con nonroot)
USER nonroot:nonroot

# Entrypoint
ENTRYPOINT ["/service"]
