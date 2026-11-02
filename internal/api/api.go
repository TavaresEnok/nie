package api

import (
	"context"
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"nie/internal/config"
	"nie/internal/domain"
	"nie/internal/store"
	"nie/internal/urlutil"
	"nie/internal/version"
	"nie/internal/webui"
)

type Server struct {
	store   *store.Store
	cfg     config.Config
	logger  *slog.Logger
	started time.Time
}

func New(st *store.Store, cfg config.Config, logger *slog.Logger) http.Handler {
	s := &Server{store: st, cfg: cfg, logger: logger, started: time.Now()}
	mux := http.NewServeMux()
	hcfg := huma.DefaultConfig("Network Intelligence Engine API", version.Version)
	hcfg.Info.Description = "Motor de descoberta, coleta e busca técnica com evidências citáveis."
	hcfg.OpenAPIPath = "/api/openapi"
	hcfg.DocsPath = "/api/docs"
	hcfg.SchemasPath = "/api/schemas"
	hcfg.OpenAPI.Servers = []*huma.Server{{URL: "/"}}
	api := humago.New(mux, hcfg)

	registerHealth(api, s)
	registerMeta(api, s)
	registerStats(api, s)
	registerCampaigns(api, s)
	registerSearch(api, s)
	registerDocuments(api, s)
	registerJobs(api, s)

	mux.Handle("/", webui.Handler(cfg.WebDir))
	return s.middleware(mux)
}

func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; font-src 'self' data:; base-uri 'self'; frame-ancestors 'none'")
		}
		if !s.authorized(r) {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"title":"unauthorized","status":401,"detail":"informe o token da API"}`))
			return
		}
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		defer func() {
			if recovered := recover(); recovered != nil {
				s.logger.Error("panic na API", "path", r.URL.Path, "panic", recovered)
				http.Error(rec, `{"title":"internal error","status":500}`, http.StatusInternalServerError)
			}
			s.logger.Info("http", "method", r.Method, "path", r.URL.Path, "status", rec.status, "duration", time.Since(start).Round(time.Millisecond))
		}()
		next.ServeHTTP(rec, r)
	})
}

func (s *Server) authorized(r *http.Request) bool {
	if s.cfg.APIToken == "" {
		return true
	}
	switch r.URL.Path {
	case "/api/health", "/api/meta", "/api/docs", "/api/openapi", "/api/openapi.json", "/api/openapi.yaml":
		return true
	}
	if strings.HasPrefix(r.URL.Path, "/api/schemas/") {
		return true
	}
	if !strings.HasPrefix(r.URL.Path, "/api/") {
		return true
	}
	got := r.Header.Get("X-API-Token")
	if auth := r.Header.Get("Authorization"); got == "" && strings.HasPrefix(auth, "Bearer ") {
		got = strings.TrimPrefix(auth, "Bearer ")
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(s.cfg.APIToken)) == 1
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func registerHealth(api huma.API, s *Server) {
	type out struct {
		Body struct {
			Status  string `json:"status"`
			Version string `json:"version"`
			Uptime  int64  `json:"uptime_seconds"`
			DB      string `json:"db"`
		}
	}
	huma.Register(api, huma.Operation{
		OperationID: "health", Method: http.MethodGet, Path: "/api/health", Summary: "Healthcheck",
	}, func(ctx context.Context, _ *struct{}) (*out, error) {
		o := &out{}
		o.Body.Status = "ok"
		o.Body.Version = version.Version
		o.Body.Uptime = int64(time.Since(s.started).Seconds())
		o.Body.DB = "ok"
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if err := s.store.Ping(pingCtx); err != nil {
			o.Body.Status = "degraded"
			o.Body.DB = "unavailable"
			return o, huma.Error503ServiceUnavailable("banco indisponível")
		}
		return o, nil
	})
}

func registerMeta(api huma.API, s *Server) {
	type out struct {
		Body struct {
			Version      string `json:"version"`
			WebDiscovery bool   `json:"web_discovery"`
			GitDiscovery bool   `json:"git_discovery"`
			AuthRequired bool   `json:"auth_required"`
		}
	}
	huma.Register(api, huma.Operation{
		OperationID: "meta", Method: http.MethodGet, Path: "/api/meta", Summary: "Capacidades do servidor",
	}, func(ctx context.Context, _ *struct{}) (*out, error) {
		o := &out{}
		o.Body.Version = version.Version
		o.Body.WebDiscovery = s.cfg.SearXNGURL != ""
		o.Body.GitDiscovery = s.cfg.GitHubEnabled
		o.Body.AuthRequired = s.cfg.APIToken != ""
		return o, nil
	})
}

func registerStats(api huma.API, s *Server) {
	type out struct {
		Body domain.Stats
	}
	huma.Register(api, huma.Operation{
		OperationID: "stats", Method: http.MethodGet, Path: "/api/stats", Summary: "Estatísticas gerais",
	}, func(ctx context.Context, _ *struct{}) (*out, error) {
		st, err := s.store.Stats(ctx)
		if err != nil {
			return nil, s.internal(err)
		}
		st.Version = version.Version
		st.StartedAt = s.started
		st.UptimeSecs = int64(time.Since(s.started).Seconds())
		return &out{Body: st}, nil
	})
}

type campaignInput struct {
	Body struct {
		Query           string   `json:"query" required:"true" minLength:"3" maxLength:"500"`
		SeedURLs        []string `json:"seed_urls,omitempty" maxItems:"20"`
		MaxDepth        int      `json:"max_depth,omitempty"`
		MaxURLs         int      `json:"max_urls,omitempty"`
		MaxDomains      int      `json:"max_domains,omitempty"`
		MaxBytes        int64    `json:"max_bytes,omitempty"`
		DomainInterval  int      `json:"min_domain_interval_seconds,omitempty"`
		UseWebDiscovery *bool    `json:"use_web_discovery,omitempty"`
		UseGitDiscovery *bool    `json:"use_git_discovery,omitempty"`
	}
}

type campaignOut struct {
	Body domain.Campaign
}

type campaignListOut struct {
	Body struct {
		Items []domain.Campaign `json:"items"`
		Count int               `json:"count"`
	}
}

func registerCampaigns(api huma.API, s *Server) {
	huma.Register(api, huma.Operation{
		OperationID: "create-campaign", Method: http.MethodPost, Path: "/api/campaigns",
		Summary: "Criar campanha de investigação", DefaultStatus: http.StatusCreated,
		Errors: []int{400, 500},
	}, func(ctx context.Context, in *campaignInput) (*campaignOut, error) {
		query := strings.TrimSpace(in.Body.Query)
		if len(query) < 3 {
			return nil, huma.Error400BadRequest("a pergunta deve ter ao menos 3 caracteres")
		}
		b := domain.Budget{
			MaxDepth:                 in.Body.MaxDepth,
			MaxURLs:                  in.Body.MaxURLs,
			MaxDomains:               in.Body.MaxDomains,
			MaxBytes:                 in.Body.MaxBytes,
			MinDomainIntervalSeconds: in.Body.DomainInterval,
		}
		if b.MaxDepth < 0 {
			b.MaxDepth = 0
		}
		if b.MaxDepth == 0 {
			b.MaxDepth = 3
		}
		if b.MaxDepth > 10 {
			return nil, huma.Error400BadRequest("max_depth deve ser <= 10")
		}
		if b.MaxURLs == 0 {
			b.MaxURLs = 200
		}
		if b.MaxURLs < 1 || b.MaxURLs > 5000 {
			return nil, huma.Error400BadRequest("max_urls deve estar entre 1 e 5000")
		}
		if b.MaxDomains == 0 {
			b.MaxDomains = 20
		}
		if b.MaxDomains < 1 || b.MaxDomains > 500 {
			return nil, huma.Error400BadRequest("max_domains deve estar entre 1 e 500")
		}
		if b.MaxBytes == 0 {
			b.MaxBytes = 1 << 30
		}
		if b.MaxBytes < 1<<20 || b.MaxBytes > 10<<30 {
			return nil, huma.Error400BadRequest("max_bytes deve estar entre 1 MiB e 10 GiB")
		}
		if b.MinDomainIntervalSeconds == 0 {
			b.MinDomainIntervalSeconds = int(s.cfg.DomainInterval().Seconds())
		}
		if b.MinDomainIntervalSeconds < 1 || b.MinDomainIntervalSeconds > 3600 {
			return nil, huma.Error400BadRequest("min_domain_interval_seconds deve estar entre 1 e 3600")
		}

		seeds := make([]string, 0, len(in.Body.SeedURLs))
		for _, raw := range in.Body.SeedURLs {
			raw = strings.TrimSpace(raw)
			if raw == "" {
				continue
			}
			norm, err := urlutil.Normalize(raw)
			if err != nil {
				return nil, huma.Error400BadRequest("seed inválida: " + raw + " (" + err.Error() + ")")
			}
			seeds = append(seeds, norm)
		}

		web := s.cfg.SearXNGURL != ""
		if in.Body.UseWebDiscovery != nil {
			web = *in.Body.UseWebDiscovery
		}
		git := s.cfg.GitHubEnabled
		if in.Body.UseGitDiscovery != nil {
			git = *in.Body.UseGitDiscovery
		}
		if web && s.cfg.SearXNGURL == "" {
			return nil, huma.Error400BadRequest("descoberta web solicitada, mas NIE_SEARXNG_URL não está configurada")
		}
		if git && !s.cfg.GitHubEnabled {
			return nil, huma.Error400BadRequest("descoberta GitHub solicitada, mas o adaptador está desabilitado")
		}
		if len(seeds) == 0 && !web && !git {
			return nil, huma.Error400BadRequest("informe ao menos uma seed ou habilite uma descoberta externa")
		}

		c, err := s.store.CreateCampaign(ctx, query, seeds, b)
		if err != nil {
			return nil, huma.Error400BadRequest("não foi possível criar a campanha: " + err.Error())
		}
		if web {
			if _, err := s.store.EnqueueDiscovery(ctx, c.ID, "discover_web", query); err != nil {
				s.logger.Error("enfileirar descoberta web", "campaign", c.ID, "err", err)
			}
		}
		if git {
			if _, err := s.store.EnqueueDiscovery(ctx, c.ID, "discover_github", query); err != nil {
				s.logger.Error("enfileirar descoberta github", "campaign", c.ID, "err", err)
			}
		}
		_ = s.store.FinishIfIdle(ctx, c.ID)
		fresh, err := s.store.GetCampaign(ctx, c.ID)
		if err == nil {
			c = fresh
		}
		return &campaignOut{Body: c}, nil
	})

	type listIn struct {
		Limit int `query:"limit" default:"50" minimum:"1" maximum:"200"`
	}
	huma.Register(api, huma.Operation{
		OperationID: "list-campaigns", Method: http.MethodGet, Path: "/api/campaigns", Summary: "Listar campanhas",
	}, func(ctx context.Context, in *listIn) (*campaignListOut, error) {
		items, err := s.store.ListCampaigns(ctx, in.Limit)
		if err != nil {
			return nil, s.internal(err)
		}
		o := &campaignListOut{}
		o.Body.Items = items
		o.Body.Count = len(items)
		return o, nil
	})

	type getIn struct {
		ID string `path:"id" required:"true"`
	}
	huma.Register(api, huma.Operation{
		OperationID: "get-campaign", Method: http.MethodGet, Path: "/api/campaigns/{id}", Summary: "Detalhar campanha",
	}, func(ctx context.Context, in *getIn) (*campaignOut, error) {
		c, err := s.store.GetCampaign(ctx, in.ID)
		if err != nil {
			return nil, s.mapError(err)
		}
		return &campaignOut{Body: c}, nil
	})

	type actionOut struct {
		Body struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		}
	}
	huma.Register(api, huma.Operation{
		OperationID: "pause-campaign", Method: http.MethodPost, Path: "/api/campaigns/{id}/pause", Summary: "Pausar campanha",
	}, func(ctx context.Context, in *getIn) (*actionOut, error) {
		if err := s.store.SetCampaignStatus(ctx, in.ID, "paused"); err != nil {
			return nil, s.mapError(err)
		}
		return &actionOut{Body: struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		}{ID: in.ID, Status: "paused"}}, nil
	})
	huma.Register(api, huma.Operation{
		OperationID: "resume-campaign", Method: http.MethodPost, Path: "/api/campaigns/{id}/resume", Summary: "Retomar campanha",
	}, func(ctx context.Context, in *getIn) (*actionOut, error) {
		if err := s.store.SetCampaignStatus(ctx, in.ID, "running"); err != nil {
			return nil, s.mapError(err)
		}
		return &actionOut{Body: struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		}{ID: in.ID, Status: "running"}}, nil
	})

	type edgesOut struct {
		Body struct {
			Items []domain.DiscoveryEdge `json:"items"`
			Count int                    `json:"count"`
		}
	}
	type edgesIn struct {
		ID    string `path:"id" required:"true"`
		Limit int    `query:"limit" default:"500" minimum:"1" maximum:"2000"`
	}
	huma.Register(api, huma.Operation{
		OperationID: "campaign-edges", Method: http.MethodGet, Path: "/api/campaigns/{id}/edges", Summary: "Grafo de descobertas",
	}, func(ctx context.Context, in *edgesIn) (*edgesOut, error) {
		items, err := s.store.ListEdges(ctx, in.ID, in.Limit)
		if err != nil {
			return nil, s.mapError(err)
		}
		o := &edgesOut{}
		o.Body.Items = items
		o.Body.Count = len(items)
		return o, nil
	})

	type dossierOut struct {
		Body domain.Dossier
	}
	huma.Register(api, huma.Operation{
		OperationID: "campaign-dossier", Method: http.MethodGet, Path: "/api/campaigns/{id}/dossier", Summary: "Gerar dossiê técnico da campanha",
	}, func(ctx context.Context, in *getIn) (*dossierOut, error) {
		d, err := s.store.GenerateCampaignDossier(ctx, in.ID)
		if err != nil {
			return nil, s.mapError(err)
		}
		return &dossierOut{Body: d}, nil
	})
}

func registerSearch(api huma.API, s *Server) {
	type in struct {
		Query      string `query:"q" required:"true" minLength:"2" maxLength:"500"`
		CampaignID string `query:"campaign_id" maxLength:"36"`
		Limit      int    `query:"limit" default:"20" minimum:"1" maximum:"100"`
	}
	type out struct {
		Body struct {
			Query string                `json:"query"`
			Items []domain.SearchResult `json:"items"`
			Count int                   `json:"count"`
		}
	}
	huma.Register(api, huma.Operation{
		OperationID: "search", Method: http.MethodGet, Path: "/api/search", Summary: "Busca exata e lexical",
	}, func(ctx context.Context, in *in) (*out, error) {
		items, err := s.store.Search(ctx, in.Query, in.CampaignID, in.Limit)
		if err != nil {
			return nil, s.mapError(err)
		}
		o := &out{}
		o.Body.Query = in.Query
		o.Body.Items = items
		o.Body.Count = len(items)
		return o, nil
	})
}

func registerDocuments(api huma.API, s *Server) {
	type listIn struct {
		Limit      int    `query:"limit" default:"50" minimum:"1" maximum:"200"`
		CampaignID string `query:"campaign_id" maxLength:"36"`
	}
	type listOut struct {
		Body struct {
			Items []domain.DocumentSummary `json:"items"`
			Count int                      `json:"count"`
		}
	}
	huma.Register(api, huma.Operation{
		OperationID: "list-documents", Method: http.MethodGet, Path: "/api/documents", Summary: "Biblioteca de documentos",
	}, func(ctx context.Context, in *listIn) (*listOut, error) {
		items, err := s.store.ListDocuments(ctx, in.Limit, in.CampaignID)
		if err != nil {
			return nil, s.mapError(err)
		}
		o := &listOut{}
		o.Body.Items = items
		o.Body.Count = len(items)
		return o, nil
	})

	type getIn struct {
		ID string `path:"id" required:"true"`
	}
	type getOut struct {
		Body domain.Document
	}
	huma.Register(api, huma.Operation{
		OperationID: "get-document", Method: http.MethodGet, Path: "/api/documents/{id}", Summary: "Documento e trechos citáveis",
	}, func(ctx context.Context, in *getIn) (*getOut, error) {
		d, err := s.store.GetDocument(ctx, in.ID)
		if err != nil {
			return nil, s.mapError(err)
		}
		return &getOut{Body: d}, nil
	})
}

func registerJobs(api huma.API, s *Server) {
	type in struct {
		CampaignID string `query:"campaign_id" maxLength:"36"`
		Status     string `query:"status" maxLength:"20"`
		Limit      int    `query:"limit" default:"100" minimum:"1" maximum:"500"`
	}
	type out struct {
		Body struct {
			Items []domain.Job `json:"items"`
			Count int          `json:"count"`
		}
	}
	huma.Register(api, huma.Operation{
		OperationID: "list-jobs", Method: http.MethodGet, Path: "/api/jobs", Summary: "Fila de trabalho",
	}, func(ctx context.Context, in *in) (*out, error) {
		switch in.Status {
		case "", "pending", "running", "done", "dead":
		default:
			return nil, huma.Error400BadRequest("status inválido")
		}
		items, err := s.store.ListJobs(ctx, in.CampaignID, in.Status, in.Limit)
		if err != nil {
			return nil, s.mapError(err)
		}
		o := &out{}
		o.Body.Items = items
		o.Body.Count = len(items)
		return o, nil
	})
}

func (s *Server) internal(err error) error {
	s.logger.Error("erro interno", "err", err)
	return huma.Error500InternalServerError("erro interno")
}

func (s *Server) mapError(err error) error {
	if errors.Is(err, store.ErrNotFound) {
		return huma.Error404NotFound("não encontrado")
	}
	return s.internal(err)
}
