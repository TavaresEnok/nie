-- NIE Fase 2.1: Alinhamento de schema para claims e evidências estruturadas

ALTER TABLE claims DROP CONSTRAINT IF EXISTS claims_status_check;

ALTER TABLE claims DROP CONSTRAINT IF EXISTS claims_applicability_check;

ALTER TABLE claims
    ADD CONSTRAINT claims_status_check
    CHECK (status IN ('unconfirmed', 'documented', 'sustained', 'corroborated', 'contested', 'validated_lab'));

ALTER TABLE claims
    ADD CONSTRAINT claims_applicability_check
    CHECK (applicability IN ('confirmed', 'applicable', 'incompatible', 'indeterminate'));

ALTER TABLE evidence ADD COLUMN IF NOT EXISTS vendor_match boolean NOT NULL DEFAULT false;
ALTER TABLE evidence ADD COLUMN IF NOT EXISTS model_match boolean NOT NULL DEFAULT false;
ALTER TABLE evidence ADD COLUMN IF NOT EXISTS firmware_match boolean NOT NULL DEFAULT false;
ALTER TABLE evidence ADD COLUMN IF NOT EXISTS feature_match boolean NOT NULL DEFAULT false;
ALTER TABLE evidence ADD COLUMN IF NOT EXISTS mode_match boolean NOT NULL DEFAULT false;
ALTER TABLE evidence ADD COLUMN IF NOT EXISTS prerequisites_match boolean NOT NULL DEFAULT false;
ALTER TABLE evidence ADD COLUMN IF NOT EXISTS source_span text NOT NULL DEFAULT '';
ALTER TABLE evidence ADD COLUMN IF NOT EXISTS validation_run_id text;
