package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"nie/internal/config"
	"nie/internal/crawler"
	"nie/internal/discovery"
	"nie/internal/domain"
	"nie/internal/extract"
	"nie/internal/storage"
	"nie/internal/store"
	"nie/internal/urlutil"
)

type Worker struct {
	store   *store.Store
	storage *storage.Store
	fetcher *crawler.Fetcher
	robots  *crawler.Robots
	searx   *discovery.SearXNG
	github  *discovery.GitHub
	cfg     config.Config
	logger  *slog.Logger
}

var errDeferred = errors.New("job adiado")

type jobError struct {
	err        error
	retryAfter time.Duration
	permanent  bool
}

func (e *jobError) Error() string { return e.err.Error() }
func (e *jobError) Unwrap() error { return e.err }

func New(st *store.Store, storage *storage.Store, fetcher *crawler.Fetcher, robots *crawler.Robots,
	searx *discovery.SearXNG, github *discovery.GitHub, cfg config.Config, logger *slog.Logger) *Worker {
	return &Worker{
		store: st, storage: storage, fetcher: fetcher, robots: robots,
		searx: searx, github: github, cfg: cfg, logger: logger,
	}
}

func (w *Worker) Run(ctx context.Context) {
	if !w.cfg.WorkerEnabled {
		w.logger.Info("worker desativado")
		return
	}
	count := w.cfg.WorkerCount
	if count < 1 {
		count = 1
	}
	w.logger.Info("worker iniciado", "workers", count, "poll", w.cfg.PollInterval)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		w.reapLoop(ctx)
	}()
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			w.loop(ctx, id)
		}(i + 1)
	}
	wg.Wait()
	w.logger.Info("worker encerrado")
}

func (w *Worker) reapLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		if n, err := w.store.ReapExpiredJobs(ctx); err != nil {
			w.logger.Error("reaper de leases", "err", err)
		} else if n > 0 {
			w.logger.Warn("jobs com lease expirado encerrados", "count", n)
		}
		if n, err := w.store.FinishIdleCampaigns(ctx); err != nil {
			w.logger.Error("finalizar campanhas ociosas", "err", err)
		} else if n > 0 {
			w.logger.Info("campanhas concluídas", "count", n)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *Worker) loop(ctx context.Context, id int) {
	for {
		if ctx.Err() != nil {
			return
		}
		job, err := w.store.ClaimJob(ctx, 2*time.Minute)
		if err != nil {
			w.logger.Error("claim job", "worker", id, "err", err)
			sleep(ctx, w.cfg.PollInterval)
			continue
		}
		if job == nil {
			sleep(ctx, w.cfg.PollInterval)
			continue
		}
		w.logger.Info("job recebido", "worker", id, "job", job.ID, "kind", job.Kind, "url", job.URL, "depth", job.Depth, "attempt", job.Attempts)
		err = w.process(ctx, job)
		switch {
		case err == nil:
			if err := w.store.CompleteJob(ctx, job.ID); err != nil {
				w.logger.Error("completar job", "job", job.ID, "err", err)
			}
		case errors.Is(err, errDeferred):
			// O job já foi reagendado pelo throttle.
		default:
			var je *jobError
			if errors.As(err, &je) {
				if je.permanent || job.Attempts >= job.MaxAttempts {
					w.logger.Warn("job encerrado", "job", job.ID, "url", job.URL, "err", err)
					_ = w.store.DeadJob(ctx, job.ID, err)
				} else {
					delay := je.retryAfter
					if delay <= 0 {
						delay = backoff(job.Attempts)
					}
					w.logger.Warn("job reagendado", "job", job.ID, "url", job.URL, "delay", delay, "err", err)
					_ = w.store.RetryJob(ctx, job.ID, delay, err)
				}
			} else {
				w.logger.Warn("job falhou", "job", job.ID, "url", job.URL, "err", err)
				if job.Attempts >= job.MaxAttempts {
					_ = w.store.DeadJob(ctx, job.ID, err)
				} else {
					_ = w.store.RetryJob(ctx, job.ID, backoff(job.Attempts), err)
				}
			}
		}
		if err := w.store.FinishIfIdle(ctx, job.CampaignID); err != nil {
			w.logger.Error("finalizar campanha", "campaign", job.CampaignID, "err", err)
		}
	}
}

func (w *Worker) process(ctx context.Context, job *domain.Job) error {
	switch job.Kind {
	case "fetch":
		return w.processFetch(ctx, job)
	case "discover_web":
		return w.processSearX(ctx, job)
	case "discover_github":
		return w.processGitHub(ctx, job)
	default:
		return &jobError{err: fmt.Errorf("kind desconhecido: %s", job.Kind), permanent: true}
	}
}

func (w *Worker) processFetch(ctx context.Context, job *domain.Job) error {
	u, norm, err := urlutil.ParseNormalized(job.URL)
	if err != nil {
		return &jobError{err: err, permanent: true}
	}
	budget, used, err := w.store.CampaignContext(ctx, job.CampaignID)
	if err != nil {
		return err
	}
	if used >= budget.MaxBytes {
		w.logger.Info("orçamento de bytes esgotado; job ignorado", "campaign", job.CampaignID, "used", used, "max", budget.MaxBytes)
		return nil
	}
	if job.Depth > budget.MaxDepth {
		return nil
	}

	interval := time.Duration(budget.MinDomainIntervalSeconds) * time.Second
	if interval <= 0 {
		interval = w.cfg.DomainInterval()
	}
	next, ok, err := w.store.Throttle(ctx, u.Hostname(), interval)
	if err != nil {
		return err
	}
	if !ok {
		if err := w.store.DeferJob(ctx, job.ID, next); err != nil {
			return err
		}
		return errDeferred
	}

	if !w.robots.Allowed(ctx, u) {
		w.logger.Info("robots.txt bloqueou", "url", norm)
		return nil
	}

	resp, err := w.fetcher.Fetch(ctx, norm)
	if err != nil {
		return classifyFetchError(err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		switch {
		case resp.StatusCode == 429:
			return &jobError{err: &crawler.HTTPError{StatusCode: 429, URL: norm}, retryAfter: retryAfter(resp.Header, 60*time.Second)}
		case resp.StatusCode == 408 || resp.StatusCode == 425:
			return &jobError{err: &crawler.HTTPError{StatusCode: resp.StatusCode, URL: norm}}
		case resp.StatusCode >= 500:
			return &jobError{err: &crawler.HTTPError{StatusCode: resp.StatusCode, URL: norm}}
		default:
			w.logger.Info("resposta ignorada", "url", norm, "status", resp.StatusCode)
			return nil
		}
	}

	finalRaw := resp.URL
	if finalRaw == "" {
		finalRaw = norm
	}
	finalNorm, err := urlutil.Normalize(finalRaw)
	if err != nil {
		finalNorm = norm
	}
	finalURL, err := url.Parse(finalNorm)
	if err != nil {
		finalURL = u
	}

	var ex extract.Result
	switch {
	case strings.HasPrefix(resp.ContentType, "application/pdf"):
		ex, err = extract.PDF(resp.Body)
	case strings.HasPrefix(resp.ContentType, "text/html") || strings.HasPrefix(resp.ContentType, "application/xhtml+xml"):
		ex, err = extract.HTML(finalURL, resp.ContentType, resp.Body)
	default:
		ex, err = extract.Text(resp.Body)
	}
	if err != nil {
		return &jobError{err: err, permanent: true}
	}
	if strings.TrimSpace(ex.Text) == "" {
		w.logger.Info("sem texto extraído", "url", finalNorm)
		return nil
	}

	stored, err := w.storage.Put(resp.Body)
	if err != nil {
		return err
	}
	passages := make([]domain.Passage, 0, len(ex.Passages))
	for _, p := range ex.Passages {
		passages = append(passages, domain.Passage{Ordinal: p.Ordinal, Page: p.Page, Text: p.Text})
	}
	docID, created, err := w.store.RecordDocument(ctx, store.DocumentRecord{
		SHA256:        stored.SHA256,
		MIME:          ex.MIME,
		Title:         ex.Title,
		Size:          stored.Size,
		Text:          ex.Text,
		Metadata:      map[string]any{"requested_url": norm, "final_url": finalNorm, "passages": len(passages)},
		CampaignID:    job.CampaignID,
		URL:           finalNorm,
		NormalizedURL: finalNorm,
		Domain:        urlutil.Domain(finalNorm),
		StatusCode:    resp.StatusCode,
		ETag:          resp.Header.Get("ETag"),
		LastModified:  resp.Header.Get("Last-Modified"),
		Passages:      passages,
	})
	if err != nil {
		return err
	}
	w.logger.Info("documento processado", "url", finalNorm, "doc", docID, "sha256", stored.SHA256, "created", created, "passages", len(passages), "bytes", stored.Size)

	if job.Depth < budget.MaxDepth {
		enqueued := 0
		for _, link := range ex.Links {
			if !crawlableLink(link) {
				continue
			}
			ok, err := w.store.EnqueueFetch(ctx, job.CampaignID, link, job.Depth+1, finalNorm, "link", 100)
			if err != nil {
				w.logger.Warn("enfileirar link", "url", link, "err", err)
				continue
			}
			if ok {
				enqueued++
			}
		}
		if enqueued > 0 {
			w.logger.Info("novas pistas", "from", finalNorm, "count", enqueued)
		}
	}
	return nil
}

func (w *Worker) processSearX(ctx context.Context, job *domain.Job) error {
	if w.searx == nil {
		w.logger.Info("descoberta web não configurada; job ignorado")
		return nil
	}
	var payload struct {
		Query string `json:"query"`
	}
	_ = json.Unmarshal(job.Payload, &payload)
	if strings.TrimSpace(payload.Query) == "" {
		return &jobError{err: errors.New("query de descoberta vazia"), permanent: true}
	}
	results, err := w.searx.Search(ctx, payload.Query, 10)
	if err != nil {
		return err
	}
	enqueued := 0
	for _, r := range results {
		ok, err := w.store.EnqueueFetch(ctx, job.CampaignID, r.URL, 0, "", "searxng", 20)
		if err != nil {
			w.logger.Warn("enfileirar resultado SearXNG", "url", r.URL, "err", err)
			continue
		}
		if ok {
			enqueued++
		}
	}
	w.logger.Info("descoberta web", "query", payload.Query, "results", len(results), "enqueued", enqueued)
	return nil
}

func (w *Worker) processGitHub(ctx context.Context, job *domain.Job) error {
	if w.github == nil {
		w.logger.Info("descoberta GitHub não configurada; job ignorado")
		return nil
	}
	var payload struct {
		Query string `json:"query"`
	}
	_ = json.Unmarshal(job.Payload, &payload)
	query := discovery.SanitizeQuery(payload.Query)
	if query == "" {
		return &jobError{err: errors.New("query GitHub vazia"), permanent: true}
	}
	results, err := w.github.SearchReadmes(ctx, query, 5)
	if err != nil {
		return err
	}
	enqueued := 0
	for _, r := range results {
		ok, err := w.store.EnqueueFetch(ctx, job.CampaignID, r.URL, 0, "", "github", 15)
		if err != nil {
			w.logger.Warn("enfileirar README", "url", r.URL, "err", err)
			continue
		}
		if ok {
			enqueued++
		}
	}
	w.logger.Info("descoberta GitHub", "query", query, "results", len(results), "enqueued", enqueued)
	return nil
}

func classifyFetchError(err error) error {
	switch {
	case errors.Is(err, crawler.ErrBlocked),
		errors.Is(err, crawler.ErrTooLarge),
		errors.Is(err, crawler.ErrUnsupportedType),
		errors.Is(err, urlutil.ErrInvalidURL),
		errors.Is(err, urlutil.ErrUnsupported),
		errors.Is(err, urlutil.ErrPrivateHost):
		return &jobError{err: err, permanent: true}
	default:
		return &jobError{err: err}
	}
}

func retryAfter(h http.Header, def time.Duration) time.Duration {
	v := strings.TrimSpace(h.Get("Retry-After"))
	if v == "" {
		return def
	}
	if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return def
}

func backoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	d := 30 * time.Second * time.Duration(1<<uint(min(attempt-1, 5)))
	if d > 10*time.Minute {
		d = 10 * time.Minute
	}
	jitter := time.Duration(rand.Int63n(int64(d / 5)))
	return d + jitter
}

func sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

func crawlableLink(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	ext := strings.ToLower(path.Ext(u.Path))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".ico", ".css", ".js", ".mjs",
		".zip", ".rar", ".7z", ".tar", ".gz", ".bz2", ".xz", ".exe", ".msi", ".dmg",
		".iso", ".img", ".bin", ".mp3", ".mp4", ".avi", ".mkv", ".mov", ".woff", ".woff2",
		".ttf", ".eot", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx":
		return false
	}
	return true
}
