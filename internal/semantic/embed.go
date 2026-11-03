package semantic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const Dimensions = 768

// Client pede um vetor a um endpoint compatível com OpenAI ou Ollama.
// Sem URL, não há cliente: a busca lexical segue sozinha.
type Client struct {
	endpoint string
	model    string
	apiKey   string
	http     *http.Client
}

func New(endpoint, model, apiKey string) *Client {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil
	}
	if strings.TrimSpace(model) == "" {
		model = "nomic-embed-text"
	}
	return &Client{
		endpoint: endpoint,
		model:    model,
		apiKey:   strings.TrimSpace(apiKey),
		http:     &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Model() string {
	if c == nil {
		return ""
	}
	return c.model
}

// Embed devolve 768 números. Outro tamanho é recusado para caber em passage_embeddings.
func (c *Client) Embed(ctx context.Context, text string) ([]float32, error) {
	if c == nil || c.endpoint == "" {
		return nil, fmt.Errorf("embeddings não configurados")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("texto vazio")
	}
	if len(text) > 4000 {
		text = text[:4000]
	}
	body, _ := json.Marshal(map[string]any{"model": c.model, "input": text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("embeddings HTTP %d", resp.StatusCode)
	}
	var payload struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
		Embeddings [][]float32 `json:"embeddings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	var vec []float32
	if len(payload.Data) > 0 {
		vec = payload.Data[0].Embedding
	} else if len(payload.Embeddings) > 0 {
		vec = payload.Embeddings[0]
	}
	if len(vec) != Dimensions {
		return nil, fmt.Errorf("embedding tem %d dimensões; a tabela exige %d", len(vec), Dimensions)
	}
	return vec, nil
}
