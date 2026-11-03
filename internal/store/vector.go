package store

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"nie/internal/domain"
)

// EnsureVector cria a extensão e a tabela de embeddings quando o operador
// ligou NIE_PGVECTOR. Sem o pacote vector no PostgreSQL, a chamada falha e o
// restante do NIE continua no catálogo lexical. Nenhum vetor é inventado aqui.
func (s *Store) EnsureVector(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS vector`); err != nil {
		return fmt.Errorf("pgvector: %w", err)
	}
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS passage_embeddings (
			passage_id bigint PRIMARY KEY REFERENCES passages(id) ON DELETE CASCADE,
			model text NOT NULL,
			embedding vector(768) NOT NULL,
			created_at timestamptz NOT NULL DEFAULT now()
		)`)
	return err
}

func vectorLiteral(vec []float32) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, n := range vec {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(n), 'f', -1, 32))
	}
	b.WriteByte(']')
	return b.String()
}

// UpsertEmbedding grava o vetor do trecho. O catálogo lexical não depende desta linha.
func (s *Store) UpsertEmbedding(ctx context.Context, passageID int64, model string, vec []float32) error {
	if len(vec) != 768 {
		return fmt.Errorf("embedding precisa ter 768 dimensões, recebeu %d", len(vec))
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO passage_embeddings(passage_id, model, embedding)
		VALUES ($1, $2, $3::vector)
		ON CONFLICT (passage_id) DO UPDATE
		SET model = EXCLUDED.model, embedding = EXCLUDED.embedding, created_at = now()`,
		passageID, model, vectorLiteral(vec))
	return err
}

// ActivePassages devolve os trechos da extração ativa de um documento.
func (s *Store) ActivePassages(ctx context.Context, documentID string) ([]domain.Passage, error) {
	if !validUUID(documentID) {
		return nil, ErrNotFound
	}
	rows, err := s.pool.Query(ctx, `
		SELECT p.id, p.document_id::text, p.ordinal, p.text
		FROM passages p
		JOIN documents d ON d.id = p.document_id AND d.active_extraction_id = p.extraction_run_id
		WHERE d.id = $1::uuid
		ORDER BY p.ordinal
		LIMIT 20`, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Passage
	for rows.Next() {
		var p domain.Passage
		if err := rows.Scan(&p.ID, &p.DocumentID, &p.Ordinal, &p.Text); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func scanPassageRows(rows pgx.Rows, match string) ([]domain.SearchResult, error) {
	defer rows.Close()
	var out []domain.SearchResult
	for rows.Next() {
		var r domain.SearchResult
		var page *int
		var start, end *int
		var text string
		var fetched time.Time
		if err := rows.Scan(&r.PassageID, &r.DocumentID, &r.ExtractionRunID, &r.Title, &r.MIME, &r.SHA256, &r.URL, &r.Domain,
			&page, &r.Ordinal, &text, &start, &end, &fetched, &r.Score); err != nil {
			return nil, err
		}
		r.Page = page
		r.StartOffset = start
		r.EndOffset = end
		r.FetchedAt = fetched
		r.Text = text
		r.Match = match
		out = append(out, r)
	}
	return out, rows.Err()
}

// SearchSimilar ordena trechos pela distância de cosseno. Falha se o pgvector não existir.
func (s *Store) SearchSimilar(ctx context.Context, campaignID string, vec []float32, limit int) ([]domain.SearchResult, error) {
	if len(vec) != 768 {
		return nil, fmt.Errorf("embedding precisa ter 768 dimensões, recebeu %d", len(vec))
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	var campaign any
	if campaignID != "" {
		if !validUUID(campaignID) {
			return nil, ErrNotFound
		}
		campaign = campaignID
	}
	rows, err := s.pool.Query(ctx, `
		SELECT p.id, p.document_id::text, p.extraction_run_id::text, d.title, d.mime, d.sha256, v.url, v.domain,
		       p.page, p.ordinal, p.text, p.start_offset, p.end_offset, v.fetched_at,
		       (1 - (e.embedding <=> $1::vector))::float8 AS score
		FROM passage_embeddings e
		JOIN passages p ON p.id = e.passage_id
		JOIN documents d ON d.id = p.document_id AND d.active_extraction_id = p.extraction_run_id
		JOIN LATERAL (
			SELECT dv.url, dv.domain, dv.fetched_at
			FROM document_versions dv
			WHERE dv.document_id = d.id AND ($3::uuid IS NULL OR dv.campaign_id = $3::uuid)
			ORDER BY dv.fetched_at DESC LIMIT 1
		) v ON true
		ORDER BY e.embedding <=> $1::vector
		LIMIT $2`, vectorLiteral(vec), limit, campaign)
	if err != nil {
		return nil, err
	}
	return scanPassageRows(rows, "semantic")
}

// NeighborPassages lê trechos dos documentos ligados por uma aresta a partir das URLs já encontradas.
func (s *Store) NeighborPassages(ctx context.Context, campaignID string, fromURLs []string, limit int) ([]domain.SearchResult, error) {
	if !validUUID(campaignID) || len(fromURLs) == 0 {
		return nil, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
		SELECT p.id, p.document_id::text, p.extraction_run_id::text, d.title, d.mime, d.sha256, v.url, v.domain,
		       p.page, p.ordinal, p.text, p.start_offset, p.end_offset, v.fetched_at, 0::float8 AS score
		FROM discovery_edges e
		JOIN document_versions dst ON dst.campaign_id = e.campaign_id AND dst.normalized_url = e.to_url
		JOIN documents d ON d.id = dst.document_id
		JOIN passages p ON p.document_id = d.id AND p.extraction_run_id = d.active_extraction_id
		JOIN LATERAL (
			SELECT dv.url, dv.domain, dv.fetched_at
			FROM document_versions dv
			WHERE dv.document_id = d.id AND dv.campaign_id = e.campaign_id
			ORDER BY dv.fetched_at DESC LIMIT 1
		) v ON true
		WHERE e.campaign_id = $1::uuid AND e.from_url = ANY($2::text[])
		ORDER BY p.id
		LIMIT $3`, campaignID, fromURLs, limit)
	if err != nil {
		return nil, err
	}
	return scanPassageRows(rows, "neighbor")
}
