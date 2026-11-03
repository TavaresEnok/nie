package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"nie/internal/domain"
)

type ExtractionRecord struct {
	Parser        string
	ParserVersion string
	Config        map[string]any
	InputSHA256   string
	Title         string
	Text          string
	Passages      []domain.Passage
}

// RecordExtraction publishes a new immutable extraction and changes only the
// document's active projection. Previous passages remain addressable by ID.
func (s *Store) RecordExtraction(ctx context.Context, documentID string, rec ExtractionRecord) (string, error) {
	if !validUUID(documentID) {
		return "", ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	id, err := recordExtractionTx(ctx, tx, documentID, rec)
	if err != nil {
		return "", err
	}
	return id, tx.Commit(ctx)
}

func recordExtractionTx(ctx context.Context, tx pgx.Tx, documentID string, rec ExtractionRecord) (string, error) {
	var original string
	if err := tx.QueryRow(ctx, `SELECT sha256 FROM documents WHERE id=$1::uuid FOR UPDATE`, documentID).Scan(&original); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	if rec.Parser == "" {
		rec.Parser = "unspecified"
	}
	if rec.ParserVersion == "" {
		rec.ParserVersion = "unknown"
	}
	if rec.InputSHA256 == "" {
		rec.InputSHA256 = original
	}
	if rec.Config == nil {
		rec.Config = map[string]any{}
	}
	if strings.TrimSpace(rec.Text) == "" || len(rec.Passages) == 0 {
		return "", fmt.Errorf("extração sem texto ou trechos")
	}
	// IDs from callers are never part of the new identity. Offsets refer to
	// UTF-8 bytes in this run's full text and are recorded only when exact.
	seen := map[int]bool{}
	cursor := 0
	clean := make([]domain.Passage, len(rec.Passages))
	for i, p := range rec.Passages {
		if p.Ordinal < 0 || seen[p.Ordinal] || strings.TrimSpace(p.Text) == "" {
			return "", fmt.Errorf("trecho inválido ou ordinal duplicado")
		}
		seen[p.Ordinal] = true
		clean[i] = domain.Passage{Ordinal: p.Ordinal, Page: p.Page, Text: p.Text}
		if offset := strings.Index(rec.Text[cursor:], p.Text); offset >= 0 {
			start := cursor + offset
			end := start + len(p.Text)
			clean[i].StartOffset = &start
			clean[i].EndOffset = &end
			cursor = end
		}
	}
	rec.Passages = clean
	identity, err := json.Marshal(rec)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(identity)
	config, err := json.Marshal(rec.Config)
	if err != nil {
		return "", err
	}
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO extraction_runs(document_id,parser,parser_version,config,input_sha256,identity_sha256,title,text)
        VALUES($1::uuid,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(document_id,identity_sha256) DO NOTHING RETURNING id::text`, documentID, rec.Parser, rec.ParserVersion, config, rec.InputSHA256, hex.EncodeToString(sum[:]), rec.Title, rec.Text).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		// A repeated download must not reactivate an old extraction over a
		// newer parser result. Explicit activation is a separate operation.
		if err := tx.QueryRow(ctx, `SELECT id::text FROM extraction_runs WHERE document_id=$1::uuid AND identity_sha256=$2`, documentID, hex.EncodeToString(sum[:])).Scan(&id); err != nil {
			return "", err
		}
		return id, nil
	}
	if err != nil {
		return "", err
	}
	for _, p := range clean {
		if _, err := tx.Exec(ctx, `INSERT INTO passages(document_id,extraction_run_id,ordinal,page,text,start_offset,end_offset) VALUES($1::uuid,$2::uuid,$3,$4,$5,$6,$7)`, documentID, id, p.Ordinal, p.Page, p.Text, p.StartOffset, p.EndOffset); err != nil {
			return "", err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE documents SET active_extraction_id=$2::uuid,text=$3,title=COALESCE(NULLIF($4,''),title),metadata=COALESCE(NULLIF(metadata,'null'::jsonb),'{}'::jsonb)||'{"extraction_status":"done"}'::jsonb WHERE id=$1::uuid`, documentID, id, rec.Text, rec.Title); err != nil {
		return "", err
	}
	return id, nil
}

func (s *Store) ListExtractions(ctx context.Context, documentID string) ([]domain.ExtractionRun, error) {
	if !validUUID(documentID) {
		return nil, ErrNotFound
	}
	rows, err := s.pool.Query(ctx, `SELECT id::text,document_id::text,parser,parser_version,config,input_sha256,identity_sha256,title,created_at FROM extraction_runs WHERE document_id=$1::uuid ORDER BY created_at,id`, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ExtractionRun{}
	for rows.Next() {
		var r domain.ExtractionRun
		var cfg []byte
		if err := rows.Scan(&r.ID, &r.DocumentID, &r.Parser, &r.ParserVersion, &cfg, &r.InputSHA256, &r.IdentitySHA256, &r.Title, &r.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(cfg, &r.Config); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) GetExtraction(ctx context.Context, documentID, runID string) (domain.ExtractionRun, error) {
	var r domain.ExtractionRun
	if !validUUID(documentID) || !validUUID(runID) {
		return r, ErrNotFound
	}
	var cfg []byte
	err := s.pool.QueryRow(ctx, `SELECT id::text,document_id::text,parser,parser_version,config,input_sha256,identity_sha256,title,text,created_at FROM extraction_runs WHERE document_id=$1::uuid AND id=$2::uuid`, documentID, runID).Scan(&r.ID, &r.DocumentID, &r.Parser, &r.ParserVersion, &cfg, &r.InputSHA256, &r.IdentitySHA256, &r.Title, &r.Text, &r.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, ErrNotFound
	}
	if err != nil {
		return r, err
	}
	if err := json.Unmarshal(cfg, &r.Config); err != nil {
		return r, err
	}
	r.Passages, err = s.extractionPassages(ctx, documentID, runID)
	return r, err
}

func (s *Store) extractionPassages(ctx context.Context, docID, runID string) ([]domain.Passage, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,document_id::text,extraction_run_id::text,ordinal,page,text,start_offset,end_offset FROM passages WHERE document_id=$1::uuid AND extraction_run_id=$2::uuid ORDER BY ordinal`, docID, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Passage{}
	for rows.Next() {
		var p domain.Passage
		if err := rows.Scan(&p.ID, &p.DocumentID, &p.ExtractionRunID, &p.Ordinal, &p.Page, &p.Text, &p.StartOffset, &p.EndOffset); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
