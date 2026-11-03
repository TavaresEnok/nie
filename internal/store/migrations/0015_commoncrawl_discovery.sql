ALTER TABLE jobs DROP CONSTRAINT IF EXISTS jobs_kind_check;
ALTER TABLE jobs ADD CONSTRAINT jobs_kind_check
    CHECK (kind IN (
        'fetch', 'discover_web', 'discover_github', 'discover_gitlab', 'discover_mwmbl',
        'discover_swh', 'discover_commoncrawl', 'investigate', 'revisit'
    ));
