-- V3.4.7 SCIM Groups
CREATE TABLE IF NOT EXISTS scim_groups (
 id TEXT PRIMARY KEY,
 org_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
 display_name TEXT NOT NULL,
 external_id TEXT,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 UNIQUE(org_id, display_name),
 UNIQUE(org_id, external_id)
);
CREATE INDEX IF NOT EXISTS idx_scim_groups_org ON scim_groups(org_id);
CREATE TABLE IF NOT EXISTS scim_group_members (
 group_id TEXT NOT NULL REFERENCES scim_groups(id) ON DELETE CASCADE,
 user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 PRIMARY KEY(group_id,user_id)
);
CREATE INDEX IF NOT EXISTS idx_scim_group_members_user ON scim_group_members(user_id);
