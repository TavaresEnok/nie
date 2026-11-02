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

type Result struct {
	URL     string `json:"url"`
	Title   string `json:"title,omitempty"`
	Snippet string `json:"snippet,omitempty"`
	Source  string `json:"source"`
}

type SearXNG struct {
	base   string
	client *http.Client
}

func NewSearXNG(base string) *SearXNG {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return nil
	}
	return &SearXNG{
		base:   base,
		client: &http.Client{Timeout: 20 * time.Second},
	}
}

func (s *SearXNG) Search(ctx context.Context, query string, limit int) ([]Result, error) {
	if s == nil || s.base == "" {
		return nil, fmt.Errorf("SearXNG não configurado")
	}
	if limit <= 0 || limit > 30 {
		limit = 10
	}
	endpoint := s.base + "/search?q=" + url.QueryEscape(query) + "&format=json&language=all&safesearch=1"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "NIEBot/0.1")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("SearXNG HTTP %d", resp.StatusCode)
	}
	var payload struct {
		Results []struct {
			URL     string `json:"url"`
			Title   string `json:"title"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("JSON do SearXNG: %w", err)
	}
	out := make([]Result, 0, limit)
	for _, item := range payload.Results {
		if item.URL == "" {
			continue
		}
		out = append(out, Result{URL: item.URL, Title: item.Title, Snippet: item.Content, Source: "searxng"})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}
