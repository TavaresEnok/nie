package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

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

// SaveClaims grava o resultado de uma investigação de forma idempotente:
//   - claims são identificados por (campanha, título); reexecutar /investigate
//     atualiza o claim existente e substitui suas evidências, sem duplicar;
//   - claims não reproduzidos e ainda "untested" são removidos; os que já têm
//     validação operacional (lab_validated/failed) nunca são apagados;
//   - cada fingerprint de origem vira uma linha de origin_clusters e as
//     evidências gravam cluster_id, de modo que a independência sobrevive ao
//     recarregamento do banco.
func (s *Store) SaveClaims(ctx context.Context, claims []domain.Claim) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("iniciar tx de claims: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	campaigns := make(map[string][]string) // campanha -> IDs de claims mantidos
	clusters := make(map[string]string)    // campanha|fingerprint -> UUID
	touched := make(map[string]bool)       // UUIDs de clusters usados

	for _, c := range claims {
		// A unicidade (campaign_id, title) é imposta pelo banco (migração 0006),
		// então investigações concorrentes não duplicam. Claims com validação
		// operacional (lab_validated/failed) são imutáveis: a nova investigação
		// não altera status nem apaga as evidências que sustentam a validação.
		var claimID string
		err := tx.QueryRow(ctx, `
			INSERT INTO claims (campaign_id, title, statement, status, applicability, operational_validation)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (campaign_id, title) DO UPDATE
			SET statement = EXCLUDED.statement, status = EXCLUDED.status,
			    applicability = EXCLUDED.applicability, updated_at = now()
			WHERE claims.operational_validation = 'untested'
			RETURNING id::text`,
			c.CampaignID, c.Title, c.Statement, c.Status, c.Applicability, c.OperationalValidation).Scan(&claimID)
		switch {
		case err == nil:
			// Inserido ou atualizado: as evidências são substituídas pelo novo conjunto.
			if _, err := tx.Exec(ctx, `DELETE FROM evidence WHERE claim_id=$1::uuid`, claimID); err != nil {
				return fmt.Errorf("limpar evidências do claim: %w", err)
			}
		case errors.Is(err, pgx.ErrNoRows):
			// Claim validado: mantém-no intacto e não grava evidências novas.
			if err := tx.QueryRow(ctx, `SELECT id::text FROM claims WHERE campaign_id=$1::uuid AND title=$2`,
				c.CampaignID, c.Title).Scan(&claimID); err != nil {
				return fmt.Errorf("buscar claim validado: %w", err)
			}
			campaigns[c.CampaignID] = append(campaigns[c.CampaignID], claimID)
			continue
		default:
			return fmt.Errorf("gravar claim: %w", err)
		}
		campaigns[c.CampaignID] = append(campaigns[c.CampaignID], claimID)

		for _, ev := range c.Evidences {
			var clusterID *string
			if ev.ClusterID != nil && *ev.ClusterID != "" {
				key := c.CampaignID + "|" + *ev.ClusterID
				id, ok := clusters[key]
				if !ok {
					id, err = recordOriginClusterTx(ctx, tx, c.CampaignID, *ev.ClusterID, ev.DocumentID)
					if err != nil {
						return err
					}
					clusters[key] = id
				}
				touched[id] = true
				clusterID = &id
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO evidence (
					claim_id, document_id, passage_id, cluster_id, stance, excerpt, page, confidence, is_official_doc, notes,
					vendor_match, model_match, firmware_match, feature_match, mode_match, prerequisites_match, source_span, validation_run_id
				)
				VALUES ($1, $2, $3, $4::uuid, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`,
				claimID, ev.DocumentID, ev.PassageID, clusterID, ev.Stance, ev.Excerpt, ev.Page, ev.Confidence, ev.IsOfficialDoc, ev.Notes,
				ev.VendorMatch, ev.ModelMatch, ev.FirmwareMatch, ev.FeatureMatch, ev.ModeMatch, ev.PrerequisitesMatch, ev.SourceSpan, ev.ValidationRunID)
			if err != nil {
				return fmt.Errorf("inserir evidence: %w", err)
			}
		}
	}

	for campaignID, keep := range campaigns {
		if _, err := tx.Exec(ctx, `
			DELETE FROM claims
			WHERE campaign_id=$1::uuid AND operational_validation='untested' AND id <> ALL($2::uuid[])`,
			campaignID, keep); err != nil {
			return fmt.Errorf("remover claims obsoletos: %w", err)
		}
	}
	if len(touched) > 0 {
		ids := make([]string, 0, len(touched))
		for id := range touched {
			ids = append(ids, id)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE origin_clusters oc SET duplicate_count = GREATEST(sub.n, 1)
			FROM (SELECT cluster_id, count(DISTINCT document_id) AS n FROM evidence
			      WHERE cluster_id = ANY($1::uuid[]) GROUP BY cluster_id) sub
			WHERE oc.id = sub.cluster_id`, ids); err != nil {
			return fmt.Errorf("atualizar contagem de réplicas: %w", err)
		}
	}
	return tx.Commit(ctx)
}

// recordOriginClusterTx cria (ou reaproveita) o cluster de origem de um
// fingerprint. A origem canônica é o primeiro documento visto na campanha.
func recordOriginClusterTx(ctx context.Context, tx pgx.Tx, campaignID, fingerprint, documentID string) (string, error) {
	var rootDomain, canonicalURL string
	err := tx.QueryRow(ctx, `
		SELECT domain, url FROM document_versions
		WHERE document_id=$1::uuid AND campaign_id=$2::uuid
		ORDER BY fetched_at ASC LIMIT 1`, documentID, campaignID).Scan(&rootDomain, &canonicalURL)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("buscar origem do documento: %w", err)
	}
	var id string
	if err := tx.QueryRow(ctx, `
		INSERT INTO origin_clusters (campaign_id, fingerprint, root_domain, canonical_url, duplicate_count)
		VALUES ($1::uuid, $2, $3, $4, 1)
		ON CONFLICT (campaign_id, fingerprint) DO UPDATE SET fingerprint = origin_clusters.fingerprint
		RETURNING id::text`, campaignID, fingerprint, rootDomain, canonicalURL).Scan(&id); err != nil {
		return "", fmt.Errorf("gravar origin cluster: %w", err)
	}
	return id, nil
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
