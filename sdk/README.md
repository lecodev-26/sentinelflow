# SentinelFlow SDKs

Official client SDKs for the SentinelFlow gateway.

- [Go SDK](go/README.md)
- [Python SDK](python/README.md)

Both SDKs target the HTTP gateway and keep authentication and request construction explicit. They do not store provider secrets.

## Compatibility

The SDKs are intentionally small and map to the currently implemented SentinelFlow endpoints. Check the API reference before depending on endpoint-specific response fields.
