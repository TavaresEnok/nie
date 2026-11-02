-- NIE Fase 2: Motor de Evidências, Fabricantes, Firmware e Origin Clusters

CREATE TABLE IF NOT EXISTS vendors (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug text NOT NULL UNIQUE,
    name text NOT NULL,
    cli_family text NOT NULL DEFAULT '',
    aliases text[] NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS device_models (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor_id uuid NOT NULL REFERENCES vendors(id) ON DELETE CASCADE,
    slug text NOT NULL,
    name text NOT NULL,
    family text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (vendor_id, slug)
);

CREATE TABLE IF NOT EXISTS firmware_releases (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor_id uuid NOT NULL REFERENCES vendors(id) ON DELETE CASCADE,
    model_id uuid REFERENCES device_models(id) ON DELETE SET NULL,
    name text NOT NULL,
    version text NOT NULL,
    os_name text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS origin_clusters (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id uuid NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    fingerprint text NOT NULL,
    root_domain text NOT NULL,
    canonical_url text NOT NULL,
    duplicate_count integer NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (campaign_id, fingerprint)
);

CREATE INDEX IF NOT EXISTS origin_clusters_campaign_idx
    ON origin_clusters (campaign_id, root_domain);

CREATE TABLE IF NOT EXISTS claims (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id uuid NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    title text NOT NULL,
    statement text NOT NULL,
    vendor_id uuid REFERENCES vendors(id) ON DELETE SET NULL,
    model_id uuid REFERENCES device_models(id) ON DELETE SET NULL,
    firmware_id uuid REFERENCES firmware_releases(id) ON DELETE SET NULL,
    status text NOT NULL DEFAULT 'unconfirmed'
        CHECK (status IN ('unconfirmed', 'sustained', 'corroborated', 'contested', 'validated_lab')),
    applicability text NOT NULL DEFAULT 'indeterminate'
        CHECK (applicability IN ('confirmed', 'incompatible', 'indeterminate')),
    operational_validation text NOT NULL DEFAULT 'untested'
        CHECK (operational_validation IN ('untested', 'lab_validated', 'failed')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS claims_campaign_idx
    ON claims (campaign_id, status);

CREATE TABLE IF NOT EXISTS evidence (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    claim_id uuid NOT NULL REFERENCES claims(id) ON DELETE CASCADE,
    document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    passage_id bigint REFERENCES passages(id) ON DELETE SET NULL,
    cluster_id uuid REFERENCES origin_clusters(id) ON DELETE SET NULL,
    stance text NOT NULL DEFAULT 'supports'
        CHECK (stance IN ('supports', 'refutes', 'mentions')),
    excerpt text NOT NULL,
    page integer,
    confidence real NOT NULL DEFAULT 0.8,
    is_official_doc boolean NOT NULL DEFAULT false,
    notes text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS evidence_claim_idx
    ON evidence (claim_id, stance);

CREATE TABLE IF NOT EXISTS contradictions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    claim_id uuid NOT NULL REFERENCES claims(id) ON DELETE CASCADE,
    evidence_a_id uuid NOT NULL REFERENCES evidence(id) ON DELETE CASCADE,
    evidence_b_id uuid NOT NULL REFERENCES evidence(id) ON DELETE CASCADE,
    explanation text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Seed initial vendor catalog
INSERT INTO vendors (slug, name, cli_family, aliases) VALUES
    ('huawei', 'Huawei', 'vrp', ARRAY['huawei', 'vrp', 'ne8000', 'netengine', 'smartax', 'ma5800']),
    ('mikrotik', 'MikroTik', 'routeros', ARRAY['mikrotik', 'routeros', 'ros', 'routerboard', 'ccr']),
    ('cisco', 'Cisco', 'ios', ARRAY['cisco', 'ios', 'ios-xr', 'ios-xe', 'nx-os', 'catalyst', 'asr']),
    ('datacom', 'Datacom', 'dmos', ARRAY['datacom', 'dmos', 'dm4270', 'dm4370']),
    ('zte', 'ZTE', 'zxros', ARRAY['zte', 'zxros', 'c300', 'c320', 'zxa10']),
    ('fiberhome', 'FiberHome', 'fiberhome-os', ARRAY['fiberhome', 'an5516', 'an5116']),
    ('vsol', 'VSOL', 'vsol-cli', ARRAY['vsol', 'v1600g', 'v1600d']),
    ('sonicwall', 'SonicWall', 'sonicos', ARRAY['sonicwall', 'sonicos', 'tz', 'nsa']),
    ('hillstone', 'Hillstone', 'stoneos', ARRAY['hillstone', 'stoneos', 'sg-6000'])
ON CONFLICT (slug) DO NOTHING;
