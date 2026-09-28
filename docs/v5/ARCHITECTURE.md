# SentinelFlow V5 Architecture

V5 adds an intelligent execution layer over SentinelFlow’s enterprise control/data/event/provider/observability foundation.

```text
Client
  |
  v
SentinelFlow Gateway
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
- V5 keeps the proven enterprise foundation stable while introducing explicit V5 contracts; internal compatibility packages remain versioned where needed for safe evolution.
