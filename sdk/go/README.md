# SentinelFlow Go SDK

Public, dependency-free Go client for the SentinelFlow HTTP API.

The repository root is the SentinelFlow server/application module. The supported public Go client is the nested module:

`github.com/lecodev-26/sentinelflow/sdk/go`

## Install

    go get github.com/lecodev-26/sentinelflow/sdk/go

## Basic usage

    client := sentinelflow.New("https://gateway.example.com", "sf_...")
    resp, err := client.Chat(ctx, sentinelflow.ChatRequest{
        Model: "model-id",
        Messages: []sentinelflow.Message{{Role: "user", Content: "Hello"}},
    })

Use a context with an appropriate deadline or cancellation policy for production workloads. The default client timeout is 60 seconds; provide a custom `HTTPClient` when you need different transport or timeout settings.

## Streaming

Streaming requests use `ChatStream`, which exposes Server-Sent Events as raw event payloads:

    err := client.ChatStream(ctx, sentinelflow.ChatRequest{
        Model: "model-id",
        Messages: []sentinelflow.Message{{Role: "user", Content: "Hello"}},
    }, func(event sentinelflow.StreamEvent) error {
        // event.Data contains the SSE data payload.
        return nil
    })

Calling `Chat` with `Stream: true` returns `ErrStreamingRequiresChatStream`.

## API surface

- `New` creates a client with a 60-second default HTTP timeout.
- `Chat` calls `/v1/chat/completions`.
- `ChatStream` calls `/v1/chat/completions` using SSE.
- `Responses` calls `/v1/responses`.
- `Models` calls `/v1/models`.
- `Error` exposes HTTP status, error type and message from API failures.

## Versioning

The SDK is versioned independently from the SentinelFlow server module because it is a nested Go module. Public SDK tags use the module path prefix required by Go's nested-module versioning rules.

CI runs SDK tests, race detection and vetting alongside the main server checks.
