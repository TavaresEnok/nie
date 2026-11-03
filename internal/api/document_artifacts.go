package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"nie/internal/domain"
	"nie/internal/extract"
	"nie/internal/storage"
	"nie/internal/store"
)

func (s *Server) objectStore() (*storage.Store, error) {
	root := s.cfg.DataDir
	if root == "" {
		root = "./data"
	}
	return storage.New(root)
}

func registerOriginalDownload(mux *http.ServeMux, s *Server) {
	mux.HandleFunc("GET /api/documents/{id}/original", func(w http.ResponseWriter, r *http.Request) {
		doc, err := s.store.GetDocument(r.Context(), r.PathValue("id"))
		if err != nil {
			http.Error(w, "documento não encontrado", http.StatusNotFound)
			return
		}
		objects, err := s.objectStore()
		if err != nil {
			s.logger.Error("abrir CAS", "err", err)
			http.Error(w, "acervo indisponível", 500)
			return
		}
		f, err := objects.OpenVerified(doc.SHA256)
		if err != nil {
			s.logger.Error("verificar original", "document", doc.ID, "err", err)
			http.Error(w, "original indisponível ou sem integridade comprovada", 500)
			return
		}
		defer f.Close()
		stat, err := f.Stat()
		if err != nil || stat.Size() != doc.Size {
			http.Error(w, "tamanho do original diverge do catálogo", 500)
			return
		}
		extension := ".bin"
		if doc.MIME == "application/pdf" {
			extension = ".pdf"
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s%s"`, doc.SHA256, extension))
		w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'")
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("ETag", `"`+doc.SHA256+`"`)
		http.ServeContent(w, r, doc.SHA256, doc.CreatedAt, f)
	})
}

func registerExtractions(api huma.API, s *Server) {
	type documentParam struct {
		ID string `path:"id"`
	}
	type runParam struct {
		ID    string `path:"id"`
		RunID string `path:"run_id"`
	}
	type reprocessParam struct {
		ID     string `path:"id"`
		Parser string `query:"parser" default:"native" enum:"native,ocr"`
	}
	type runsOut struct{ Body []domain.ExtractionRun }
	type runOut struct{ Body domain.ExtractionRun }
	huma.Register(api, huma.Operation{OperationID: "list-extractions", Method: http.MethodGet, Path: "/api/documents/{id}/extractions", Summary: "Histórico de extrações do original"}, func(ctx context.Context, in *documentParam) (*runsOut, error) {
		if _, err := s.store.GetDocument(ctx, in.ID); err != nil {
			return nil, s.mapError(err)
		}
		runs, err := s.store.ListExtractions(ctx, in.ID)
		if err != nil {
			return nil, s.mapError(err)
		}
		return &runsOut{Body: runs}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "get-extraction", Method: http.MethodGet, Path: "/api/documents/{id}/extractions/{run_id}", Summary: "Texto e trechos imutáveis de uma extração"}, func(ctx context.Context, in *runParam) (*runOut, error) {
		run, err := s.store.GetExtraction(ctx, in.ID, in.RunID)
		if err != nil {
			return nil, s.mapError(err)
		}
		return &runOut{Body: run}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "reprocess-document", Method: http.MethodPost, Path: "/api/documents/{id}/reprocess", Summary: "Reextrair o original preservado com parser nativo ou OCR opcional"}, func(ctx context.Context, in *reprocessParam) (*runOut, error) {
		doc, err := s.store.GetDocument(ctx, in.ID)
		if err != nil {
			return nil, s.mapError(err)
		}
		limit := s.cfg.MaxPDFBytes
		if limit <= 0 {
			limit = 50 << 20
		}
		if doc.Size > limit {
			return nil, huma.Error413RequestEntityTooLarge("original excede o limite de reprocessamento")
		}
		objects, err := s.objectStore()
		if err != nil {
			return nil, s.internal(err)
		}
		f, err := objects.OpenVerified(doc.SHA256)
		if err != nil {
			return nil, s.internal(err)
		}
		defer f.Close()
		body, err := io.ReadAll(io.LimitReader(f, limit+1))
		if err != nil {
			return nil, s.internal(err)
		}
		if int64(len(body)) != doc.Size {
			return nil, huma.Error422UnprocessableEntity("tamanho do original diverge do catálogo")
		}
		var base *url.URL
		if len(doc.Versions) > 0 {
			base, _ = url.Parse(doc.Versions[0].URL)
		}
		var result extract.Result
		parserConfig := map[string]any{"input": "original"}
		if base != nil {
			parserConfig["base_url"] = base.String()
		}
		if in.Parser == "ocr" {
			if !s.cfg.OCREnabled {
				return nil, huma.NewError(http.StatusServiceUnavailable, "OCR desabilitado nesta instalação")
			}
			if doc.MIME != "application/pdf" {
				return nil, huma.Error422UnprocessableEntity("OCR disponível para PDF nesta rota")
			}
			ocr := extract.NewOCRServiceWithConfig(extract.OCRConfig{MaxPages: s.cfg.OCRMaxPages, Language: s.cfg.OCRLanguage, Timeout: s.cfg.OCRProcessingTimeout()})
			if !ocr.CanOCRPDF() {
				return nil, huma.NewError(http.StatusServiceUnavailable, "instale Tesseract e Poppler para OCR PDF")
			}
			result, err = ocr.ExtractBytes(ctx, body, doc.MIME)
			settings := ocr.Settings()
			parserConfig["ocr_language"] = settings.Language
			parserConfig["ocr_max_pages"] = settings.MaxPages
			parserConfig["ocr_max_dimension"] = settings.MaxDimension
		} else {
			switch {
			case doc.MIME == "application/pdf":
				result, err = extract.PDF(body)
			case doc.MIME == "text/html" || doc.MIME == "application/xhtml+xml":
				contentType, _ := doc.Metadata["content_type"].(string)
				if contentType == "" {
					contentType = "text/html"
				}
				result, err = extract.HTML(base, contentType, body)
			case strings.Contains(doc.MIME, "xml"):
				result, err = extract.Feed(base, body)
			default:
				result, err = extract.Text(body)
			}
		}
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("extração falhou; original preservado: " + err.Error())
		}
		passages := make([]domain.Passage, len(result.Passages))
		for i, p := range result.Passages {
			passages[i] = domain.Passage{Ordinal: p.Ordinal, Page: p.Page, Text: p.Text}
		}
		if strings.TrimSpace(result.Text) == "" || len(passages) == 0 {
			return nil, huma.Error422UnprocessableEntity("nenhum texto extraível; original preservado")
		}
		runID, err := s.store.RecordExtraction(ctx, doc.ID, store.ExtractionRecord{Parser: result.Parser, ParserVersion: result.ParserVersion, InputSHA256: doc.SHA256, Config: parserConfig, Title: result.Title, Text: result.Text, Passages: passages})
		if err != nil {
			return nil, s.internal(err)
		}
		run, err := s.store.GetExtraction(ctx, doc.ID, runID)
		if err != nil {
			return nil, s.internal(err)
		}
		return &runOut{Body: run}, nil
	})
}
