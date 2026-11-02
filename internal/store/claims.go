package store

import (
	"context"
	"fmt"

	"nie/internal/domain"
)

func (s *Store) ListVendors(ctx context.Context) ([]domain.Vendor, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, slug, name, cli_family, aliases, created_at
		FROM vendors
		ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("listar vendors: %w", err)
	}
	defer rows.Close()

	var out []domain.Vendor
	for rows.Next() {
		var v domain.Vendor
		if err := rows.Scan(&v.ID, &v.Slug, &v.Name, &v.CLIFamily, &v.Aliases, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) SaveClaims(ctx context.Context, claims []domain.Claim) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("iniciar tx de claims: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, c := range claims {
		var claimID string
		err := tx.QueryRow(ctx, `
			INSERT INTO claims (campaign_id, title, statement, status, applicability, operational_validation)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id::text`,
			c.CampaignID, c.Title, c.Statement, c.Status, c.Applicability, c.OperationalValidation).Scan(&claimID)
		if err != nil {
			return fmt.Errorf("inserir claim: %w", err)
		}

		for _, ev := range c.Evidences {
			_, err = tx.Exec(ctx, `
				INSERT INTO evidence (
					claim_id, document_id, passage_id, stance, excerpt, page, confidence, is_official_doc, notes,
					vendor_match, model_match, firmware_match, feature_match, mode_match, prerequisites_match, source_span, validation_run_id
				)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`,
				claimID, ev.DocumentID, ev.PassageID, ev.Stance, ev.Excerpt, ev.Page, ev.Confidence, ev.IsOfficialDoc, ev.Notes,
				ev.VendorMatch, ev.ModelMatch, ev.FirmwareMatch, ev.FeatureMatch, ev.ModeMatch, ev.PrerequisitesMatch, ev.SourceSpan, ev.ValidationRunID)
			if err != nil {
				return fmt.Errorf("inserir evidence: %w", err)
			}
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) ListClaimsByCampaign(ctx context.Context, campaignID string) ([]domain.Claim, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.id::text, c.campaign_id::text, c.title, c.statement,
		       c.vendor_id::text, COALESCE(v.name, '') as vendor_name,
		       c.model_id::text, c.firmware_id::text, c.status,
		       c.applicability, c.operational_validation, c.created_at, c.updated_at
		FROM claims c
		LEFT JOIN vendors v ON v.id = c.vendor_id
		WHERE c.campaign_id = $1
		ORDER BY c.created_at ASC`, campaignID)
	if err != nil {
		return nil, fmt.Errorf("listar claims da campanha: %w", err)
	}
	defer rows.Close()

	var claims []domain.Claim
	for rows.Next() {
		var c domain.Claim
		var vID, mID, fID *string
		if err := rows.Scan(&c.ID, &c.CampaignID, &c.Title, &c.Statement,
			&vID, &c.VendorName, &mID, &fID, &c.Status,
			&c.Applicability, &c.OperationalValidation, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		c.VendorID = vID
		c.ModelID = mID
		c.FirmwareID = fID

		evs, err := s.listEvidenceByClaim(ctx, c.ID)
		if err != nil {
			return nil, err
		}
		c.Evidences = evs
		claims = append(claims, c)
	}
	return claims, rows.Err()
}

func (s *Store) listEvidenceByClaim(ctx context.Context, claimID string) ([]domain.Evidence, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, claim_id::text, document_id::text, passage_id, cluster_id::text,
		       stance, excerpt, page, confidence, is_official_doc, notes,
		       vendor_match, model_match, firmware_match, feature_match, mode_match, prerequisites_match,
		       COALESCE(source_span, ''), validation_run_id, created_at
		FROM evidence
		WHERE claim_id = $1
		ORDER BY created_at ASC`, claimID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Evidence
	for rows.Next() {
		var ev domain.Evidence
		var pID *int64
		var cID *string
		if err := rows.Scan(
			&ev.ID, &ev.ClaimID, &ev.DocumentID, &pID, &cID,
			&ev.Stance, &ev.Excerpt, &ev.Page, &ev.Confidence, &ev.IsOfficialDoc, &ev.Notes,
			&ev.VendorMatch, &ev.ModelMatch, &ev.FirmwareMatch, &ev.FeatureMatch, &ev.ModeMatch, &ev.PrerequisitesMatch,
			&ev.SourceSpan, &ev.ValidationRunID, &ev.CreatedAt); err != nil {
			return nil, err
		}
		ev.PassageID = pID
		ev.ClusterID = cID
		out = append(out, ev)
	}
	return out, rows.Err()
}

func (s *Store) RecordOriginCluster(ctx context.Context, campaignID, fingerprint, rootDomain, canonicalURL string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO origin_clusters (campaign_id, fingerprint, root_domain, canonical_url, duplicate_count)
		VALUES ($1, $2, $3, $4, 1)
		ON CONFLICT (campaign_id, fingerprint)
		DO UPDATE SET duplicate_count = origin_clusters.duplicate_count + 1
		RETURNING id::text`, campaignID, fingerprint, rootDomain, canonicalURL).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("gravar origin cluster: %w", err)
	}
	return id, nil
}
