-- NIE Fase 3: Autonomia Contínua, Agendamentos, Revisitas e Alertas

CREATE TABLE IF NOT EXISTS schedules (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    query text NOT NULL,
    seed_urls jsonb NOT NULL DEFAULT '[]'::jsonb,
    interval_seconds integer NOT NULL DEFAULT 86400,
    max_depth integer NOT NULL DEFAULT 3,
    max_urls integer NOT NULL DEFAULT 200,
    max_domains integer NOT NULL DEFAULT 20,
    enabled boolean NOT NULL DEFAULT true,
    last_run_at timestamptz,
    next_run_at timestamptz NOT NULL DEFAULT now(),
    last_campaign_id uuid REFERENCES campaigns(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS schedules_active_idx
    ON schedules (enabled, next_run_at);

CREATE TABLE IF NOT EXISTS alerts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id uuid REFERENCES campaigns(id) ON DELETE CASCADE,
    title text NOT NULL,
    message text NOT NULL,
    severity text NOT NULL DEFAULT 'info'
        CHECK (severity IN ('info', 'warning', 'critical')),
    acknowledged boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS alerts_unack_idx
    ON alerts (acknowledged, created_at DESC);

CREATE TABLE IF NOT EXISTS document_diffs (
    id bigserial PRIMARY KEY,
    document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    url text NOT NULL,
    old_sha256 text NOT NULL,
    new_sha256 text NOT NULL,
    summary text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS document_diffs_doc_idx
    ON document_diffs (document_id, created_at DESC);
