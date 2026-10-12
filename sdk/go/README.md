# SentinelFlow Go SDK

Small dependency-free client for the SentinelFlow HTTP gateway. The Go SDK is versioned independently from the SentinelFlow server.

## Install

```bash
go get github.com/lecodev-26/sentinelflow/sdk/go@v0.1.1
```

The published SDK release is `v0.1.1` (tag: `sdk/go/v0.1.1`). The `v6` Git branch is server development and does not change the SDK module version by itself.

## Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    sentinelflow "github.com/lecodev-26/sentinelflow/sdk/go/sentinelflow"
)

func main() {
    client := sentinelflow.New("https://gateway.example.com", "sf_...")
    resp, err := client.Chat(context.Background(), sentinelflow.ChatRequest{
        Model: "model-id",
        Messages: []sentinelflow.Message{{Role: "user", Content: "Hello"}},
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(resp)
}
```

The SDK currently exposes chat completions, Responses API requests and model listing. Check the server API reference before depending on endpoint-specific response fields.

## Versioning

SDK tags use `sdk/go/vX.Y.Z` and releases are independent from server tags. The `v0.x` API may evolve; compatibility guarantees for a stable `v1.x` API begin with its release.
