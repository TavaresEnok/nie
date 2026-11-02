-- NIE: um claim por (campanha, título), sem perder histórico.
-- Antes desta migração, cada execução de /investigate inseria claims novos.
-- As duplicatas são MESCLADAS no claim mantido (o validado em bancada, se
-- houver, senão o mais antigo): evidências e contradições passam para ele e a
-- fusão fica registrada em claim_merges. Nenhuma evidência é apagada; tudo roda
-- na mesma transação da migração.

CREATE TABLE IF NOT EXISTS claim_merges (
    merged_id uuid PRIMARY KEY,
    kept_id uuid NOT NULL REFERENCES claims(id) ON DELETE CASCADE,
    campaign_id uuid NOT NULL,
    title text NOT NULL,
    merged_status text NOT NULL,
    merged_operational_validation text NOT NULL,
    merged_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO claim_merges (merged_id, kept_id, campaign_id, title, merged_status, merged_operational_validation)
SELECT r.id, r.kept_id, r.campaign_id, r.title, r.status, r.operational_validation
FROM (
    SELECT id, campaign_id, title, status, operational_validation,
           first_value(id) OVER (
               PARTITION BY campaign_id, title
               ORDER BY (operational_validation <> 'untested') DESC, created_at ASC, id ASC
           ) AS kept_id
    FROM claims
) r
WHERE r.id <> r.kept_id
ON CONFLICT (merged_id) DO NOTHING;

UPDATE evidence e SET claim_id = m.kept_id
FROM claim_merges m WHERE e.claim_id = m.merged_id;

UPDATE contradictions c SET claim_id = m.kept_id
FROM claim_merges m WHERE c.claim_id = m.merged_id;

DELETE FROM claims c USING claim_merges m WHERE c.id = m.merged_id;

CREATE UNIQUE INDEX IF NOT EXISTS claims_campaign_title_uq
    ON claims (campaign_id, title);
