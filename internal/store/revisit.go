package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"nie/internal/library"
	"nie/internal/urlutil"
)

// SaveResult é o efeito de gravar um original já identificado por hash.
// Changed indica que a mesma URL da campanha apontava para outros bytes.
type SaveResult struct {
	ID          string
	Created     bool
	Changed     bool
	PreviousSHA string
}

type Observation struct {
	DocumentID   string
	SHA256       string
	ETag         string
	LastModified string
}

type DocumentChange struct {
	ID         int64     `json:"id"`
	CampaignID string    `json:"campaign_id"`
	URL        string    `json:"url"`
	OldSHA256  string    `json:"old_sha256"`
	NewSHA256  string    `json:"new_sha256"`
	Summary    string    `json:"summary"`
	CreatedAt  time.Time `json:"created_at"`
}

// SaveDocument grava o documento e a observação da URL na mesma transação.
func (s *Store) SaveDocument(ctx context.Context, rec DocumentRecord) (SaveResult, error) {
	if rec.Metadata == nil {
		rec.Metadata = map[string]any{}
	}
	meta, err := json.Marshal(rec.Metadata)
	if err != nil {
		return SaveResult{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return SaveResult{}, err
	}
	defer tx.Rollback(ctx)

	var docID string
	created := false
	err = tx.QueryRow(ctx, `
		INSERT INTO documents(sha256, mime, title, size, text, metadata)
		VALUES($1, $2, $3, $4, $5, $6)
		ON CONFLICT (sha256) DO NOTHING
		RETURNING id::text`,
		rec.SHA256, rec.MIME, rec.Title, rec.Size, rec.Text, meta,
	).Scan(&docID)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := tx.QueryRow(ctx, `SELECT id::text FROM documents WHERE sha256=$1`, rec.SHA256).Scan(&docID); err != nil {
			return SaveResult{}, err
		}
	} else if err != nil {
		return SaveResult{}, fmt.Errorf("inserir documento: %w", err)
	} else {
		created = true
	}

	if len(rec.Passages) > 0 && strings.TrimSpace(rec.Text) != "" {
		if _, err := recordExtractionTx(ctx, tx, docID, ExtractionRecord{Parser: rec.Parser, ParserVersion: rec.ParserVersion, Config: rec.ParserConfig, InputSHA256: rec.ExtractionInputSHA256, Title: rec.Title, Text: rec.Text, Passages: rec.Passages}); err != nil {
			return SaveResult{}, err
		}
		if _, err := tx.Exec(ctx, `UPDATE documents SET metadata=COALESCE(NULLIF(metadata,'null'::jsonb),'{}'::jsonb)||$2::jsonb||'{"extraction_status":"done"}'::jsonb WHERE id=$1::uuid`, docID, meta); err != nil {
			return SaveResult{}, err
		}
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO document_versions(document_id, campaign_id, url, normalized_url, domain, status_code, etag, last_modified)
		VALUES($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (campaign_id, document_id, normalized_url)
		DO UPDATE SET fetched_at=now(), status_code=EXCLUDED.status_code,
		              etag=EXCLUDED.etag, last_modified=EXCLUDED.last_modified`,
		docID, rec.CampaignID, rec.URL, rec.NormalizedURL, rec.Domain, rec.StatusCode, rec.ETag, rec.LastModified)
	if err != nil {
		return SaveResult{}, fmt.Errorf("inserir versão: %w", err)
	}
	previous, changed, err := observeTx(ctx, tx, rec.CampaignID, rec.NormalizedURL, docID, rec.SHA256, rec.ETag, rec.LastModified)
	if err != nil {
		return SaveResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return SaveResult{}, err
	}
	return SaveResult{ID: docID, Created: created, Changed: changed, PreviousSHA: previous}, nil
}

func observeTx(ctx context.Context, tx pgx.Tx, campaignID, normalizedURL, docID, sha, etag, lastModified string) (string, bool, error) {
	var previous string
	err := tx.QueryRow(ctx, `SELECT sha256 FROM source_observations WHERE campaign_id=$1::uuid AND normalized_url=$2 FOR UPDATE`, campaignID, normalizedURL).Scan(&previous)
	if errors.Is(err, pgx.ErrNoRows) {
		_, err = tx.Exec(ctx, `INSERT INTO source_observations(campaign_id,normalized_url,document_id,sha256,etag,last_modified)
			VALUES($1::uuid,$2,$3::uuid,$4,$5,$6)`, campaignID, normalizedURL, docID, sha, etag, lastModified)
		return "", false, err
	}
	if err != nil {
		return "", false, err
	}
	change := library.Compare(previous, sha)
	changed := change == library.ChangeUpdated
	query := `UPDATE source_observations SET document_id=$3::uuid, sha256=$4, etag=$5, last_modified=$6, last_checked=now() WHERE campaign_id=$1::uuid AND normalized_url=$2`
	if changed {
		query = `UPDATE source_observations SET document_id=$3::uuid, sha256=$4, etag=$5, last_modified=$6, last_checked=now(), last_changed=now() WHERE campaign_id=$1::uuid AND normalized_url=$2`
	}
	if _, err := tx.Exec(ctx, query, campaignID, normalizedURL, docID, sha, etag, lastModified); err != nil {
		return previous, false, err
	}
	if !changed {
		return previous, false, nil
	}
	summary := "o hash da URL mudou; o original anterior permanece no acervo"
	var oldText, newText string
	if err := tx.QueryRow(ctx, `SELECT COALESCE(text,'') FROM documents WHERE sha256=$1`, previous).Scan(&oldText); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return previous, false, err
	}
	if err := tx.QueryRow(ctx, `SELECT COALESCE(text,'') FROM documents WHERE id=$1::uuid`, docID).Scan(&newText); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return previous, false, err
	}
	if diff := library.DiffSummary(oldText, newText, 6); diff != "" {
		if len(diff) > 2000 {
			diff = diff[:2000]
		}
		summary = summary + "\n" + diff
	}
	if _, err := tx.Exec(ctx, `INSERT INTO document_diffs(document_id,url,old_sha256,new_sha256,summary,campaign_id) VALUES($1::uuid,$2,$3,$4,$5,$6::uuid)`, docID, normalizedURL, previous, sha, summary, campaignID); err != nil {
		return previous, false, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO alerts(campaign_id,title,message,severity) VALUES($1::uuid,$2,$3,'warning')`, campaignID, "Documento alterado", fmt.Sprintf("%s deixou de corresponder ao hash anterior.", normalizedURL)); err != nil {
		return previous, false, err
	}
	return previous, true, nil
}

// LatestObservation devolve a última captura conhecida da URL na campanha.
func (s *Store) LatestObservation(ctx context.Context, campaignID, normalizedURL string) (Observation, bool, error) {
	var obs Observation
	err := s.pool.QueryRow(ctx, `SELECT COALESCE(document_id::text,''), sha256, etag, last_modified FROM source_observations WHERE campaign_id=$1::uuid AND normalized_url=$2`, campaignID, normalizedURL).Scan(&obs.DocumentID, &obs.SHA256, &obs.ETag, &obs.LastModified)
	if errors.Is(err, pgx.ErrNoRows) {
		return Observation{}, false, nil
	}
	if err != nil {
		return Observation{}, false, err
	}
	return obs, true, nil
}

// TouchObservation registra uma revisita sem bytes novos, como uma resposta 304.
func (s *Store) TouchObservation(ctx context.Context, campaignID, normalizedURL, etag, lastModified string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE source_observations SET last_checked=now(), etag=CASE WHEN $3<>'' THEN $3 ELSE etag END, last_modified=CASE WHEN $4<>'' THEN $4 ELSE last_modified END WHERE campaign_id=$1::uuid AND normalized_url=$2`, campaignID, normalizedURL, etag, lastModified)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ReevaluateAfterChange reabre uma campanha concluída e pede outra avaliação
// quando ainda há rodadas automáticas. Campanha pausada não é retomada.
func (s *Store) ReevaluateAfterChange(ctx context.Context, campaignID string) error {
	if s.researchRounds <= 0 {
		return nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var status string
	if err := tx.QueryRow(ctx, `SELECT status FROM campaigns WHERE id=$1::uuid FOR UPDATE`, campaignID).Scan(&status); err != nil {
		return err
	}
	if status == "paused" {
		return tx.Commit(ctx)
	}
	if status == "completed" || status == "failed" {
		if _, err := tx.Exec(ctx, `UPDATE campaigns SET status='running', updated_at=now(), finished_at=NULL WHERE id=$1::uuid`, campaignID); err != nil {
			return err
		}
	}
	var active int
	if err := tx.QueryRow(ctx, `SELECT count(*)::int FROM jobs WHERE campaign_id=$1::uuid AND kind='investigate' AND status IN ('pending','running')`, campaignID).Scan(&active); err != nil {
		return err
	}
	if active == 0 {
		var rounds int
		if err := tx.QueryRow(ctx, `SELECT count(*)::int FROM jobs WHERE campaign_id=$1::uuid AND kind='investigate'`, campaignID).Scan(&rounds); err != nil {
			return err
		}
		if rounds < s.researchRounds {
			if _, err := tx.Exec(ctx, `INSERT INTO jobs(campaign_id,kind,priority,reason,payload) VALUES($1::uuid,'investigate',200,'document_changed',jsonb_build_object('round',$2::int))`, campaignID, rounds+1); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

// EnqueueRevisit agenda uma nova leitura da URL. Uma revisita já concluída
// volta para a fila; uma revisita em andamento não é duplicada.
func (s *Store) EnqueueRevisit(ctx context.Context, campaignID, rawURL string) (bool, error) {
	norm, err := urlutil.Normalize(rawURL)
	if err != nil {
		return false, nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	var status string
	if err := tx.QueryRow(ctx, `SELECT status FROM campaigns WHERE id=$1::uuid FOR UPDATE`, campaignID).Scan(&status); err != nil {
		return false, err
	}
	if status == "paused" {
		return false, tx.Commit(ctx)
	}
	tag, err := tx.Exec(ctx, `
		INSERT INTO jobs(campaign_id,kind,url,url_hash,domain,priority,reason,scheduled_at)
		VALUES($1::uuid,'revisit',$2,$3,$4,80,'revisit',now())
		ON CONFLICT (campaign_id, kind, url_hash) WHERE url_hash IS NOT NULL
		DO UPDATE SET status='pending', scheduled_at=now(), attempts=0, lease_until=NULL, last_error=NULL, updated_at=now()
		WHERE jobs.status IN ('done','dead')`,
		campaignID, norm, urlutil.Hash(norm), urlutil.Domain(norm))
	if err != nil {
		return false, err
	}
	if status == "completed" || status == "failed" {
		if _, err := tx.Exec(ctx, `UPDATE campaigns SET status='running', updated_at=now(), finished_at=NULL WHERE id=$1::uuid`, campaignID); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *Store) ListDueObservations(ctx context.Context, olderThan time.Duration) ([]struct{ CampaignID, URL string }, error) {
	if olderThan <= 0 {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT o.campaign_id::text, o.normalized_url
		FROM source_observations o
		JOIN campaigns c ON c.id = o.campaign_id
		WHERE c.status <> 'paused' AND o.last_checked <= now() - $1::interval`, fmt.Sprintf("%d seconds", int(olderThan.Seconds())))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []struct{ CampaignID, URL string }
	for rows.Next() {
		var item struct{ CampaignID, URL string }
		if err := rows.Scan(&item.CampaignID, &item.URL); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) ListDocumentChanges(ctx context.Context, campaignID string) ([]DocumentChange, error) {
	if !validUUID(campaignID) {
		return nil, ErrNotFound
	}
	rows, err := s.pool.Query(ctx, `SELECT id, campaign_id::text, url, old_sha256, new_sha256, summary, created_at FROM document_diffs WHERE campaign_id=$1::uuid ORDER BY id`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DocumentChange{}
	for rows.Next() {
		var change DocumentChange
		if err := rows.Scan(&change.ID, &change.CampaignID, &change.URL, &change.OldSHA256, &change.NewSHA256, &change.Summary, &change.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, change)
	}
	return out, rows.Err()
}
