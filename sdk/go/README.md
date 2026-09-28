# SentinelFlow Go SDK

Small dependency-free client for SentinelFlow.

```go
client := sentinelflow.New("https://gateway.example.com", "sf_...")
resp, err := client.Chat(ctx, sentinelflow.ChatRequest{
    Model: "model-id",
    Messages: []sentinelflow.Message{{Role: "user", Content: "Hello"}},
})
```
