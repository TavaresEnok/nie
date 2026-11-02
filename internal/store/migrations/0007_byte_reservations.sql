-- NIE: reserva atômica do orçamento de bytes da campanha.
-- Antes de baixar, o worker reserva o teto do download; reservas ativas contam
-- como consumo, então workers concorrentes não ultrapassam max_bytes. Reservas
-- expiradas (worker que caiu) deixam de contar sem intervenção.

CREATE TABLE IF NOT EXISTS byte_reservations (
    id bigserial PRIMARY KEY,
    campaign_id uuid NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    bytes bigint NOT NULL CHECK (bytes > 0),
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS byte_reservations_campaign_idx
    ON byte_reservations (campaign_id, expires_at);
