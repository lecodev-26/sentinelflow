-- V3.4.6 Enterprise hierarchy: Organization -> Business Unit -> Project -> Environment.
-- business_unit_id is nullable for backwards compatibility with existing projects.
CREATE TABLE IF NOT EXISTS business_units (
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    settings JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, name)
);
CREATE INDEX IF NOT EXISTS idx_business_units_org ON business_units(org_id);

ALTER TABLE projects ADD COLUMN IF NOT EXISTS business_unit_id TEXT REFERENCES business_units(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_projects_business_unit ON projects(business_unit_id);
