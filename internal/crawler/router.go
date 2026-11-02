package crawler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"nie/internal/domain"
	"nie/internal/extract"
	"nie/internal/netguard"
	"nie/internal/urlutil"
)

var (
	ErrNoExecutorAvailable = errors.New("nenhum executor disponível para a tarefa")
)

func validatePublicTarget(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("URL inválida: %w", err)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("host vazio")
	}
	if ip := net.ParseIP(host); ip != nil {
		if !netguard.IsPublicIP(ip) {
			return fmt.Errorf("IP %s bloqueado por política de egress", ip.String())
		}
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("resolver host %s: %w", host, err)
	}
	for _, ip := range ips {
		if !netguard.IsPublicIP(ip) {
			return fmt.Errorf("resolução de %s para %s bloqueada por política de egress", host, ip.String())
		}
	}
	return nil
}

// Executor define a interface canônica para qualquer motor de coleta (Native Go, Scrapy, Crawl4AI).
// Atende à diretriz do ADENDO MESTRE V2.1: todas as saídas passam pelo mesmo contrato.
type Executor interface {
	Name() string
	Execute(ctx context.Context, task domain.FetchTask) (*domain.FetchResult, error)
}

// NativeExecutor executa coletas diretamente em Go com netguard, robots e limites de segurança.
type NativeExecutor struct {
	fetcher *Fetcher
}

func NewNativeExecutor(fetcher *Fetcher) *NativeExecutor {
	return &NativeExecutor{fetcher: fetcher}
}

func (n *NativeExecutor) Name() string { return "native" }

func (n *NativeExecutor) Execute(ctx context.Context, task domain.FetchTask) (*domain.FetchResult, error) {
	start := time.Now()
	resp, err := n.fetcher.Fetch(ctx, task.URL)
	duration := time.Since(start)

	if err != nil {
		return &domain.FetchResult{
			TaskID:       task.ID,
			EffectiveURL: task.URL,
			Duration:     duration,
			Error:        err.Error(),
		}, err
	}

	finalURL, _ := url.Parse(resp.URL)
	var ex extract.Result
	switch {
	case strings.HasPrefix(resp.ContentType, "application/pdf"):
		ex, _ = extract.PDF(resp.Body)
	case strings.HasPrefix(resp.ContentType, "text/html") || strings.HasPrefix(resp.ContentType, "application/xhtml+xml"):
		ex, _ = extract.HTML(finalURL, resp.ContentType, resp.Body)
	default:
		ex, _ = extract.Text(resp.Body)
	}

	mentions := extract.ExtractMentions(ex.Text, 25)

	return &domain.FetchResult{
		TaskID:            task.ID,
		EffectiveURL:      resp.URL,
		StatusCode:        resp.StatusCode,
		ContentType:       resp.ContentType,
		RawContent:        resp.Body,
		Text:              ex.Text,
		Title:             ex.Title,
		ETag:              resp.Header.Get("ETag"),
		LastModified:      resp.Header.Get("Last-Modified"),
		ExtractedLinks:    ex.Links,
		ExtractedMentions: mentions,
		Duration:          duration,
		Metadata: map[string]any{
			"executor":   "native",
			"byte_count": len(resp.Body),
			"title":      ex.Title,
			"mime":       ex.MIME,
			"passages":   ex.Passages,
		},
	}, nil
}

// ScrapyBridge envia a tarefa de coleta em lote/recursiva para o worker Scrapy isolado.
type ScrapyBridge struct {
	endpoint string
	client   *http.Client
	fallback Executor
}

func NewScrapyBridge(endpoint string, fallback Executor) *ScrapyBridge {
	return &ScrapyBridge{
		endpoint: strings.TrimRight(strings.TrimSpace(endpoint), "/"),
		client:   &http.Client{Timeout: 30 * time.Second},
		fallback: fallback,
	}
}

func (s *ScrapyBridge) Name() string { return "scrapy" }

func (s *ScrapyBridge) Execute(ctx context.Context, task domain.FetchTask) (*domain.FetchResult, error) {
	if err := validatePublicTarget(task.URL); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBlocked, err)
	}
	if s.endpoint == "" {
		if s.fallback != nil {
			return s.fallback.Execute(ctx, task)
		}
		return nil, fmt.Errorf("worker Scrapy não configurado")
	}

	payload, err := json.Marshal(map[string]any{
		"task_id":     task.ID,
		"url":         task.URL,
		"campaign_id": task.CampaignID,
		"max_bytes":   task.MaxBytes,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint+"/crawl", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := s.client.Do(req)
	duration := time.Since(start)

	if err != nil {
		if s.fallback != nil {
			return s.fallback.Execute(ctx, task)
		}
		return nil, fmt.Errorf("comunicação com Scrapy bridge: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if s.fallback != nil {
			return s.fallback.Execute(ctx, task)
		}
		return nil, fmt.Errorf("Scrapy bridge retornou HTTP %d", resp.StatusCode)
	}

	var res struct {
		EffectiveURL   string   `json:"effective_url"`
		StatusCode     int      `json:"status_code"`
		ContentType    string   `json:"content_type"`
		Text           string   `json:"text"`
		ExtractedLinks []string `json:"extracted_links"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	mentions := extract.ExtractMentions(res.Text, 25)

	return &domain.FetchResult{
		TaskID:            task.ID,
		EffectiveURL:      res.EffectiveURL,
		StatusCode:        res.StatusCode,
		ContentType:       res.ContentType,
		RawContent:        []byte(res.Text),
		Text:              res.Text,
		ExtractedLinks:    res.ExtractedLinks,
		ExtractedMentions: mentions,
		Duration:          duration,
		Metadata: map[string]any{
			"executor": "scrapy",
		},
	}, nil
}

// Crawl4AIBridge envia requisições dinâmicas/JS e deep crawl para o worker Crawl4AI isolado.
type Crawl4AIBridge struct {
	endpoint string
	client   *http.Client
	fallback Executor
}

func NewCrawl4AIBridge(endpoint string, fallback Executor) *Crawl4AIBridge {
	return &Crawl4AIBridge{
		endpoint: strings.TrimRight(strings.TrimSpace(endpoint), "/"),
		client:   &http.Client{Timeout: 45 * time.Second},
		fallback: fallback,
	}
}

func (c *Crawl4AIBridge) Name() string { return "crawl4ai" }

func (c *Crawl4AIBridge) Execute(ctx context.Context, task domain.FetchTask) (*domain.FetchResult, error) {
	if err := validatePublicTarget(task.URL); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBlocked, err)
	}
	if c.endpoint == "" {
		if c.fallback != nil {
			return c.fallback.Execute(ctx, task)
		}
		return nil, fmt.Errorf("worker Crawl4AI não configurado")
	}

	payload, err := json.Marshal(map[string]any{
		"task_id":     task.ID,
		"url":         task.URL,
		"campaign_id": task.CampaignID,
		"mode":        task.Mode,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/crawl", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := c.client.Do(req)
	duration := time.Since(start)

	if err != nil {
		if c.fallback != nil {
			return c.fallback.Execute(ctx, task)
		}
		return nil, fmt.Errorf("comunicação com Crawl4AI bridge: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if c.fallback != nil {
			return c.fallback.Execute(ctx, task)
		}
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Crawl4AI bridge HTTP %d: %s", resp.StatusCode, string(body))
	}

	var res struct {
		EffectiveURL   string   `json:"effective_url"`
		StatusCode     int      `json:"status_code"`
		ContentType    string   `json:"content_type"`
		Markdown       string   `json:"markdown"`
		ExtractedLinks []string `json:"extracted_links"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	mentions := extract.ExtractMentions(res.Markdown, 25)

	return &domain.FetchResult{
		TaskID:            task.ID,
		EffectiveURL:      res.EffectiveURL,
		StatusCode:        res.StatusCode,
		ContentType:       res.ContentType,
		RawContent:        []byte(res.Markdown),
		Text:              res.Markdown,
		ExtractedLinks:    res.ExtractedLinks,
		ExtractedMentions: mentions,
		Duration:          duration,
		Metadata: map[string]any{
			"executor": "crawl4ai",
			"format":   "markdown",
		},
	}, nil
}

// Router gerencia e seleciona o executor adequado para cada tarefa.
// Garante o princípio fundamental: nunca executar dois crawlers concorrentes na mesma URL.
type Router struct {
	mu        sync.RWMutex
	executors map[string]Executor
}

func NewRouter() *Router {
	return &Router{
		executors: make(map[string]Executor),
	}
}

func (r *Router) Register(exec Executor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.executors[exec.Name()] = exec
}

// SelectExecutor escolhe o executor adequado conforme política da tarefa e características do destino.
func (r *Router) SelectExecutor(task domain.FetchTask) Executor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 1. Escolha explícita na tarefa
	if task.Executor != "" {
		if exec, ok := r.executors[task.Executor]; ok {
			return exec
		}
	}

	// 2. Modo browser ou renderização JS necessária -> Crawl4AI
	if task.Mode == "browser" {
		if exec, ok := r.executors["crawl4ai"]; ok {
			return exec
		}
	}

	// 3. Destinos estáticos ou padrão -> Native
	if exec, ok := r.executors["native"]; ok {
		return exec
	}

	// 4. Qualquer executor registrado
	for _, exec := range r.executors {
		return exec
	}

	return nil
}

// Execute despacha a tarefa para o executor selecionado pelo roteador.
func (r *Router) Execute(ctx context.Context, task domain.FetchTask) (*domain.FetchResult, error) {
	exec := r.SelectExecutor(task)
	if exec == nil {
		return nil, ErrNoExecutorAvailable
	}
	return exec.Execute(ctx, task)
}

// BuildFetchTask constrói um FetchTask a partir de uma URL e parâmetros de campanha.
func BuildFetchTask(campaignID, rawURL, mode, preferredExecutor string, maxBytes int64) domain.FetchTask {
	return domain.FetchTask{
		ID:           fmt.Sprintf("task-%d", time.Now().UnixNano()),
		CampaignID:   campaignID,
		URL:          rawURL,
		Mode:         mode,
		Executor:     preferredExecutor,
		MaxBytes:     maxBytes,
		DomainPolicy: urlutil.Domain(rawURL),
		CreatedAt:    time.Now(),
	}
}
