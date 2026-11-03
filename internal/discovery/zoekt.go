package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Zoekt consulta um índice já construído. Não pesquisa repositórios que nunca
// foram indexados nesse servidor.
type Zoekt struct {
	base   string
	client *http.Client
}

type CodeHit struct {
	Repo     string `json:"repo"`
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Fragment string `json:"fragment"`
}

func NewZoekt(base string) *Zoekt {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return nil
	}
	return &Zoekt{base: base, client: &http.Client{Timeout: 20 * time.Second}}
}

func (z *Zoekt) Search(ctx context.Context, query string, limit int) ([]CodeHit, error) {
	if z == nil || z.base == "" {
		return nil, fmt.Errorf("Zoekt não configurado")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("consulta Zoekt vazia")
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	endpoint := z.base + "/api/search?num=" + strconv.Itoa(limit) + "&q=" + url.QueryEscape(query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "NIEBot/0.1")
	resp, err := z.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Zoekt HTTP %d", resp.StatusCode)
	}
	var payload struct {
		Result struct {
			FileMatches []struct {
				FileName string `json:"FileName"`
				Repo     string `json:"Repo"`
				Matches  []struct {
					LineNum   int `json:"LineNum"`
					Fragments []struct {
						Pre string `json:"Pre"`
					} `json:"Fragments"`
				} `json:"Matches"`
			} `json:"FileMatches"`
		} `json:"Result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	var out []CodeHit
	for _, file := range payload.Result.FileMatches {
		if len(file.Matches) == 0 {
			out = append(out, CodeHit{Repo: file.Repo, Path: file.FileName})
			continue
		}
		for _, match := range file.Matches {
			fragment := ""
			if len(match.Fragments) > 0 {
				fragment = match.Fragments[0].Pre
			}
			out = append(out, CodeHit{Repo: file.Repo, Path: file.FileName, Line: match.LineNum, Fragment: fragment})
			if len(out) >= limit {
				return out, nil
			}
		}
	}
	return out, nil
}
