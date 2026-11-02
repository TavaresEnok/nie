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
	router  *crawler.Router
	robots  *crawler.Robots
	searx   *discovery.SearXNG
	github  *discovery.GitHub
	mwmbl   *discovery.Mwmbl
	wayback *discovery.WaybackMachine
	swh     *discovery.SoftwareHeritage
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

	// Redirecionamentos também precisam respeitar o robots.txt do destino final.
	if robots != nil && fetcher != nil {
		fetcher.SetRedirectCheck(func(ctx context.Context, u *url.URL) error {
			if !robots.Allowed(ctx, u) {
				return fmt.Errorf("%w: %s", crawler.ErrRobotsDisallowed, u.String())
			}
			return nil
		})
	}

	router := crawler.NewRouter()
	native := crawler.NewNativeExecutor(fetcher)
	router.Register(native)

	if cfg.ScrapyURL != "" {
		router.Register(crawler.NewScrapyBridge(cfg.ScrapyURL, native))
		logger.Info("executor Scrapy registrado", "url", cfg.ScrapyURL)
	}
	if cfg.Crawl4AIURL != "" {
		router.Register(crawler.NewCrawl4AIBridge(cfg.Crawl4AIURL, native))
		logger.Info("executor Crawl4AI registrado", "url", cfg.Crawl4AIURL)
	}

	var mwmbl *discovery.Mwmbl
	if cfg.MwmblURL != "" {
		mwmbl = discovery.NewMwmbl(cfg.MwmblURL)
		logger.Info("descoberta Mwmbl habilitada", "url", cfg.MwmblURL)
	}

	var wayback *discovery.WaybackMachine
	if cfg.WaybackEnabled {
		wayback = discovery.NewWaybackMachine()
	}

	var swh *discovery.SoftwareHeritage
	if cfg.SoftwareHeritageEnabled {
		swh = discovery.NewSoftwareHeritage()
	}

	return &Worker{
		store:   st,
		storage: storage,
		fetcher: fetcher,
		router:  router,
		robots:  robots,
		searx:   searx,
		github:  github,
		mwmbl:   mwmbl,
		wayback: wayback,
		swh:     swh,
		cfg:     cfg,
		logger:  logger,
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
	case "discover_mwmbl":
		return w.processMwmbl(ctx, job)
	case "discover_swh":
		return w.processSoftwareHeritage(ctx, job)
	default:
		return &jobError{err: fmt.Errorf("kind desconhecido: %s", job.Kind), permanent: true}
	}
}

func (w *Worker) processFetch(ctx context.Context, job *domain.Job) error {
	u, norm, err := urlutil.ParseNormalized(job.URL)
	if err != nil {
		return &jobError{err: err, permanent: true}
	}
	budget, used, campaignQuery, err := w.store.CampaignContext(ctx, job.CampaignID)
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

	mode := "http"
	executor := ""
	lowerURL := strings.ToLower(norm)
	if strings.Contains(lowerURL, "/#/") || strings.Contains(lowerURL, "/app/") || strings.Contains(lowerURL, "/portal/") || strings.Contains(lowerURL, "/dashboard/") {
		mode = "browser"
		executor = "crawl4ai"
	} else if strings.Contains(lowerURL, "/docs/") || strings.Contains(lowerURL, "/manual/") || strings.Contains(lowerURL, "/wiki/") || strings.Contains(lowerURL, "/catalog/") || strings.Contains(lowerURL, "/reference/") {
		executor = "scrapy"
	}

	perDocument := documentByteLimit(w.cfg.MaxHTMLBytes, w.cfg.MaxPDFBytes,
		strings.HasSuffix(strings.ToLower(u.Path), ".pdf"), budget.MaxBytes, used)

	task := crawler.BuildFetchTask(job.CampaignID, norm, mode, executor, perDocument)
	fetchResult, err := w.router.Execute(ctx, task)
	if err != nil {
		return classifyFetchError(err)
	}
	if fetchResult.StatusCode < 200 || fetchResult.StatusCode >= 300 {
		if (fetchResult.StatusCode == 404 || fetchResult.StatusCode == 410) && w.wayback != nil && w.cfg.WaybackEnabled {
			if snap, wbErr := w.wayback.CheckAvailability(ctx, norm); wbErr == nil && snap != nil && snap.Available && snap.SnapshotURL != "" {
				w.logger.Info("recurso recuperado via Wayback Machine", "original", norm, "snapshot", snap.SnapshotURL)
				_, _ = w.store.EnqueueFetch(ctx, job.CampaignID, snap.SnapshotURL, job.Depth, norm, "wayback", 30)
			}
		}
		switch {
		case fetchResult.StatusCode == 429:
			return &jobError{err: &crawler.HTTPError{StatusCode: 429, URL: norm}, retryAfter: 60 * time.Second}
		case fetchResult.StatusCode == 408 || fetchResult.StatusCode == 425:
			return &jobError{err: &crawler.HTTPError{StatusCode: fetchResult.StatusCode, URL: norm}}
		case fetchResult.StatusCode >= 500:
			return &jobError{err: &crawler.HTTPError{StatusCode: fetchResult.StatusCode, URL: norm}}
		default:
			w.logger.Info("resposta ignorada", "url", norm, "status", fetchResult.StatusCode)
			return nil
		}
	}

	finalRaw := fetchResult.EffectiveURL
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

	rawBytes := fetchResult.RawContent
	if len(rawBytes) == 0 && fetchResult.Text != "" {
		rawBytes = []byte(fetchResult.Text)
	}
	if len(rawBytes) == 0 {
		w.logger.Info("conteúdo vazio", "url", finalNorm)
		return nil
	}

	var ex extract.Result
	if passagesRaw, ok := fetchResult.Metadata["passages"].([]extract.Passage); ok && len(passagesRaw) > 0 {
		ex.Text = fetchResult.Text
		ex.Title = fetchResult.Title
		ex.Links = fetchResult.ExtractedLinks
		ex.Passages = passagesRaw
		if mimeStr, ok := fetchResult.Metadata["mime"].(string); ok {
			ex.MIME = mimeStr
		}
		if kind, ok := fetchResult.Metadata["link_kind"].(string); ok {
			ex.LinkKind = kind
		}
		if feeds, ok := fetchResult.Metadata["feed_links"].([]string); ok {
			ex.FeedLinks = feeds
		}
	} else {
		switch {
		case strings.HasPrefix(fetchResult.ContentType, "application/pdf"):
			ex, err = extract.PDF(rawBytes)
		case extract.IsFeedCandidate(fetchResult.ContentType, finalURL.Path):
			// Sitemap/RSS/Atom; XML de outro formato cai na extração de texto.
			if ex, err = extract.Feed(finalURL, rawBytes); err != nil {
				ex, err = extract.Text(rawBytes)
			}
		case len(fetchResult.RenderedContent) > 0:
			// DOM renderizado já vem em UTF-8 do navegador; o original HTTP vai para o CAS.
			ex, err = extract.HTML(finalURL, "text/html; charset=utf-8", fetchResult.RenderedContent)
		case strings.HasPrefix(fetchResult.ContentType, "text/html") || strings.HasPrefix(fetchResult.ContentType, "application/xhtml+xml"):
			ex, err = extract.HTML(finalURL, fetchResult.ContentType, rawBytes)
		default:
			ex, err = extract.Text(rawBytes)
		}
		if err != nil {
			// O original válido não se perde por falha de extração (ex.: PDF
			// escaneado ou protegido): fica no CAS para reprocessamento.
			return w.preserveUnextracted(ctx, job, norm, finalNorm, rawBytes, fetchResult, err.Error())
		}
	}

	if strings.TrimSpace(ex.Text) == "" && strings.TrimSpace(fetchResult.Text) != "" {
		ex.Text = fetchResult.Text
	}
	if strings.TrimSpace(ex.Text) == "" {
		return w.preserveUnextracted(ctx, job, norm, finalNorm, rawBytes, fetchResult,
			"nenhum texto extraível (possível documento escaneado; OCR pendente)")
	}

	stored, err := w.storage.Put(rawBytes)
	if err != nil {
		return err
	}
	passages := make([]domain.Passage, 0, len(ex.Passages))
	for _, p := range ex.Passages {
		passages = append(passages, domain.Passage{Ordinal: p.Ordinal, Page: p.Page, Text: p.Text})
	}
	docID, created, err := w.store.RecordDocument(ctx, store.DocumentRecord{
		SHA256: stored.SHA256,
		MIME:   ex.MIME,
		Title:  ex.Title,
		Size:   stored.Size,
		Text:   ex.Text,
		Metadata: map[string]any{
			"requested_url":   norm,
			"final_url":       finalNorm,
			"passages":        len(passages),
			"executor":        fetchResult.Metadata["executor"],
			"rendered":        len(fetchResult.RenderedContent) > 0,
			"fallback_reason": fetchResult.Metadata["fallback_reason"],
		},
		CampaignID:    job.CampaignID,
		URL:           finalNorm,
		NormalizedURL: finalNorm,
		Domain:        urlutil.Domain(finalNorm),
		StatusCode:    fetchResult.StatusCode,
		ETag:          fetchResult.ETag,
		LastModified:  fetchResult.LastModified,
		Passages:      passages,
	})
	if err != nil {
		return err
	}
	w.logger.Info("documento processado", "url", finalNorm, "doc", docID, "sha256", stored.SHA256, "created", created, "passages", len(passages), "bytes", stored.Size)

	if job.Depth < budget.MaxDepth {
		enqueued := 0
		linksToFollow := ex.Links
		if len(linksToFollow) == 0 {
			linksToFollow = fetchResult.ExtractedLinks
		}
		linkKind := "link"
		if ex.LinkKind != "" {
			linkKind = ex.LinkKind
		}
		enqueueLinks := func(links []string, kind string) {
			for _, link := range links {
				if !crawlableLink(link) {
					continue
				}
				prio := computeFrontierPriority(link, job.Depth+1, campaignQuery)
				ok, err := w.store.EnqueueFetch(ctx, job.CampaignID, link, job.Depth+1, finalNorm, kind, prio)
				if err != nil {
					w.logger.Warn("enfileirar link", "url", link, "err", err)
					continue
				}
				if ok {
					enqueued++
				}
			}
		}
		enqueueLinks(linksToFollow, linkKind)
		enqueueLinks(ex.FeedLinks, extract.LinkKindFeed)
		if enqueued > 0 {
			w.logger.Info("novas pistas", "from", finalNorm, "count", enqueued)
		}

		// V2.1: Descoberta autônoma de menções técnicas sem hyperlink (árvore que cresce sozinha)
		if w.cfg.MentionsDiscoveryEnabled {
			mentions := extract.ExtractMentions(ex.Text, 15)
			for _, mention := range mentions {
				_ = w.store.RecordDiscoveryEdge(ctx, job.CampaignID, finalNorm, "mention:"+mention, "mention", "menção técnica não linkada", job.Depth+1)
				if isSearchableMention(mention) {
					if w.searx != nil {
						_, _ = w.store.EnqueueDiscovery(ctx, job.CampaignID, "discover_web", mention)
					} else if w.mwmbl != nil {
						_, _ = w.store.EnqueueDiscovery(ctx, job.CampaignID, "discover_mwmbl", mention)
					}
				}
			}
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

func (w *Worker) processMwmbl(ctx context.Context, job *domain.Job) error {
	if w.mwmbl == nil {
		w.logger.Info("descoberta Mwmbl não configurada; job ignorado")
		return nil
	}
	var payload struct {
		Query string `json:"query"`
	}
	_ = json.Unmarshal(job.Payload, &payload)
	if strings.TrimSpace(payload.Query) == "" {
		return &jobError{err: errors.New("query de descoberta Mwmbl vazia"), permanent: true}
	}
	results, err := w.mwmbl.Search(ctx, payload.Query, 10)
	if err != nil {
		return err
	}
	enqueued := 0
	for _, r := range results {
		ok, err := w.store.EnqueueFetch(ctx, job.CampaignID, r.URL, 0, "", "mwmbl", 25)
		if err != nil {
			w.logger.Warn("enfileirar resultado Mwmbl", "url", r.URL, "err", err)
			continue
		}
		if ok {
			enqueued++
		}
	}
	w.logger.Info("descoberta Mwmbl concluída", "query", payload.Query, "results", len(results), "enqueued", enqueued)
	return nil
}

func (w *Worker) processSoftwareHeritage(ctx context.Context, job *domain.Job) error {
	if w.swh == nil {
		w.logger.Info("descoberta Software Heritage não configurada; job ignorado")
		return nil
	}
	var payload struct {
		Query string `json:"query"`
	}
	_ = json.Unmarshal(job.Payload, &payload)
	query := discovery.SanitizeQuery(payload.Query)
	if query == "" {
		return &jobError{err: errors.New("query Software Heritage vazia após higienização"), permanent: true}
	}
	results, err := w.swh.SearchOrigins(ctx, query, 10)
	if err != nil {
		return err
	}
	enqueued := 0
	for _, r := range results {
		ok, err := w.store.EnqueueFetch(ctx, job.CampaignID, r.URL, 0, "", "software_heritage", 25)
		if err != nil {
			continue
		}
		if ok {
			enqueued++
		}
	}
	w.logger.Info("descoberta Software Heritage concluída", "query", query, "results", len(results), "enqueued", enqueued)
	return nil
}

// documentByteLimit calcula o teto de um download: o limite do tipo de
// documento (HTML ou PDF), reduzido ao saldo restante do orçamento de bytes da
// campanha (maxBytes - used). O saldo evita que uma campanha com 9 MiB de 10 MiB
// usados baixe mais 5 MiB. Com vários workers simultâneos o excesso fica
// limitado a (workers - 1) documentos, pois a reserva não é atômica.
func documentByteLimit(maxHTML, maxPDF int64, isPDF bool, maxBytes, used int64) int64 {
	limit := maxHTML
	if isPDF {
		limit = maxPDF
	}
	if maxBytes > 0 {
		remaining := maxBytes - used
		if remaining < 1 {
			remaining = 1
		}
		if remaining < limit {
			limit = remaining
		}
	}
	return limit
}

// preserveUnextracted arquiva o documento original no CAS e registra um
// documento sem texto, marcado como extraction_pending, quando a extração falha
// ou não produz texto. O original e seu SHA-256 ficam disponíveis para um
// reprocessamento futuro (OCR/Docling) sem nova coleta.
func (w *Worker) preserveUnextracted(ctx context.Context, job *domain.Job, requested, final string, raw []byte, fr *domain.FetchResult, reason string) error {
	stored, err := w.storage.Put(raw)
	if err != nil {
		return err
	}
	mime := strings.TrimSpace(strings.SplitN(fr.ContentType, ";", 2)[0])
	if mime == "" {
		mime = "application/octet-stream"
	}
	if _, _, err := w.store.RecordDocument(ctx, store.DocumentRecord{
		SHA256: stored.SHA256,
		MIME:   mime,
		Size:   stored.Size,
		Metadata: map[string]any{
			"requested_url":     requested,
			"final_url":         final,
			"executor":          fr.Metadata["executor"],
			"extraction_status": "pending",
			"extraction_reason": reason,
		},
		CampaignID:    job.CampaignID,
		URL:           final,
		NormalizedURL: final,
		Domain:        urlutil.Domain(final),
		StatusCode:    fr.StatusCode,
		ETag:          fr.ETag,
		LastModified:  fr.LastModified,
	}); err != nil {
		return err
	}
	w.logger.Warn("documento preservado sem texto extraído", "url", final, "sha256", stored.SHA256, "motivo", reason)
	return nil
}

func classifyFetchError(err error) error {
	switch {
	case errors.Is(err, crawler.ErrBlocked),
		errors.Is(err, crawler.ErrTooLarge),
		errors.Is(err, crawler.ErrUnsupportedType),
		errors.Is(err, crawler.ErrRobotsDisallowed),
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
		".ttf", ".eot":
		return false
	}
	return true
}

func computeFrontierPriority(link string, depth int, campaignQuery string) int {
	score := 50 + depth*10
	lowerURL := strings.ToLower(link)

	if campaignQuery != "" {
		terms := strings.Fields(strings.ToLower(campaignQuery))
		for _, term := range terms {
			if len(term) >= 3 && strings.Contains(lowerURL, term) {
				score -= 15
			}
		}
	}

	docIndicators := []string{"doc", "manual", "guide", "cli", "command", "config", "spec", "telecom", "firmware", "pdf", "rfc", "support", "architecture", "whitepaper"}
	for _, ind := range docIndicators {
		if strings.Contains(lowerURL, ind) {
			score -= 10
			break
		}
	}

	noiseIndicators := []string{"login", "signin", "signup", "cart", "cookie", "privacy", "terms", "facebook", "twitter", "linkedin", "instagram", "share"}
	for _, noise := range noiseIndicators {
		if strings.Contains(lowerURL, noise) {
			score += 40
			break
		}
	}

	if score < 5 {
		score = 5
	}
	if score > 200 {
		score = 200
	}
	return score
}

// isSearchableMention decide se uma menção sem link vira nova consulta de
// descoberta. Referências genéricas (CVE, DOI, arXiv, normas) valem para
// qualquer assunto; os prefixos de telecom são a especialização inicial.
func isSearchableMention(m string) bool {
	upper := strings.ToUpper(strings.TrimSpace(m))
	if len(upper) < 3 {
		return false
	}
	generic := []string{"CVE-", "ARXIV:", "ISO", "IEC", "IEEE", "ITU-T", "NIST", "ABNT", "10."}
	for _, p := range generic {
		if strings.HasPrefix(upper, p) {
			return true
		}
	}
	// Prefixos de telecom / infraestrutura
	prefixes := []string{
		"RFC", "NE8000", "CCR", "DM4", "AN5516", "AN6000", "MA5800", "MA5608T",
		"OLT", "ASR", "7600", "MX", "PTX", "VRP", "ROUTEROS", "IOS-XR", "JUNOS",
		"BGP", "PPPOE", "VLAN", "MPLS", "OSPF", "FIBERHOME", "HUAWEI", "MIKROTIK", "CISCO", "DATACOM",
		"SONICWALL", "FORTINET", "PALOALTO", "CHECKPOINT", "ZTE", "VSOL", "DELL", "ARUBA", "JUNIPER",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(upper, p) {
			return true
		}
	}
	// Aceita identificadores técnicos alfanuméricos com hífen/versão (ex: CVE-2024, V7.12, RFC-8950, etc.)
	if strings.Contains(upper, "-") || strings.Contains(upper, ".") || strings.Contains(upper, "_") {
		hasDigit := false
		for _, r := range upper {
			if r >= '0' && r <= '9' {
				hasDigit = true
				break
			}
		}
		if hasDigit && len(upper) >= 4 {
			return true
		}
	}
	return false
}
