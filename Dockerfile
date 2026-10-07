FROM golang:1.27-alpine AS build
ARG SERVICE=gateway
ARG VERSION=5.0.1
WORKDIR /src
RUN apk add --no-cache ca-certificates git
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X github.com/lecodev-26/sentinelflow/internal/version.Version=${VERSION} -X github.com/lecodev-26/sentinelflow/internal/version.Commit=$(git rev-parse --short HEAD)" -o /out/service ./cmd/${SERVICE}

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/service /app/service
COPY --from=build /src/web /app/web
USER nonroot:nonroot
EXPOSE 8080 8081
ENTRYPOINT ["/app/service"]
