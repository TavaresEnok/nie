package discovery

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// RemoteFile é um arquivo em um ref imutável. O SHA é o blob ou o commit
// informado pela API, não uma prova de que o working tree local está igual.
type RemoteFile struct {
	Host    string `json:"host"`
	Repo    string `json:"repo"`
	Path    string `json:"path"`
	Ref     string `json:"ref"`
	SHA     string `json:"sha"`
	URL     string `json:"url"`
	Content string `json:"content"`
}

func (g *GitHub) File(ctx context.Context, repo, path, ref string) (RemoteFile, error) {
	if g == nil {
		return RemoteFile{}, fmt.Errorf("GitHub não configurado")
	}
	repo = strings.Trim(repo, "/")
	path = strings.Trim(path, "/")
	if repo == "" || path == "" || strings.TrimSpace(ref) == "" {
		return RemoteFile{}, fmt.Errorf("repositório, caminho e revisão são obrigatórios")
	}
	base := "https://api.github.com"
	if g.base != "" {
		base = strings.TrimRight(g.base, "/")
	}
	endpoint := base + "/repos/" + repo + "/contents/" + path + "?ref=" + url.QueryEscape(ref)
	var payload struct {
		SHA         string `json:"sha"`
		HTMLURL     string `json:"html_url"`
		DownloadURL string `json:"download_url"`
		Encoding    string `json:"encoding"`
		Content     string `json:"content"`
	}
	if err := g.getJSON(ctx, endpoint, &payload); err != nil {
		return RemoteFile{}, err
	}
	content, err := decodeGitContent(payload.Encoding, payload.Content)
	if err != nil {
		return RemoteFile{}, err
	}
	fileURL := payload.HTMLURL
	if fileURL == "" {
		fileURL = payload.DownloadURL
	}
	return RemoteFile{Host: "github", Repo: repo, Path: path, Ref: ref, SHA: payload.SHA, URL: fileURL, Content: content}, nil
}

func (g *GitLab) File(ctx context.Context, repo, path, ref string) (RemoteFile, error) {
	if g == nil {
		return RemoteFile{}, fmt.Errorf("GitLab não configurado")
	}
	repo = strings.Trim(repo, "/")
	path = strings.Trim(path, "/")
	if repo == "" || path == "" || strings.TrimSpace(ref) == "" {
		return RemoteFile{}, fmt.Errorf("repositório, caminho e revisão são obrigatórios")
	}
	endpoint := g.base + "/api/v4/projects/" + url.PathEscape(repo) + "/repository/files/" + url.PathEscape(path) + "?ref=" + url.QueryEscape(ref)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return RemoteFile{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "NIEBot/0.1")
	if g.token != "" {
		req.Header.Set("PRIVATE-TOKEN", g.token)
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return RemoteFile{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return RemoteFile{}, fmt.Errorf("GitLab HTTP %d", resp.StatusCode)
	}
	var payload struct {
		BlobID   string `json:"blob_id"`
		FileName string `json:"file_name"`
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return RemoteFile{}, err
	}
	content, err := decodeGitContent(payload.Encoding, payload.Content)
	if err != nil {
		return RemoteFile{}, err
	}
	raw := strings.TrimRight(g.base, "/") + "/" + repo + "/-/raw/" + url.PathEscape(ref) + "/" + path
	return RemoteFile{Host: "gitlab", Repo: repo, Path: path, Ref: ref, SHA: payload.BlobID, URL: raw, Content: content}, nil
}

func decodeGitContent(encoding, content string) (string, error) {
	if strings.EqualFold(encoding, "base64") || encoding == "" {
		raw, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(content, "\n", ""))
		if err != nil {
			return "", err
		}
		if len(raw) > 1<<20 {
			raw = raw[:1<<20]
		}
		return string(raw), nil
	}
	return "", fmt.Errorf("codificação não suportada: %s", encoding)
}

// ReadLimited exists so git file fetches stay bounded if a caller streams.
func ReadLimited(r io.Reader, n int64) ([]byte, error) {
	return io.ReadAll(io.LimitReader(r, n))
}
