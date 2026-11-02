package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"nie/internal/domain"
)

// GenerateCampaignDossier constrói um dossiê técnico com evidências citáveis em formato Markdown.
func (s *Store) GenerateCampaignDossier(ctx context.Context, campaignID string) (domain.Dossier, error) {
	c, err := s.GetCampaign(ctx, campaignID)
	if err != nil {
		return domain.Dossier{}, err
	}

	docs, err := s.ListDocuments(ctx, 500, campaignID)
	if err != nil {
		return domain.Dossier{}, fmt.Errorf("listar documentos: %w", err)
	}

	edges, err := s.ListEdges(ctx, campaignID, 1000)
	if err != nil {
		return domain.Dossier{}, fmt.Errorf("listar arestas: %w", err)
	}

	var totalPassages int
	var totalBytes int64
	domainSet := make(map[string]struct{})

	type docDetail struct {
		summary  domain.DocumentSummary
		passages []domain.Passage
	}
	details := make([]docDetail, 0, len(docs))

	for _, d := range docs {
		domainSet[d.Domain] = struct{}{}
		totalBytes += d.Size
		fullDoc, err := s.GetDocument(ctx, d.ID)
		if err == nil {
			totalPassages += len(fullDoc.Passages)
			details = append(details, docDetail{
				summary:  d,
				passages: fullDoc.Passages,
			})
		} else {
			details = append(details, docDetail{summary: d})
		}
	}

	now := time.Now().UTC()
	var b strings.Builder

	b.WriteString("# DOSSIÊ DE INVESTIGAÇÃO TÉCNICA — NIE\n\n")
	b.WriteString(fmt.Sprintf("**Pergunta Técnica:** %s  \n", c.Query))
	b.WriteString(fmt.Sprintf("**ID da Campanha:** `%s`  \n", c.ID))
	b.WriteString(fmt.Sprintf("**Status da Coleta:** %s  \n", c.Status))
	b.WriteString(fmt.Sprintf("**Gerado em:** %s  \n", now.Format("02/01/2006 15:04:05 UTC")))
	b.WriteString("\n---\n\n")

	b.WriteString("## 1. Resumo Executivo da Coleta\n\n")
	b.WriteString(fmt.Sprintf("- **Documentos Preservados no Acervo:** %d\n", len(docs)))
	b.WriteString(fmt.Sprintf("- **Trechos Analisados e Citáveis:** %d\n", totalPassages))
	b.WriteString(fmt.Sprintf("- **Domínios Auditados:** %d\n", len(domainSet)))
	b.WriteString(fmt.Sprintf("- **Volume Total Preservado:** %s\n", formatBytes(totalBytes)))
	b.WriteString(fmt.Sprintf("- **Profundidade Máxima Atingida:** %d níveis\n", c.Budget.MaxDepth))
	b.WriteString("\n> **Nota de Validação Epistemológica:** Este material foi coletado e indexado de fontes públicas permitidas. " +
		"Trechos e comandos CLI devem ser validados em ambiente controlado de laboratório ou bancada antes de qualquer aplicação em equipamentos de produção.\n\n")

	b.WriteString("---\n\n")
	b.WriteString("## 2. Acervo de Evidências por Documento\n\n")

	if len(details) == 0 {
		b.WriteString("*Nenhum documento preservado nesta campanha até o momento.*\n\n")
	} else {
		for i, item := range details {
			d := item.summary
			title := d.Title
			if title == "" {
				title = d.URL
			}
			b.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, title))
			b.WriteString(fmt.Sprintf("- **URL de Origem:** [%s](%s)\n", d.URL, d.URL))
			b.WriteString(fmt.Sprintf("- **Domínio:** `%s`\n", d.Domain))
			b.WriteString(fmt.Sprintf("- **Hash Imutável (SHA-256):** `%s`\n", d.SHA256))
			b.WriteString(fmt.Sprintf("- **Tipo MIME:** `%s` (%s)\n", d.MIME, formatBytes(d.Size)))
			b.WriteString(fmt.Sprintf("- **Data da Coleta:** %s\n\n", d.FetchedAt.Format("02/01/2006 15:04:05 UTC")))

			if len(item.passages) > 0 {
				b.WriteString("#### Trechos Extraídos e Citáveis:\n\n")
				for _, p := range item.passages {
					pageStr := ""
					if p.Page != nil {
						pageStr = fmt.Sprintf(" · Página %d", *p.Page)
					}
					b.WriteString(fmt.Sprintf("> %s\n>\n> *— Trecho %d%s*\n\n", strings.ReplaceAll(p.Text, "\n", "\n> "), p.Ordinal+1, pageStr))
				}
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("---\n\n")
	b.WriteString("## 3. Grafo de Proveniência (Trilha de Descoberta)\n\n")
	if len(edges) == 0 {
		b.WriteString("*Nenhuma aresta de descoberta registrada.*\n\n")
	} else {
		b.WriteString("| Nível | Tipo | Origem | Destino | Justificativa |\n")
		b.WriteString("| :---: | :---: | :--- | :--- | :--- |\n")
		for _, e := range edges {
			from := e.FromURL
			if from == "" {
				from = "*(seed/busca inicial)*"
			}
			b.WriteString(fmt.Sprintf("| n%d | `%s` | %s | %s | %s |\n", e.Depth, e.Kind, from, e.ToURL, e.Reason))
		}
		b.WriteString("\n")
	}

	return domain.Dossier{
		Campaign:       c,
		DocumentsCount: len(docs),
		PassagesCount:  totalPassages,
		DomainsCount:   len(domainSet),
		BytesTotal:     totalBytes,
		Markdown:       b.String(),
		GeneratedAt:    now,
	}, nil
}

func formatBytes(n int64) string {
	if n <= 0 {
		return "0 B"
	}
	units := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	i := 0
	v := float64(n)
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%.0f %s", v, units[i])
	}
	return fmt.Sprintf("%.1f %s", v, units[i])
}
