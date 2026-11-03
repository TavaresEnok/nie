package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"time"
)

type ClaimRevision struct {
	ID         string          `json:"id"`
	CampaignID string          `json:"campaign_id"`
	ClaimID    string          `json:"claim_id"`
	Snapshot   json.RawMessage `json:"snapshot"`
	CreatedAt  time.Time       `json:"created_at"`
}

func archiveClaimsTx(ctx context.Context, tx pgx.Tx, campaignID string) error {
	rows, err := tx.Query(ctx, `SELECT c.id::text,to_jsonb(c)||jsonb_build_object(
        'evidences',COALESCE((SELECT jsonb_agg(to_jsonb(e) ORDER BY e.id) FROM evidence e WHERE e.claim_id=c.id),'[]'::jsonb),
        'contradictions',COALESCE((SELECT jsonb_agg(to_jsonb(x) ORDER BY x.id) FROM contradictions x WHERE x.claim_id=c.id),'[]'::jsonb))
        FROM claims c WHERE campaign_id=$1::uuid ORDER BY c.id`, campaignID)
	if err != nil {
		return err
	}
	type item struct {
		id       string
		snapshot []byte
	}
	var snapshots []item
	for rows.Next() {
		var i item
		if err := rows.Scan(&i.id, &i.snapshot); err != nil {
			rows.Close()
			return err
		}
		snapshots = append(snapshots, i)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, i := range snapshots {
		hash := sha256.Sum256(i.snapshot)
		if _, err := tx.Exec(ctx, `INSERT INTO claim_revisions(campaign_id,claim_id,snapshot_sha256,snapshot) VALUES($1::uuid,$2::uuid,$3,$4) ON CONFLICT DO NOTHING`, campaignID, i.id, hex.EncodeToString(hash[:]), i.snapshot); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ListClaimRevisions(ctx context.Context, campaignID string, limit int) ([]ClaimRevision, error) {
	if !validUUID(campaignID) {
		return nil, ErrNotFound
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `SELECT id::text,campaign_id::text,claim_id::text,snapshot,created_at FROM claim_revisions WHERE campaign_id=$1::uuid ORDER BY created_at DESC,id LIMIT $2`, campaignID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ClaimRevision{}
	for rows.Next() {
		var r ClaimRevision
		if err := rows.Scan(&r.ID, &r.CampaignID, &r.ClaimID, &r.Snapshot, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
