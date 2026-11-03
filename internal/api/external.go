package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"nie/internal/archive"
	"nie/internal/discovery"
)

func registerExternal(api huma.API, s *Server) {
	huma.Register(api, huma.Operation{OperationID: "search-zoekt", Method: http.MethodGet, Path: "/api/search/zoekt", Summary: "Busca num índice Zoekt já construído"}, func(ctx context.Context, in *struct {
		Query string `query:"q" minLength:"2"`
		Limit int    `query:"limit" default:"20" minimum:"1" maximum:"50"`
	}) (*struct {
		Body struct {
			Results []discovery.CodeHit `json:"results"`
		}
	}, error) {
		client := discovery.NewZoekt(s.cfg.ZoektURL)
		if client == nil {
			return nil, huma.Error400BadRequest("NIE_ZOEKT_URL não está configurado")
		}
		hits, err := client.Search(ctx, in.Query, in.Limit)
		if err != nil {
			return nil, huma.Error502BadGateway(err.Error())
		}
		out := &struct {
			Body struct {
				Results []discovery.CodeHit `json:"results"`
			}
		}{}
		out.Body.Results = hits
		return out, nil
	})
	huma.Register(api, huma.Operation{OperationID: "search-opensearch", Method: http.MethodGet, Path: "/api/search/opensearch", Summary: "Consulta um índice OpenSearch externo sem substituir o catálogo"}, func(ctx context.Context, in *struct {
		Query string `query:"q" minLength:"2"`
		Limit int    `query:"limit" default:"20" minimum:"1" maximum:"50"`
	}) (*struct {
		Body struct {
			Results []discovery.ExternalHit `json:"results"`
		}
	}, error) {
		client := discovery.NewOpenSearch(s.cfg.OpenSearchURL, s.cfg.OpenSearchIndex, s.cfg.OpenSearchUser, s.cfg.OpenSearchPassword)
		if client == nil {
			return nil, huma.Error400BadRequest("NIE_OPENSEARCH_URL não está configurado")
		}
		hits, err := client.Search(ctx, in.Query, in.Limit)
		if err != nil {
			return nil, huma.Error502BadGateway(err.Error())
		}
		out := &struct {
			Body struct {
				Results []discovery.ExternalHit `json:"results"`
			}
		}{}
		out.Body.Results = hits
		return out, nil
	})
	huma.Register(api, huma.Operation{OperationID: "git-remote-file", Method: http.MethodGet, Path: "/api/git/remote-file", Summary: "Lê um arquivo remoto em um ref do GitHub ou do GitLab"}, func(ctx context.Context, in *struct {
		Host string `query:"host" enum:"github,gitlab"`
		Repo string `query:"repo" minLength:"1"`
		Path string `query:"path" minLength:"1"`
		Ref  string `query:"ref" minLength:"1"`
	}) (*struct{ Body discovery.RemoteFile }, error) {
		var file discovery.RemoteFile
		var err error
		switch in.Host {
		case "github":
			file, err = discovery.NewGitHub(s.cfg.GitHubToken, s.cfg.GitHubEnabled).File(ctx, in.Repo, in.Path, in.Ref)
		case "gitlab":
			file, err = discovery.NewGitLab(s.cfg.GitLabURL, s.cfg.GitLabToken, s.cfg.GitLabEnabled).File(ctx, in.Repo, in.Path, in.Ref)
		default:
			return nil, huma.Error400BadRequest("host deve ser github ou gitlab")
		}
		if err != nil {
			return nil, huma.Error502BadGateway(err.Error())
		}
		return &struct{ Body discovery.RemoteFile }{Body: file}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "archive-url", Method: http.MethodPost, Path: "/api/archive", Summary: "Pede uma captura seletiva ao ArchiveBox"}, func(ctx context.Context, in *struct {
		Body struct {
			URL string `json:"url" format:"uri"`
		}
	}) (*struct {
		Body struct {
			Status string `json:"status"`
		}
	}, error) {
		if err := archive.Add(ctx, s.cfg.ArchiveBoxDir, in.Body.URL); err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}
		out := &struct {
			Body struct {
				Status string `json:"status"`
			}
		}{}
		out.Body.Status = "archived"
		return out, nil
	})
	huma.Register(api, huma.Operation{OperationID: "external-report", Method: http.MethodPost, Path: "/api/research/external-report", Summary: "Pede um relatório ao GPT Researcher sem certificá-lo como evidência"}, func(ctx context.Context, in *struct {
		Body struct {
			Query string `json:"query" minLength:"3" maxLength:"500"`
		}
	}) (*struct {
		Body struct {
			Report string `json:"report"`
			Notice string `json:"notice"`
		}
	}, error) {
		client := discovery.NewResearcher(s.cfg.GPTResearcherURL)
		if client == nil {
			return nil, huma.Error400BadRequest("NIE_GPT_RESEARCHER_URL não está configurado")
		}
		report, err := client.Report(ctx, in.Body.Query)
		if err != nil {
			return nil, huma.Error502BadGateway(err.Error())
		}
		out := &struct {
			Body struct {
				Report string `json:"report"`
				Notice string `json:"notice"`
			}
		}{}
		out.Body.Report = report
		out.Body.Notice = "Relatório externo. Não é claim, não comprova firmware e não entra sozinho na biblioteca."
		return out, nil
	})
}
