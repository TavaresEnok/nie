package api

import (
	"context"
	"nie/internal/domain"
	"nie/internal/research"
	"nie/internal/store"
)

func retrievalQueries(question string) []string { return research.RetrievalQueries(question) }
func retrievePassages(ctx context.Context, st *store.Store, question, campaignID string, limit int) ([]domain.SearchResult, error) {
	return research.RetrievePassages(ctx, st, question, campaignID, limit)
}
