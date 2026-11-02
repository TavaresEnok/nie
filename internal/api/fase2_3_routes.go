package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"nie/internal/discovery"
	"nie/internal/domain"
)

type vendorsOut struct {
	Body struct {
		Vendors []domain.Vendor `json:"vendors"`
	}
}

type claimsOut struct {
	Body struct {
		Claims []domain.Claim `json:"claims"`
	}
}

type campaignParam struct {
	ID string `path:"id" doc:"ID da campanha"`
}

type graphOut struct {
	Body domain.DiscoveryGraph
}

type deepDossierOut struct {
	Body domain.InvestigationDossier
}

type scheduleInput struct {
	Body struct {
		Name            string   `json:"name" minLength:"3" example:"Monitoramento BGP NE8000"`
		Query           string   `json:"query" minLength:"3" example:"Huawei NE8000 BGP BNG"`
		SeedURLs        []string `json:"seed_urls" example:"[\"https://support.huawei.com\"]"`
		IntervalSeconds int      `json:"interval_seconds" example:"86400"`
		MaxDepth        int      `json:"max_depth" example:"3"`
		MaxURLs         int      `json:"max_urls" example:"100"`
		MaxDomains      int      `json:"max_domains" example:"10"`
	}
}

type scheduleOut struct {
	Body domain.Schedule
}

type schedulesListOut struct {
	Body struct {
		Schedules []domain.Schedule `json:"schedules"`
	}
}

type alertsListOut struct {
	Body struct {
		Alerts []domain.Alert `json:"alerts"`
	}
}

type alertAckParam struct {
	ID string `path:"id" doc:"ID do alerta"`
}

type gitSearchInput struct {
	Query string `query:"q" minLength:"2" doc:"Termo para busca em repositórios Git locais"`
	Limit int    `query:"limit" doc:"Máximo de resultados" default:"20"`
}

type gitSearchOut struct {
	Body struct {
		Results []discovery.LocalFileMatch `json:"results"`
	}
}

func registerFase2and3(api huma.API, s *Server) {
	// Vendors
	huma.Register(api, huma.Operation{
		OperationID: "list-vendors", Method: http.MethodGet, Path: "/api/vendors",
		Summary: "Listar catálogo de fabricantes de telecom",
	}, func(ctx context.Context, in *struct{}) (*vendorsOut, error) {
		vendors, err := s.store.ListVendors(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}
		res := &vendorsOut{}
		res.Body.Vendors = vendors
		return res, nil
	})

	// Claims by campaign
	huma.Register(api, huma.Operation{
		OperationID: "list-campaign-claims", Method: http.MethodGet, Path: "/api/campaigns/{id}/claims",
		Summary: "Listar claims e evidências independentes da campanha",
	}, func(ctx context.Context, in *campaignParam) (*claimsOut, error) {
		claims, err := s.store.ListClaimsByCampaign(ctx, in.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}
		res := &claimsOut{}
		res.Body.Claims = claims
		return res, nil
	})

	// Run Deep Investigation & AI Dossier
	huma.Register(api, huma.Operation{
		OperationID: "investigate-campaign", Method: http.MethodPost, Path: "/api/campaigns/{id}/investigate",
		Summary: "Executar motor de evidências, sintetizar claims e gerar Dossiê Técnico",
	}, func(ctx context.Context, in *campaignParam) (*deepDossierOut, error) {
		camp, err := s.store.GetCampaign(ctx, in.ID)
		if err != nil {
			return nil, huma.Error404NotFound("campanha não encontrada")
		}

		// Buscar passagens da campanha para evidência
		passages, err := s.store.Search(ctx, camp.Query, camp.ID, 100)
		if err != nil {
			return nil, huma.Error500InternalServerError("erro ao recuperar passagens: " + err.Error())
		}

		// Construir claims e origin clusters
		claims := s.evidenceEng.BuildClaimsFromPassages(camp.ID, camp.Query, passages)
		if len(claims) > 0 {
			if err := s.store.SaveClaims(ctx, claims); err != nil {
				return nil, huma.Error500InternalServerError("erro ao persistir claims e evidências: " + err.Error())
			}
		}

		// Gerar Dossiê de Investigação
		dossier := s.aiSvc.GenerateInvestigationDossier(ctx, camp, claims, passages)

		// P1-06: Fechamento do ciclo autônomo recursivo de investigação.
		// Enfileira as pistas investigativas adaptativas (AdaptiveInvestigationLeads)
		// geradas para claims incompletas (contested, documented, unconfirmed) como
		// tarefas de descoberta de 2ª rodada na fila de jobs persistente PostgreSQL.
		for _, lead := range dossier.InvestigationLeads {
			if lead.SearchLead != "" {
				if s.cfg.SearXNGURL != "" {
					_, _ = s.store.EnqueueDiscovery(ctx, camp.ID, "discover_web", lead.SearchLead)
				}
				if s.cfg.MwmblURL != "" {
					_, _ = s.store.EnqueueDiscovery(ctx, camp.ID, "discover_mwmbl", lead.SearchLead)
				}
				if s.cfg.GitHubEnabled {
					_, _ = s.store.EnqueueDiscovery(ctx, camp.ID, "discover_github", lead.SearchLead)
				}
				if s.cfg.SoftwareHeritageEnabled {
					_, _ = s.store.EnqueueDiscovery(ctx, camp.ID, "discover_swh", lead.SearchLead)
				}
			}
		}

		res := &deepDossierOut{Body: dossier}
		return res, nil
	})

	// Discovery Graph
	huma.Register(api, huma.Operation{
		OperationID: "get-campaign-graph", Method: http.MethodGet, Path: "/api/campaigns/{id}/graph",
		Summary: "Obter Grafo Visual de Descoberta (nós e ramificações)",
	}, func(ctx context.Context, in *campaignParam) (*graphOut, error) {
		graph, err := s.store.GetDiscoveryGraph(ctx, in.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("erro ao carregar grafo: " + err.Error())
		}
		return &graphOut{Body: graph}, nil
	})

	// Schedules (Autonomia)
	huma.Register(api, huma.Operation{
		OperationID: "create-schedule", Method: http.MethodPost, Path: "/api/schedules",
		Summary: "Criar agendamento para investigação contínua e autônoma", DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *scheduleInput) (*scheduleOut, error) {
		if in.Body.IntervalSeconds <= 0 {
			in.Body.IntervalSeconds = 86400
		}
		if in.Body.MaxDepth <= 0 {
			in.Body.MaxDepth = 3
		}
		if in.Body.MaxURLs <= 0 {
			in.Body.MaxURLs = 100
		}
		if in.Body.MaxDomains <= 0 {
			in.Body.MaxDomains = 10
		}
		sch, err := s.store.CreateSchedule(ctx, in.Body.Name, in.Body.Query, in.Body.SeedURLs, in.Body.IntervalSeconds, in.Body.MaxDepth, in.Body.MaxURLs, in.Body.MaxDomains)
		if err != nil {
			return nil, huma.Error500InternalServerError("erro ao criar agendamento: " + err.Error())
		}
		return &scheduleOut{Body: sch}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-schedules", Method: http.MethodGet, Path: "/api/schedules",
		Summary: "Listar agendamentos de autonomia contínua",
	}, func(ctx context.Context, in *struct{}) (*schedulesListOut, error) {
		list, err := s.store.ListSchedules(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}
		res := &schedulesListOut{}
		res.Body.Schedules = list
		return res, nil
	})

	// Alerts
	huma.Register(api, huma.Operation{
		OperationID: "list-alerts", Method: http.MethodGet, Path: "/api/alerts",
		Summary: "Listar alertas técnicos e mudanças em documentações",
	}, func(ctx context.Context, in *struct{}) (*alertsListOut, error) {
		alerts, err := s.store.ListAlerts(ctx, 50)
		if err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}
		res := &alertsListOut{}
		res.Body.Alerts = alerts
		return res, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "ack-alert", Method: http.MethodPost, Path: "/api/alerts/{id}/ack",
		Summary: "Confirmar ciência de alerta",
	}, func(ctx context.Context, in *alertAckParam) (*struct{}, error) {
		if err := s.store.AcknowledgeAlert(ctx, in.ID); err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}
		return &struct{}{}, nil
	})

	// Git Local Code Search
	huma.Register(api, huma.Operation{
		OperationID: "search-git-local", Method: http.MethodGet, Path: "/api/search/git",
		Summary: "Pesquisar arquivos e configurações em repositórios Git locais",
	}, func(ctx context.Context, in *gitSearchInput) (*gitSearchOut, error) {
		q := strings.TrimSpace(in.Query)
		matches, err := s.gitLocal.Search(ctx, q, in.Limit)
		if err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}
		res := &gitSearchOut{}
		res.Body.Results = matches
		return res, nil
	})
}
