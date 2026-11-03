package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// OpenSearch consulta um índice externo. O PostgreSQL continua sendo o catálogo.
type OpenSearch struct {
	base   string
	index  string
	user   string
	secret string
	client *http.Client
}

type ExternalHit struct {
	ID     string  `json:"id"`
	Score  float64 `json:"score"`
	Source any     `json:"source"`
}

func NewOpenSearch(base, index, user, secret string) *OpenSearch {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return nil
	}
	if strings.TrimSpace(index) == "" {
		index = "nie"
	}
	return &OpenSearch{base: base, index: index, user: user, secret: secret, client: &http.Client{Timeout: 20 * time.Second}}
}

func (o *OpenSearch) Search(ctx context.Context, query string, limit int) ([]ExternalHit, error) {
	if o == nil || o.base == "" {
		return nil, fmt.Errorf("OpenSearch não configurado")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("consulta OpenSearch vazia")
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	body, _ := json.Marshal(map[string]any{
		"size": limit,
		"query": map[string]any{
			"simple_query_string": map[string]any{"query": query, "default_operator": "and"},
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.base+"/"+urlPath(o.index)+"/_search", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if o.user != "" {
		req.SetBasicAuth(o.user, o.secret)
	}
	resp, err := o.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("OpenSearch HTTP %d", resp.StatusCode)
	}
	var payload struct {
		Hits struct {
			Hits []struct {
				ID     string  `json:"_id"`
				Score  float64 `json:"_score"`
				Source any     `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	out := make([]ExternalHit, 0, len(payload.Hits.Hits))
	for _, hit := range payload.Hits.Hits {
		out = append(out, ExternalHit{ID: hit.ID, Score: hit.Score, Source: hit.Source})
	}
	return out, nil
}

func urlPath(index string) string {
	return strings.Trim(strings.ReplaceAll(index, " ", ""), "/")
}
