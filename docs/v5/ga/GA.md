# SentinelFlow V5 GA

SentinelFlow V5 is the current release line. It preserves the enterprise gateway/control-plane foundation and adds an intelligent AI execution layer.

## V5 capabilities

- Intelligent AI request understanding and adaptive routing
- Agent runtime and universal tool gateway
- Memory and knowledge/RAG with explicit tenant/project scope
- AI Security 2.0
- AI FinOps and semantic observability
- Evaluation and prompt management
- Multimodal request primitives
- Governance and plugin extensibility
- Global routing, self-healing, learning signals and Zero Trust AI
- Capacity/autoscaling foundations

## Release boundary

V5.0.0 was the initial GA software release. The current maintenance line must be released from the hardened post-GA branch rather than reusing the original V5.0.0 tag.

Production operation still requires environment-specific PostgreSQL/Redis, provider credentials, identity configuration, secret management, backups, monitoring, load testing and operational verification.

The public Go client is the nested module under `sdk/go`; the repository root is the SentinelFlow server/application module.

See [production readiness](production-readiness.md).
