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

// WaybackSnapshot armazena a versão histórica recuperada do Internet Archive.
type WaybackSnapshot struct {
	OriginalURL string    `json:"original_url"`
	SnapshotURL string    `json:"snapshot_url"`
	Timestamp   time.Time `json:"timestamp"`
	Available   bool      `json:"available"`
}

// WaybackMachine integra a API de disponibilidade da Wayback Machine (Internet Archive).
type WaybackMachine struct {
	base   string
	client *http.Client
}

func NewWaybackMachine(base ...string) *WaybackMachine {
	endpoint := "https://archive.org/wayback/available"
	if len(base) > 0 && strings.TrimSpace(base[0]) != "" {
		endpoint = strings.TrimRight(strings.TrimSpace(base[0]), "/")
	}
	return &WaybackMachine{
		base:   endpoint,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// CheckAvailability verifica se há uma cópia histórica preservada para uma URL (útil em caso de 404).
func (w *WaybackMachine) CheckAvailability(ctx context.Context, targetURL string) (*WaybackSnapshot, error) {
	if w == nil || w.base == "" {
		return nil, fmt.Errorf("wayback machine não configurada")
	}
	targetURL = strings.TrimSpace(targetURL)
	if targetURL == "" {
		return nil, fmt.Errorf("URL alvo vazia")
	}

	endpoint := fmt.Sprintf("%s?url=%s", w.base, url.QueryEscape(targetURL))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "NIE-Crawler/2.1 (Network Intelligence Engine)")

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Wayback HTTP %d", resp.StatusCode)
	}

	var payload struct {
		ArchivedSnapshots struct {
			Closest struct {
				Available bool   `json:"available"`
				URL       string `json:"url"`
				Timestamp string `json:"timestamp"`
				Status    string `json:"status"`
			} `json:"closest"`
		} `json:"archived_snapshots"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("parse Wayback JSON: %w", err)
	}

	closest := payload.ArchivedSnapshots.Closest
	if !closest.Available || closest.URL == "" {
		return nil, nil // Não disponível
	}

	var parsedTime time.Time
	if len(closest.Timestamp) == 14 {
		parsedTime, _ = time.Parse("20060102150405", closest.Timestamp)
	}

	return &WaybackSnapshot{
		OriginalURL: targetURL,
		SnapshotURL: closest.URL,
		Timestamp:   parsedTime,
		Available:   true,
	}, nil
}
