CREATE TABLE claim_revisions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id uuid NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    claim_id uuid NOT NULL,
    snapshot_sha256 text,
    snapshot jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE(campaign_id,claim_id,snapshot_sha256)
);
CREATE INDEX claim_revisions_campaign_idx ON claim_revisions(campaign_id,created_at);
-- Preserve the current state of existing installations before future updates.
INSERT INTO claim_revisions(campaign_id,claim_id,snapshot)
SELECT c.campaign_id,c.id,to_jsonb(c)||jsonb_build_object(
    'evidences',COALESCE((SELECT jsonb_agg(to_jsonb(e) ORDER BY e.id) FROM evidence e WHERE e.claim_id=c.id),'[]'::jsonb),
    'contradictions',COALESCE((SELECT jsonb_agg(to_jsonb(x) ORDER BY x.id) FROM contradictions x WHERE x.claim_id=c.id),'[]'::jsonb))
FROM claims c;
