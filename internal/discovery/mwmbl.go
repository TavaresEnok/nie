package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	if baseURL == "" || baseURL == "https://mwmbl.org" {
		baseURL = "https://api.mwmbl.org"
	}
	return &Mwmbl{
		base:   baseURL,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func parseMwmblText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		return str
	}
	var parts []struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(raw, &parts); err == nil {
		var sb strings.Builder
		for _, p := range parts {
			sb.WriteString(p.Value)
		}
		return sb.String()
	}
	return ""
}

type mwmblItem struct {
	URL     string          `json:"url"`
	Title   json.RawMessage `json:"title"`
	Extract json.RawMessage `json:"extract"`
	Snippet string          `json:"snippet"`
}

func (m *Mwmbl) Search(ctx context.Context, query string, limit int) ([]Result, error) {
	if m == nil || m.base == "" {
		return nil, fmt.Errorf("mwmbl não configurado")
	}
	if limit <= 0 || limit > 30 {
		limit = 10
	}

	var endpoint string
	if strings.Contains(m.base, "mwmbl.org") {
		endpoint = fmt.Sprintf("https://api.mwmbl.org/api/v1/search/?s=%s", url.QueryEscape(query))
	} else {
		endpoint = fmt.Sprintf("%s/api/search?q=%s", m.base, url.QueryEscape(query))
	}

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

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ler corpo Mwmbl: %w", err)
	}
	bodyStr := strings.TrimSpace(string(bodyBytes))
	if len(bodyStr) == 0 {
		return []Result{}, nil
	}

	out := make([]Result, 0, limit)

	// Resposta array nativa da api.mwmbl.org: [ {"url": "...", "title": [...]}, ... ]
	if strings.HasPrefix(bodyStr, "[") {
		var items []mwmblItem
		if err := json.Unmarshal(bodyBytes, &items); err != nil {
			return nil, fmt.Errorf("parse Mwmbl JSON array: %w", err)
		}
		for _, item := range items {
			if item.URL == "" {
				continue
			}
			snip := parseMwmblText(item.Extract)
			if snip == "" {
				snip = item.Snippet
			}
			out = append(out, Result{
				URL:     item.URL,
				Title:   parseMwmblText(item.Title),
				Snippet: snip,
				Source:  "mwmbl",
			})
			if len(out) >= limit {
				break
			}
		}
		return out, nil
	}

	// Resposta encapsulada em objeto {...} (mocks de teste ou gateways)
	var rawObj struct {
		Results []struct {
			URL     string `json:"url"`
			Title   string `json:"title"`
			Snippet string `json:"snippet"`
		} `json:"results"`
		Value []mwmblItem `json:"value"`
	}
	if err := json.Unmarshal(bodyBytes, &rawObj); err != nil {
		return nil, fmt.Errorf("parse Mwmbl JSON object: %w", err)
	}

	for _, item := range rawObj.Value {
		if item.URL == "" {
			continue
		}
		snip := parseMwmblText(item.Extract)
		if snip == "" {
			snip = item.Snippet
		}
		out = append(out, Result{
			URL:     item.URL,
			Title:   parseMwmblText(item.Title),
			Snippet: snip,
			Source:  "mwmbl",
		})
		if len(out) >= limit {
			break
		}
	}

	for _, item := range rawObj.Results {
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
