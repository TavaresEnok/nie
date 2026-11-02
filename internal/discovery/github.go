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

type GitHub struct {
	token  string
	client *http.Client
}

type githubRepo struct {
	FullName      string `json:"full_name"`
	HTMLURL       string `json:"html_url"`
	DefaultBranch string `json:"default_branch"`
	Description   string `json:"description"`
}

func NewGitHub(token string, enabled bool) *GitHub {
	if !enabled {
		return nil
	}
	return &GitHub{
		token:  token,
		client: &http.Client{Timeout: 25 * time.Second},
	}
}

func (g *GitHub) SearchReadmes(ctx context.Context, query string, limit int) ([]Result, error) {
	if g == nil {
		return nil, fmt.Errorf("GitHub não configurado")
	}
	if limit <= 0 || limit > 10 {
		limit = 5
	}
	searchURL := "https://api.github.com/search/repositories?q=" + url.QueryEscape(query) +
		"&sort=stars&order=desc&per_page=" + fmt.Sprint(limit)
	var repos struct {
		Items []githubRepo `json:"items"`
	}
	if err := g.getJSON(ctx, searchURL, &repos); err != nil {
		return nil, err
	}
	out := make([]Result, 0, len(repos.Items))
	for _, repo := range repos.Items {
		var readme struct {
			DownloadURL string `json:"download_url"`
			HTMLURL     string `json:"html_url"`
		}
		endpoint := "https://api.github.com/repos/" + repo.FullName + "/readme"
		if err := g.getJSON(ctx, endpoint, &readme); err != nil {
			continue
		}
		target := readme.DownloadURL
		if target == "" {
			target = readme.HTMLURL
		}
		if target == "" {
			continue
		}
		out = append(out, Result{
			URL:     target,
			Title:   repo.FullName,
			Snippet: repo.Description,
			Source:  "github",
		})
	}
	return out, nil
}

func (g *GitHub) getJSON(ctx context.Context, endpoint string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "NIEBot/0.1")
	if g.token != "" {
		req.Header.Set("Authorization", "Bearer "+g.token)
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GitHub HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}

// SanitizeQuery removes GitHub qualifiers that could break the search endpoint.
func SanitizeQuery(q string) string {
	q = strings.TrimSpace(q)
	q = strings.ReplaceAll(q, "\x00", " ")
	if len(q) > 240 {
		q = q[:240]
	}
	return q
}
