package api

import (
	"context"
	"strings"

	"nie/internal/domain"
	"nie/internal/evidence"
	"nie/internal/store"
)

// retrievalQueries decompõe uma pergunta longa em consultas curtas. A busca
// lexical exige todos os termos, então a pergunta inteira quase nunca recupera
// um trecho específico; as variantes combinam fabricante, modelo, firmware e
// recurso (e cada recurso/modelo isolado). A pergunta original vem primeiro.
func retrievalQueries(question string) []string {
	req := evidence.ParseRequirements(question)
	seen := make(map[string]bool)
	var out []string
	add := func(parts ...string) {
		q := strings.Join(strings.Fields(strings.Join(parts, " ")), " ")
		if q == "" || seen[strings.ToLower(q)] {
			return
		}
		seen[strings.ToLower(q)] = true
		out = append(out, q)
	}
	add(question)

	first := func(xs []string) string {
		if len(xs) > 0 {
			return xs[0]
		}
		return ""
	}
	vendor, model, fw := first(req.Vendors), first(req.Models), first(req.Firmwares)
	add(vendor, model, fw)
	for _, f := range req.Features {
		add(model, f)
		add(vendor, f)
		add(f, fw)
		add(f)
	}
	add(model)
	return out
}

// retrievePassages busca com a pergunta e suas variantes e funde os resultados
// sem repetir trechos (a ordem de aparição preserva a prioridade da consulta).
func retrievePassages(ctx context.Context, st *store.Store, question, campaignID string, limit int) ([]domain.SearchResult, error) {
	seen := make(map[int64]bool)
	var merged []domain.SearchResult
	for _, q := range retrievalQueries(question) {
		res, err := st.Search(ctx, q, campaignID, limit)
		if err != nil {
			return nil, err
		}
		for _, r := range res {
			if !seen[r.PassageID] {
				seen[r.PassageID] = true
				merged = append(merged, r)
			}
		}
		if len(merged) >= limit {
			break
		}
	}
	if len(merged) > limit {
		merged = merged[:limit]
	}
	return merged, nil
}
