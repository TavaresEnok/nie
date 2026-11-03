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

// CommonCrawl consulta o índice CDX público. A busca é por URL ou domínio,
// não por texto livre de todo o acervo.
type CommonCrawl struct {
	collinfo string
	client   *http.Client
}

func NewCommonCrawl(collinfo string, enabled bool) *CommonCrawl {
	if !enabled {
		return nil
	}
	collinfo = strings.TrimSpace(collinfo)
	if collinfo == "" {
		collinfo = "https://index.commoncrawl.org/collinfo.json"
	}
	return &CommonCrawl{collinfo: collinfo, client: &http.Client{Timeout: 25 * time.Second}}
}

func (c *CommonCrawl) Search(ctx context.Context, query string, limit int) ([]Result, error) {
	if c == nil || c.collinfo == "" {
		return nil, fmt.Errorf("Common Crawl não configurado")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("consulta Common Crawl vazia")
	}
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	pattern := query
	if !strings.Contains(pattern, "/") && !strings.Contains(pattern, "*") {
		pattern = "*." + pattern + "/*"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.collinfo, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "NIEBot/0.1")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Common Crawl collinfo HTTP %d", resp.StatusCode)
	}
	var indexes []struct {
		CDX string `json:"cdx-api"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&indexes); err != nil {
		return nil, err
	}
	if len(indexes) == 0 || indexes[0].CDX == "" {
		return nil, fmt.Errorf("Common Crawl sem índice CDX")
	}
	cdx := indexes[0].CDX + "?url=" + url.QueryEscape(pattern) + "&output=json&fl=url,timestamp,status&limit=" + fmt.Sprint(limit)
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, cdx, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "NIEBot/0.1")
	resp2, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp2.Body.Close()
	if resp2.StatusCode < 200 || resp2.StatusCode >= 300 {
		return nil, fmt.Errorf("Common Crawl CDX HTTP %d", resp2.StatusCode)
	}
	var rows []struct {
		URL       string `json:"url"`
		Timestamp string `json:"timestamp"`
		Status    string `json:"status"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&rows); err != nil {
		return nil, err
	}
	out := make([]Result, 0, len(rows))
	for _, row := range rows {
		if row.URL == "" {
			continue
		}
		out = append(out, Result{URL: row.URL, Title: row.Timestamp, Snippet: "status " + row.Status, Source: "commoncrawl"})
	}
	return out, nil
}
