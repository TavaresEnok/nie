package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"nie/internal/domain"
)

type searchRow struct {
	result domain.SearchResult
	text   string
}

func (s *Store) Search(ctx context.Context, q, campaignID string, limit int) ([]domain.SearchResult, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return []domain.SearchResult{}, nil
	}
	if len(q) > 500 {
		q = q[:500]
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var campaign any
	if campaignID != "" {
		if !validUUID(campaignID) {
			return nil, ErrNotFound
		}
		campaign = campaignID
	}

	exact, err := s.searchExact(ctx, q, campaign, limit)
	if err != nil {
		return nil, err
	}
	seen := make(map[int64]struct{}, len(exact))
	exclude := make([]int64, 0, len(exact))
	for _, r := range exact {
		seen[r.result.PassageID] = struct{}{}
		exclude = append(exclude, r.result.PassageID)
	}

	fts, err := s.searchFTS(ctx, q, campaign, limit, exclude)
	if err != nil {
		return nil, err
	}
	for _, r := range fts {
		if _, ok := seen[r.result.PassageID]; ok {
			continue
		}
		seen[r.result.PassageID] = struct{}{}
		exclude = append(exclude, r.result.PassageID)
		exact = append(exact, r)
	}

	if len(exact) < limit && utf8.RuneCountInString(q) >= 3 {
		fuzzy, err := s.searchFuzzy(ctx, q, campaign, limit, exclude)
		if err != nil {
			return nil, err
		}
		for _, r := range fuzzy {
			if _, ok := seen[r.result.PassageID]; ok {
				continue
			}
			seen[r.result.PassageID] = struct{}{}
			exact = append(exact, r)
		}
	}

	for i := range exact {
		exact[i].result.Text = exact[i].text
		exact[i].result.Snippet = makeSnippet(exact[i].text, q)
	}
	if len(exact) > limit {
		exact = exact[:limit]
	}
	out := make([]domain.SearchResult, len(exact))
	for i := range exact {
		out[i] = exact[i].result
	}
	return out, nil
}

const searchSelect = `
SELECT p.id, p.document_id::text, p.extraction_run_id::text, d.title, d.mime, d.sha256, v.url, v.domain,
       p.page, p.ordinal, p.text, p.start_offset, p.end_offset, v.fetched_at, %s AS score
FROM passages p
JOIN documents d ON d.id = p.document_id
JOIN LATERAL (
    SELECT dv.url, dv.domain, dv.fetched_at
    FROM document_versions dv
    WHERE dv.document_id = d.id AND ($3::uuid IS NULL OR dv.campaign_id = $3::uuid)
    ORDER BY dv.fetched_at DESC LIMIT 1
) v ON true
WHERE p.extraction_run_id=d.active_extraction_id AND (%s)
ORDER BY %s
LIMIT $2`

func (s *Store) searchExact(ctx context.Context, q string, campaign any, limit int) ([]searchRow, error) {
	query := fmt.Sprintf(searchSelect,
		`(CASE WHEN strpos(p.text, $1) > 0 THEN 3.0 ELSE 2.0 END)::float8`,
		`strpos(lower(p.text), lower($1)) > 0`,
		`score DESC, p.id ASC`)
	return s.scanSearch(ctx, "exact", query, q, limit, campaign)
}

func (s *Store) searchFTS(ctx context.Context, q string, campaign any, limit int, exclude []int64) ([]searchRow, error) {
	query := fmt.Sprintf(searchSelect,
		`ts_rank_cd(p.search_vector, plainto_tsquery('simple', $1))::float8`,
		`p.search_vector @@ plainto_tsquery('simple', $1) AND p.id <> ALL($4::bigint[])`,
		`score DESC, p.id ASC`)
	if exclude == nil {
		exclude = []int64{}
	}
	return s.scanSearch(ctx, "fts", query, q, limit, campaign, exclude)
}

func (s *Store) searchFuzzy(ctx context.Context, q string, campaign any, limit int, exclude []int64) ([]searchRow, error) {
	query := fmt.Sprintf(searchSelect,
		`similarity(p.text, $1)::float8`,
		`similarity(p.text, $1) > 0.12 AND p.id <> ALL($4::bigint[])`,
		`score DESC, p.id ASC`)
	if exclude == nil {
		exclude = []int64{}
	}
	return s.scanSearch(ctx, "fuzzy", query, q, limit, campaign, exclude)
}

func (s *Store) scanSearch(ctx context.Context, match, query string, args ...any) ([]searchRow, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]searchRow, 0)
	for rows.Next() {
		var r searchRow
		var page *int
		var start, end *int
		var fetched time.Time
		if err := rows.Scan(&r.result.PassageID, &r.result.DocumentID, &r.result.ExtractionRunID, &r.result.Title, &r.result.MIME,
			&r.result.SHA256, &r.result.URL, &r.result.Domain, &page, &r.result.Ordinal,
			&r.text, &start, &end, &fetched, &r.result.Score); err != nil {
			return nil, err
		}
		r.result.Page = page
		r.result.StartOffset = start
		r.result.EndOffset = end
		r.result.FetchedAt = fetched
		r.result.Snippet = ""
		r.result.Match = match
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func makeSnippet(text, q string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	lowerText := strings.ToLower(text)
	lowerQ := strings.ToLower(q)
	idx := strings.Index(lowerText, lowerQ)
	if idx < 0 {
		idx = 0
	}
	start := idx - 180
	if start < 0 {
		start = 0
	}
	end := idx + len(q) + 260
	if end > len(text) {
		end = len(text)
	}
	for start > 0 && !utf8.RuneStart(text[start]) {
		start--
	}
	for end < len(text) && !utf8.RuneStart(text[end]) {
		end++
	}
	window := text[start:end]
	prefix := ""
	suffix := ""
	if start > 0 {
		prefix = "… "
	}
	if end < len(text) {
		suffix = " …"
	}

	if idx < 0 || q == "" {
		return prefix + html.EscapeString(window) + suffix
	}
	rel := strings.Index(strings.ToLower(window), lowerQ)
	if rel < 0 {
		return prefix + html.EscapeString(window) + suffix
	}
	before := window[:rel]
	match := window[rel : rel+len(q)]
	after := window[rel+len(q):]
	return prefix + html.EscapeString(before) + "<mark>" + html.EscapeString(match) + "</mark>" + html.EscapeString(after) + suffix
}

func (s *Store) GetDocument(ctx context.Context, id string) (domain.Document, error) {
	if !validUUID(id) {
		return domain.Document{}, ErrNotFound
	}
	var d domain.Document
	var meta []byte
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, sha256, mime, title, size, text, metadata, created_at, active_extraction_id::text
		FROM documents WHERE id=$1::uuid`, id).
		Scan(&d.ID, &d.SHA256, &d.MIME, &d.Title, &d.Size, &d.Text, &meta, &d.CreatedAt, &d.ActiveExtractionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Document{}, ErrNotFound
	}
	if err != nil {
		return domain.Document{}, err
	}
	_ = json.Unmarshal(meta, &d.Metadata)

	rows, err := s.pool.Query(ctx, `
		SELECT id::text, document_id::text, campaign_id::text, url, normalized_url, domain,
		       status_code, etag, last_modified, fetched_at
		FROM document_versions WHERE document_id=$1::uuid ORDER BY fetched_at DESC`, id)
	if err != nil {
		return domain.Document{}, err
	}
	for rows.Next() {
		var v domain.DocumentVersion
		if err := rows.Scan(&v.ID, &v.DocumentID, &v.CampaignID, &v.URL, &v.NormalizedURL, &v.Domain,
			&v.StatusCode, &v.ETag, &v.LastModified, &v.FetchedAt); err != nil {
			rows.Close()
			return domain.Document{}, err
		}
		d.Versions = append(d.Versions, v)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return domain.Document{}, err
	}

	if d.ActiveExtractionID != nil {
		d.Passages, err = s.extractionPassages(ctx, id, *d.ActiveExtractionID)
	}
	return d, err
}

func (s *Store) ListDocuments(ctx context.Context, limit int, campaignID string) ([]domain.DocumentSummary, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var campaign any
	if campaignID != "" {
		if !validUUID(campaignID) {
			return nil, ErrNotFound
		}
		campaign = campaignID
	}
	rows, err := s.pool.Query(ctx, `
		SELECT d.id::text, d.sha256, d.mime, d.title, d.size, v.url, v.domain,
		       v.campaign_id::text, v.fetched_at, d.created_at
		FROM documents d
		JOIN LATERAL (
			SELECT dv.url, dv.domain, dv.campaign_id, dv.fetched_at
			FROM document_versions dv
			WHERE dv.document_id=d.id AND ($2::uuid IS NULL OR dv.campaign_id=$2::uuid)
			ORDER BY dv.fetched_at DESC LIMIT 1
		) v ON true
		ORDER BY v.fetched_at DESC LIMIT $1`, limit, campaign)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.DocumentSummary, 0)
	for rows.Next() {
		var d domain.DocumentSummary
		if err := rows.Scan(&d.ID, &d.SHA256, &d.MIME, &d.Title, &d.Size, &d.URL, &d.Domain,
			&d.CampaignID, &d.FetchedAt, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
