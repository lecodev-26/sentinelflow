# SentinelFlow Go SDK

The SentinelFlow Go SDK is the public, dependency-free Go client for the SentinelFlow API.

## Install

```bash
go get github.com/lecodev-26/sentinelflow/sdk/go@v0.1.0
```

## Quick start

```go
package main

import (
    "context"
    "fmt"

    sentinelflow "github.com/lecodev-26/sentinelflow/sdk/go/sentinelflow"
)

func main() {
    client, err := sentinelflow.New(sentinelflow.Config{BaseURL: "http://localhost:8080", APIKey: "your-api-key"})
    if err != nil {
        panic(err)
    }

    response, err := client.Chat(context.Background(), sentinelflow.ChatRequest{Model: "default", Input: "Hello SentinelFlow"})
    if err != nil {
        panic(err)
    }

    fmt.Println(response)
}
```

## Streaming

Use `ChatStream` when `ChatRequest.Stream` is enabled. The callback receives each raw SSE event payload.

```go
err := client.ChatStream(ctx, sentinelflow.ChatRequest{Model: "default", Input: "Hello", Stream: true}, func(event []byte) error {
    fmt.Println(string(event))
    return nil
})
```

## Versioning

The Go SDK is versioned independently from the SentinelFlow server. Releases use Go module-compatible semantic versions and repository tags in the form `sdk/go/vX.Y.Z`.

- `v0.x`: public API may evolve between minor versions; breaking changes are documented.
- `v1.x`: stable API compatibility policy begins.
- Patch releases contain backward-compatible fixes.
- New public APIs are added without requiring a server major release.

The server release line and SDK release line are therefore intentionally independent.

## Documentation

API documentation is published automatically through pkg.go.dev from the public Go module proxy.
