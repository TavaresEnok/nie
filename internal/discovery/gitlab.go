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

// GitLab consulta a API pública de projetos e devolve a URL bruta do README.
// Não baixa o repositório inteiro e não pesquisa o conteúdo de todos os arquivos.
type GitLab struct {
	base   string
	token  string
	client *http.Client
}

type gitlabProject struct {
	ID            int    `json:"id"`
	Path          string `json:"path_with_namespace"`
	WebURL        string `json:"web_url"`
	Description   string `json:"description"`
	DefaultBranch string `json:"default_branch"`
}

func NewGitLab(base, token string, enabled bool) *GitLab {
	if !enabled {
		return nil
	}
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		base = "https://gitlab.com"
	}
	return &GitLab{
		base:   base,
		token:  token,
		client: &http.Client{Timeout: 25 * time.Second},
	}
}

func (g *GitLab) SearchReadmes(ctx context.Context, query string, limit int) ([]Result, error) {
	if g == nil {
		return nil, fmt.Errorf("GitLab não configurado")
	}
	if limit <= 0 || limit > 10 {
		limit = 5
	}
	query = SanitizeQuery(query)
	if query == "" {
		return nil, fmt.Errorf("consulta GitLab vazia")
	}
	endpoint := g.base + "/api/v4/projects?simple=true&order_by=last_activity_at&search=" + url.QueryEscape(query) + "&per_page=" + fmt.Sprint(limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "NIEBot/0.1")
	if g.token != "" {
		req.Header.Set("PRIVATE-TOKEN", g.token)
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GitLab HTTP %d", resp.StatusCode)
	}
	var projects []gitlabProject
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, err
	}
	out := make([]Result, 0, len(projects))
	for _, project := range projects {
		if project.Path == "" || project.WebURL == "" {
			continue
		}
		branch := project.DefaultBranch
		if branch == "" {
			branch = "main"
		}
		raw := strings.TrimRight(project.WebURL, "/") + "/-/raw/" + url.PathEscape(branch) + "/README.md"
		out = append(out, Result{
			URL:     raw,
			Title:   project.Path,
			Snippet: project.Description,
			Source:  "gitlab",
		})
	}
	return out, nil
}
