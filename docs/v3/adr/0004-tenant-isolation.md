
ADR-0004: Tenant Isolation

Status: Accepted
Date: 2026-09-18

Context

V2 had tenant isolation issues:

· Cache keys didn't include tenant
· Rate limits used IP only
· Some metrics not scoped

Decision

Every operation is scoped by tenant_id + project_id.

Includes:

· Cache keys: tenant/project/model/policy/request-hash
· Rate limits: per API key + project + tenant
· Dedup: scoped by tenant
· Usage records: tenant_id mandatory
· Audit logs: tenant_id mandatory
· Provider credentials: per tenant
· Routing rules: per tenant

Enforcement

· Middleware sets TenantID in RequestContext
· Storage layer requires TenantID in queries
· Tests verify cross-tenant isolation

Consequences

Positive

· Legal compliance (GDPR, SOC2)
· Safe for SaaS
· Predictable behavior

Negative

· Slightly more storage per record
· More complex queries

Mitigation

· Composite indexes on (tenant_id, ...)
· Partition large tables by tenant
