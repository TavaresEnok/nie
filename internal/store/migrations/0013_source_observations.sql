ALTER TABLE jobs DROP CONSTRAINT IF EXISTS jobs_kind_check;
ALTER TABLE jobs ADD CONSTRAINT jobs_kind_check
    CHECK (kind IN ('fetch', 'discover_web', 'discover_github', 'discover_mwmbl', 'discover_swh', 'investigate', 'revisit'));

CREATE TABLE source_observations (
    campaign_id uuid NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    normalized_url text NOT NULL,
    document_id uuid REFERENCES documents(id) ON DELETE SET NULL,
    sha256 text NOT NULL DEFAULT '',
    etag text NOT NULL DEFAULT '',
    last_modified text NOT NULL DEFAULT '',
    first_seen timestamptz NOT NULL DEFAULT now(),
    last_checked timestamptz NOT NULL DEFAULT now(),
    last_changed timestamptz,
    PRIMARY KEY (campaign_id, normalized_url)
);

ALTER TABLE document_diffs ADD COLUMN IF NOT EXISTS campaign_id uuid REFERENCES campaigns(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS document_diffs_campaign_idx ON document_diffs (campaign_id, created_at DESC);
