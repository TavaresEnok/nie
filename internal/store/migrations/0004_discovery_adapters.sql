-- 0004_discovery_adapters.sql
-- Expande o check constraint de jobs para aceitar Mwmbl e Software Heritage

ALTER TABLE jobs DROP CONSTRAINT IF EXISTS jobs_kind_check;
ALTER TABLE jobs ADD CONSTRAINT jobs_kind_check 
    CHECK (kind IN ('fetch', 'discover_web', 'discover_github', 'discover_mwmbl', 'discover_swh'));
