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

// SoftwareHeritage integra a API de acervo histórico de código aberto da Software Heritage.
type SoftwareHeritage struct {
	base   string
	client *http.Client
}

func NewSoftwareHeritage(base ...string) *SoftwareHeritage {
	endpoint := "https://archive.softwareheritage.org/api/1"
	if len(base) > 0 && strings.TrimSpace(base[0]) != "" {
		endpoint = strings.TrimRight(strings.TrimSpace(base[0]), "/")
	}
	return &SoftwareHeritage{
		base:   endpoint,
		client: &http.Client{Timeout: 20 * time.Second},
	}
}

// SearchOrigins pesquisa origens de código arquivadas (ex: FRRouting, BIRD, OpenBGPd, Quagga).
func (s *SoftwareHeritage) SearchOrigins(ctx context.Context, query string, limit int) ([]Result, error) {
	if s == nil || s.base == "" {
		return nil, fmt.Errorf("software heritage não configurado")
	}
	if limit <= 0 || limit > 30 {
		limit = 10
	}

	endpoint := fmt.Sprintf("%s/origin/search/%s/?limit=%d", s.base, url.PathEscape(query), limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "NIE-Crawler/2.1 (Network Intelligence Engine)")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("SoftwareHeritage HTTP %d", resp.StatusCode)
	}

	var origins []struct {
		URL  string `json:"url"`
		Type string `json:"type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&origins); err != nil {
		return nil, fmt.Errorf("parse SoftwareHeritage JSON: %w", err)
	}

	out := make([]Result, 0, len(origins))
	for _, o := range origins {
		if o.URL == "" {
			continue
		}
		out = append(out, Result{
			URL:     o.URL,
			Title:   fmt.Sprintf("[%s] %s", strings.ToUpper(o.Type), o.URL),
			Snippet: fmt.Sprintf("Acervo histórico Software Heritage origin: %s", o.URL),
			Source:  "software_heritage",
		})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}
