# === BUILD STAGE ===
FROM golang:1.27-alpine AS builder

WORKDIR /build

# Instalar dependencias del sistema
RUN apk add --no-cache git ca-certificates tzdata

# Copiar go.mod y go.sum primero (cache de capas)
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copiar código fuente
COPY . .

# Compilar binario estático
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w -extldflags '-static'" \
    -trimpath \
    -o sentinelflow cmd/proxy/main.go

# === RUNTIME STAGE ===
FROM gcr.io/distroless/static-debian12:nonroot

# Metadatos
LABEL org.opencontainers.image.title="SentinelFlow"
LABEL org.opencontainers.image.description="AI Gateway & Control Plane"
LABEL org.opencontainers.image.version="0.3.0"
LABEL org.opencontainers.image.licenses="MIT"

# Copiar binario
COPY --from=builder /build/sentinelflow /sentinelflow

# Copiar configs y web
COPY --from=builder /build/configs /configs
COPY --from=builder /build/web /web

# Exponer puertos
EXPOSE 8080 8081 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/sentinelflow", "-health"]

# Usuario no-root (ya es nonroot por distroless)
USER nonroot:nonroot

# Entrypoint
ENTRYPOINT ["/sentinelflow"]
CMD ["-config", "/configs/rules.yaml", "-port", "8080"]
