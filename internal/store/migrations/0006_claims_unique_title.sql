-- NIE: um claim por (campanha, título). Antes desta migração, cada execução de
-- /investigate inseria claims novos; remove as duplicatas (mantendo o validado
-- em bancada, se houver, ou o mais antigo) antes de impor a unicidade, que
-- também protege contra investigações concorrentes.

DELETE FROM claims c
USING (
    SELECT id,
           row_number() OVER (
               PARTITION BY campaign_id, title
               ORDER BY (operational_validation <> 'untested') DESC, created_at ASC, id ASC
           ) AS rn
    FROM claims
) d
WHERE c.id = d.id AND d.rn > 1;

CREATE UNIQUE INDEX IF NOT EXISTS claims_campaign_title_uq
    ON claims (campaign_id, title);
