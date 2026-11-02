-- NIE MVP schema v1
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS campaigns (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    query text NOT NULL,
    status text NOT NULL DEFAULT 'queued'
        CHECK (status IN ('queued', 'running', 'paused', 'completed', 'failed')),
    seed_urls jsonb NOT NULL DEFAULT '[]'::jsonb,
    max_depth integer NOT NULL DEFAULT 3,
    max_urls integer NOT NULL DEFAULT 200,
    max_domains integer NOT NULL DEFAULT 20,
    max_bytes bigint NOT NULL DEFAULT 1073741824,
    min_domain_interval_seconds integer NOT NULL DEFAULT 10,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz
);

CREATE TABLE IF NOT EXISTS jobs (
    id bigserial PRIMARY KEY,
    campaign_id uuid NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    kind text NOT NULL CHECK (kind IN ('fetch', 'discover_web', 'discover_github')),
    url text,
    url_hash text,
    domain text NOT NULL DEFAULT '',
    depth integer NOT NULL DEFAULT 0,
    priority integer NOT NULL DEFAULT 100,
    parent_url text,
    reason text,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    status text NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'running', 'done', 'dead')),
    attempts integer NOT NULL DEFAULT 0,
    max_attempts integer NOT NULL DEFAULT 3,
    scheduled_at timestamptz NOT NULL DEFAULT now(),
    lease_until timestamptz,
    last_error text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS jobs_unique_url
    ON jobs (campaign_id, kind, url_hash)
    WHERE url_hash IS NOT NULL;

CREATE INDEX IF NOT EXISTS jobs_claim_idx
    ON jobs (status, scheduled_at, priority, id);
CREATE INDEX IF NOT EXISTS jobs_campaign_idx
    ON jobs (campaign_id, status);
CREATE INDEX IF NOT EXISTS jobs_domain_idx
    ON jobs (campaign_id, domain);

CREATE TABLE IF NOT EXISTS documents (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    sha256 text NOT NULL UNIQUE,
    mime text NOT NULL,
    title text NOT NULL DEFAULT '',
    size bigint NOT NULL,
    text text NOT NULL DEFAULT '',
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS document_versions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    campaign_id uuid NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    url text NOT NULL,
    normalized_url text NOT NULL,
    domain text NOT NULL,
    status_code integer NOT NULL DEFAULT 0,
    etag text NOT NULL DEFAULT '',
    last_modified text NOT NULL DEFAULT '',
    fetched_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (campaign_id, document_id, normalized_url)
);

CREATE INDEX IF NOT EXISTS document_versions_campaign_idx
    ON document_versions (campaign_id, fetched_at DESC);
CREATE INDEX IF NOT EXISTS document_versions_domain_idx
    ON document_versions (domain);
CREATE INDEX IF NOT EXISTS document_versions_url_idx
    ON document_versions (normalized_url);

CREATE TABLE IF NOT EXISTS passages (
    id bigserial PRIMARY KEY,
    document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    ordinal integer NOT NULL,
    page integer,
    text text NOT NULL,
    search_vector tsvector GENERATED ALWAYS AS (to_tsvector('simple', text)) STORED
);

CREATE INDEX IF NOT EXISTS passages_fts_idx
    ON passages USING gin (search_vector);
CREATE INDEX IF NOT EXISTS passages_trgm_idx
    ON passages USING gin (text gin_trgm_ops);
CREATE INDEX IF NOT EXISTS passages_document_idx
    ON passages (document_id, ordinal);

CREATE TABLE IF NOT EXISTS discovery_edges (
    id bigserial PRIMARY KEY,
    campaign_id uuid NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    from_url text NOT NULL DEFAULT '',
    to_url text NOT NULL,
    to_domain text NOT NULL DEFAULT '',
    kind text NOT NULL DEFAULT 'link',
    depth integer NOT NULL DEFAULT 0,
    reason text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (campaign_id, from_url, to_url, kind)
);

CREATE INDEX IF NOT EXISTS discovery_edges_campaign_idx
    ON discovery_edges (campaign_id, created_at);

CREATE TABLE IF NOT EXISTS domain_state (
    domain text PRIMARY KEY,
    next_allowed_at timestamptz NOT NULL DEFAULT now(),
    last_status integer,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS schema_migrations (
    version text PRIMARY KEY,
    applied_at timestamptz NOT NULL DEFAULT now()
);
