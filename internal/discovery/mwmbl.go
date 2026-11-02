package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Mwmbl integra o buscador livre e independente Mwmbl (https://mwmbl.org)
// conforme previsto no ADENDO MESTRE V2.1 para descoberta sem viés comercial.
type Mwmbl struct {
	base   string
	client *http.Client
}

func NewMwmbl(baseURL string) *Mwmbl {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = "https://mwmbl.org"
	}
	return &Mwmbl{
		base:   baseURL,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (m *Mwmbl) Search(ctx context.Context, query string, limit int) ([]Result, error) {
	if m == nil || m.base == "" {
		return nil, fmt.Errorf("mwmbl não configurado")
	}
	if limit <= 0 || limit > 30 {
		limit = 10
	}

	endpoint := fmt.Sprintf("%s/api/search?q=%s", m.base, url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "NIE-Crawler/2.1 (Network Intelligence Engine)")

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Mwmbl HTTP %d", resp.StatusCode)
	}

	var payload struct {
		Results []struct {
			URL     string `json:"url"`
			Title   string `json:"title"`
			Snippet string `json:"snippet"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("parse Mwmbl JSON: %w", err)
	}

	out := make([]Result, 0, limit)
	for _, item := range payload.Results {
		if item.URL == "" {
			continue
		}
		out = append(out, Result{
			URL:     item.URL,
			Title:   item.Title,
			Snippet: item.Snippet,
			Source:  "mwmbl",
		})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}
