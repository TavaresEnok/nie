CREATE TABLE research_questions (
    campaign_id uuid NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    ordinal integer NOT NULL,
    text text NOT NULL,
    origin text NOT NULL,
    status text NOT NULL,
    evidence_note text NOT NULL DEFAULT '',
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (campaign_id, ordinal),
    CONSTRAINT research_questions_status_check CHECK (status IN ('open', 'queued', 'addressed'))
);
