
ADR-0005: Versioned Policy Engine

Status: Accepted
Date: 2026-09-18

Context

V2 policy was scattered in Go code with if...if...if.

Hard to:

· Change without deploying
· Roll back
· Audit
· Test independently

Decision

Introduce a versioned Policy Engine.

Model

```yaml
policy:
  name: production-ai
  version: 3
  
  deny:
    - model: "*"
      when:
        data_classification: "restricted"
  
  limits:
    max_tokens: 8192
  
  routing:
    allowed_models:
      - gpt-*
      - claude-*
  
  security:
    pii_detection: true
    prompt_injection: true
  
  budget:
    daily: 100
```

Pipeline

```
Request
  ↓
Policy Compiler (YAML → bytecode)
  ↓
Compiled Policy
  ↓
Evaluator (per request)
  ↓
ALLOW / DENY / MODIFY / ROUTE
```

Versioning

· Policies are immutable once published
· Each request uses a specific policy_version
· Config changes produce a new version
· Rollback = point to previous version

Consequences

Positive

· Change without deploy
· Full audit trail
· Rollback
· Test policies independently

Negative

· Compiler complexity
· Evaluation overhead

Mitigation

· Cache compiled policies in memory
· Recompile only on version change
