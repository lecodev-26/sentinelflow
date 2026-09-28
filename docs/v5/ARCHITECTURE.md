# SentinelFlow V5 Architecture

V5 layers intelligent execution over the V4 control/data/event/provider/observability planes.

```text
Client
  |
  v
V4 Gateway
  |
  +--> AI Understanding ------> Requirements
  |
  +--> Security 2.0 ----------> Risk decision
  |
  +--> Governance ------------> Allowed execution envelope
  |
  +--> Adaptive Routing ------> Model/provider/region
  |
  +--> Agent Runtime ---------> Steps / approvals
  |       |
  |       +--> Tool Gateway
  |       +--> Memory
  |       +--> Knowledge/RAG
  |
  +--> Provider execution
  |
  +--> Evaluation / semantic tracing
  |
  +--> FinOps / Learning signals
  |
  v
Response
```

## Hard boundaries

- Tenant and project isolation is explicit in memory, knowledge and Zero Trust decisions.
- Agent steps do not bypass tool authorization or approval requirements.
- Adaptive learning records signals; it does not silently mutate production policy.
- Governance constrains execution; it does not replace authentication/authorization.
- V5 contracts are additive to V4 until a migration replaces the legacy implementation path.
