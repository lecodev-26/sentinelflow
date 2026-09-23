-- Security: Enable Row Level Security on all public tables
--
-- This blocks the automatic PostgREST endpoint (Supabase /rest/v1/*)
-- for the 'anon' role, but does NOT affect the 'postgres' role used
-- by the SentinelFlow connection string.
--
-- See: https://supabase.com/docs/guides/database/postgres/row-level-security

ALTER TABLE IF EXISTS organizations ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS projects ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS users ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS api_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS usage_records ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS budgets ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS config_versions ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS schema_migrations ENABLE ROW LEVEL SECURITY;
