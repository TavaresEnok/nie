package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Researcher fala com um GPT Researcher já em execução. O texto volta como
// relatório externo e não vira claim certificado.
type Researcher struct {
	base   string
	client *http.Client
}

func NewResearcher(base string) *Researcher {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return nil
	}
	return &Researcher{base: base, client: &http.Client{Timeout: 2 * time.Minute}}
}

func (r *Researcher) Report(ctx context.Context, query string) (string, error) {
	if r == nil || r.base == "" {
		return "", fmt.Errorf("GPT Researcher não configurado")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return "", fmt.Errorf("consulta vazia")
	}
	body, _ := json.Marshal(map[string]string{"query": query})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.base+"/report", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("GPT Researcher HTTP %d", resp.StatusCode)
	}
	var payload struct {
		Report string `json:"report"`
	}
	if json.Unmarshal(raw, &payload) == nil && strings.TrimSpace(payload.Report) != "" {
		return payload.Report, nil
	}
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return "", fmt.Errorf("GPT Researcher não devolveu relatório")
	}
	return text, nil
}
