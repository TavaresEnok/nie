ALTER TABLE jobs DROP CONSTRAINT IF EXISTS jobs_kind_check;
ALTER TABLE jobs ADD CONSTRAINT jobs_kind_check
    CHECK (kind IN ('fetch', 'discover_web', 'discover_github', 'discover_mwmbl', 'discover_swh', 'investigate'));

CREATE TABLE research_rounds (
    job_id bigint PRIMARY KEY REFERENCES jobs(id) ON DELETE CASCADE,
    campaign_id uuid NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    round_number integer NOT NULL,
    decision text NOT NULL,
    reason text NOT NULL,
    claims_count integer NOT NULL,
    passages_count integer NOT NULL,
    enqueued_queries integer NOT NULL,
    leads jsonb NOT NULL DEFAULT '[]',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX research_rounds_campaign_idx ON research_rounds(campaign_id, round_number);
