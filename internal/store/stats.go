package store

import (
	"context"

	"nie/internal/domain"
)

func (s *Store) Stats(ctx context.Context) (domain.Stats, error) {
	var st domain.Stats
	if err := s.pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*)::int FROM campaigns),
			(SELECT count(*)::int FROM documents),
			(SELECT count(*)::int FROM passages p JOIN documents d ON d.id=p.document_id AND d.active_extraction_id=p.extraction_run_id),
			COALESCE((SELECT sum(x.size) FROM (SELECT DISTINCT id, size FROM documents) x), 0)::bigint,
			(SELECT count(DISTINCT domain)::int FROM document_versions)
	`).Scan(&st.Campaigns, &st.Documents, &st.Passages, &st.Bytes, &st.Domains); err != nil {
		return st, err
	}
	st.Jobs = map[string]int{}
	rows, err := s.pool.Query(ctx, `SELECT status, count(*)::int FROM jobs GROUP BY status`)
	if err != nil {
		return st, err
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var n int
		if err := rows.Scan(&status, &n); err != nil {
			return st, err
		}
		st.Jobs[status] = n
	}
	return st, rows.Err()
}
