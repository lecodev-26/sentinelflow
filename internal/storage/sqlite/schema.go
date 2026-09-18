package sqlite

// Schema contiene el DDL de SQLite
const Schema = `
CREATE TABLE IF NOT EXISTS organizations (
id TEXT PRIMARY KEY,
name TEXT NOT NULL,
description TEXT,
settings TEXT DEFAULT '{}',
created_at DATETIME NOT NULL,
updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS projects (
id TEXT PRIMARY KEY,
org_id TEXT NOT NULL,
name TEXT NOT NULL,
description TEXT,
settings TEXT DEFAULT '{}',
created_at DATETIME NOT NULL,
updated_at DATETIME NOT NULL,
FOREIGN KEY (org_id) REFERENCES organizations(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_projects_org ON projects(org_id);

CREATE TABLE IF NOT EXISTS users (
id TEXT PRIMARY KEY,
email TEXT NOT NULL UNIQUE,
name TEXT NOT NULL,
role TEXT NOT NULL,
org_id TEXT NOT NULL,
active INTEGER NOT NULL DEFAULT 1,
created_at DATETIME NOT NULL,
updated_at DATETIME NOT NULL,
FOREIGN KEY (org_id) REFERENCES organizations(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_users_org ON users(org_id);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

CREATE TABLE IF NOT EXISTS api_keys (
id TEXT PRIMARY KEY,
key_hash TEXT NOT NULL UNIQUE,
key_prefix TEXT NOT NULL,
name TEXT NOT NULL,
user_id TEXT NOT NULL,
org_id TEXT NOT NULL,
project_id TEXT,
scopes TEXT DEFAULT '[]',
active INTEGER NOT NULL DEFAULT 1,
created_at DATETIME NOT NULL,
updated_at DATETIME NOT NULL,
last_used DATETIME,
expires_at DATETIME NOT NULL,
rotated_from TEXT,
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_apikeys_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_apikeys_user ON api_keys(user_id);

CREATE TABLE IF NOT EXISTS budgets (
id TEXT PRIMARY KEY,
tenant_id TEXT NOT NULL UNIQUE,
monthly_limit REAL NOT NULL,
spent REAL NOT NULL DEFAULT 0,
month_start DATETIME NOT NULL,
reset_date DATETIME NOT NULL,
updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS policies (
id TEXT PRIMARY KEY,
name TEXT NOT NULL,
description TEXT,
tenant_id TEXT NOT NULL,
priority INTEGER DEFAULT 100,
enabled INTEGER NOT NULL DEFAULT 1,
rules TEXT NOT NULL,
created_at DATETIME NOT NULL,
updated_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_policies_tenant ON policies(tenant_id);

CREATE TABLE IF NOT EXISTS audit_events (
id TEXT PRIMARY KEY,
timestamp DATETIME NOT NULL,
type TEXT NOT NULL,
tenant_id TEXT,
user_id TEXT,
request_id TEXT,
action TEXT NOT NULL,
result TEXT NOT NULL,
reason TEXT,
metadata TEXT DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS idx_audit_tenant ON audit_events(tenant_id);
CREATE INDEX IF NOT EXISTS idx_audit_request ON audit_events(request_id);
CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_events(timestamp);

CREATE TABLE IF NOT EXISTS usage_records (
id TEXT PRIMARY KEY,
request_id TEXT NOT NULL,
tenant_id TEXT NOT NULL,
project_id TEXT,
provider TEXT NOT NULL,
model TEXT NOT NULL,
input_tokens INTEGER NOT NULL DEFAULT 0,
output_tokens INTEGER NOT NULL DEFAULT 0,
total_tokens INTEGER NOT NULL DEFAULT 0,
cost_usd REAL NOT NULL DEFAULT 0,
latency_ms INTEGER NOT NULL DEFAULT 0,
status TEXT NOT NULL,
timestamp DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_usage_tenant ON usage_records(tenant_id);
CREATE INDEX IF NOT EXISTS idx_usage_timestamp ON usage_records(timestamp);

CREATE TABLE IF NOT EXISTS schema_migrations (
version INTEGER PRIMARY KEY,
applied_at DATETIME NOT NULL
);
`
