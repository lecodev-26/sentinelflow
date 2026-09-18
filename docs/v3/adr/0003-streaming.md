
ADR-0003: Real SSE Streaming

Status: Accepted
Date: 2026-09-18

Context

V2 Stream() waited for full response before yielding a single chunk.

This is not real streaming.

Decision

Implement real SSE streaming end-to-end:

```
Client → Gateway → Provider
              ← tokens
              ← tokens
              ← tokens
              ← [DONE]
```

Requirements

· Use http.Flusher correctly
· Don't buffer provider response
· Pass through SSE events as they arrive
· Preserve TTFT (Time To First Token)
· Handle client cancellation
· Handle provider mid-stream failure (fail, don't retry)

Consequences

Positive

· Real streaming UX
· Lower TTFT
· Competitive feature

Negative

· Cannot retry after first byte
· Accounting must handle partial usage
· Cache must be skipped for streaming

Mitigation

· Retry BEFORE first byte only
· Publish StreamInterrupted event
· Skip cache for stream: true
  EOF
