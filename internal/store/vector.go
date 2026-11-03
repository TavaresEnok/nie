package store

import (
	"context"
	"fmt"
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
