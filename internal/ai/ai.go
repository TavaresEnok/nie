package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"nie/internal/classifier"
	"nie/internal/domain"
)

type Config struct {
	Provider string // "none", "ollama", "openai"
	BaseURL  string // ex: "http://localhost:11434" ou "https://api.openai.com/v1"
	APIKey   string
	Model    string
	Timeout  time.Duration
}

type Service struct {
	cfg        Config
	httpClient *http.Client
}

func NewService(cfg Config) *Service {
	if cfg.Timeout == 0 {
		cfg.Timeout = 45 * time.Second
	}
	if cfg.Provider == "" {
		cfg.Provider = "none"
	}
	return &Service{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// PlanSubquestions generates subqueries to deepen research with vendor awareness.
func (s *Service) PlanSubquestions(ctx context.Context, query string) []string {
	cl := classifier.Classify(query, "")
	var base []string
	cleanQ := strings.TrimSpace(query)

	if len(cl.Vendors) > 0 {
		v := cl.Vendors[0]
		switch v.Name {
		case "Huawei":
			base = append(base,
				fmt.Sprintf("%s VRP display current-configuration syntax", cleanQ),
				fmt.Sprintf("%s HedEx product documentation release notes", cleanQ),
				fmt.Sprintf("%s rollback configuration undo syntax caveats", cleanQ),
				fmt.Sprintf("%s troubleshooting operational verification", cleanQ),
			)
		case "MikroTik":
			base = append(base,
				fmt.Sprintf("%s RouterOS v7 syntax difference v6 routing filter", cleanQ),
				fmt.Sprintf("%s CLI print detail export command", cleanQ),
				fmt.Sprintf("%s Safe Mode caveats known limitations", cleanQ),
				fmt.Sprintf("%s forum mikrotik official documentation", cleanQ),
			)
		case "Cisco":
			base = append(base,
				fmt.Sprintf("%s IOS-XR commit confirmed rollback configuration", cleanQ),
				fmt.Sprintf("%s show running-config syntax verification", cleanQ),
				fmt.Sprintf("%s caveats bug search release notes", cleanQ),
				fmt.Sprintf("%s Cisco TAC deployment guideline", cleanQ),
			)
		case "Datacom":
			base = append(base,
				fmt.Sprintf("%s DmOS show running-config syntax route-map", cleanQ),
				fmt.Sprintf("%s manual configuracao suporte datacom", cleanQ),
				fmt.Sprintf("%s firmware release notes limitacoes", cleanQ),
				fmt.Sprintf("%s interface L3 vlan comandos", cleanQ),
			)
		case "FiberHome":
			base = append(base,
				fmt.Sprintf("%s AN5516 AN6000 manual configuracao CLI", cleanQ),
				fmt.Sprintf("%s TL1 northbound interface commands", cleanQ),
				fmt.Sprintf("%s ONU authorization profile provisioning", cleanQ),
				fmt.Sprintf("%s firmware update troubleshooting", cleanQ),
			)
		default:
			base = append(base,
				fmt.Sprintf("%s %s CLI configuration syntax example", cleanQ, v.Name),
				fmt.Sprintf("%s %s verification show display commands", cleanQ, v.Name),
				fmt.Sprintf("%s %s limitations caveats firmware", cleanQ, v.Name),
				fmt.Sprintf("%s %s failover redundancy high availability", cleanQ, v.Name),
			)
		}
	} else {
		base = []string{
			fmt.Sprintf("%s syntax CLI example configuration", cleanQ),
			fmt.Sprintf("%s troubleshooting verification display show", cleanQ),
			fmt.Sprintf("%s limitations caveats compatibility firmware", cleanQ),
			fmt.Sprintf("%s failover redundancy high availability", cleanQ),
		}
	}

	if s.cfg.Provider == "none" || s.cfg.BaseURL == "" {
		return base
	}

	prompt := fmt.Sprintf("Gere 4 subperguntas de busca técnica focadas em roteadores/redes para aprofundar: %s. Responda apenas com 1 pergunta por linha.", query)
	resp, err := s.callLLM(ctx, prompt)
	if err != nil || len(resp) == 0 {
		return base
	}

	lines := strings.Split(resp, "\n")
	var out []string
	for _, l := range lines {
		trimmed := strings.TrimSpace(strings.TrimLeft(l, "0123456789.-* "))
		if len(trimmed) > 5 {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return base
	}
	return out
}

type CommandProvenance struct {
	Command string
	Domain  string
	URL     string
}

// GenerateInvestigationDossier creates a full technical dossier based on claims and verified passages.
func (s *Service) GenerateInvestigationDossier(ctx context.Context, campaign domain.Campaign, claims []domain.Claim, passages []domain.SearchResult) domain.InvestigationDossier {
	var cliCommands []string
	var cmdProvenance []CommandProvenance
	seenCmd := make(map[string]bool)
	var warnings []string
	compatibilityMatrix := make(map[string]string)
	var keyFindings []string

	for _, p := range passages {
		cl := classifier.Classify(p.Snippet, p.Domain)
		for _, cmd := range cl.CLICommands {
			if !seenCmd[cmd] {
				seenCmd[cmd] = true
				cliCommands = append(cliCommands, cmd)
				cmdProvenance = append(cmdProvenance, CommandProvenance{
					Command: cmd,
					Domain:  p.Domain,
					URL:     p.URL,
				})
			}
		}
		for _, v := range cl.Vendors {
			if v.OS != "" {
				compatibilityMatrix[v.Name] = v.OS
			}
		}
	}

	cliCommands = dedupe(cliCommands)

	for _, c := range claims {
		keyFindings = append(keyFindings, fmt.Sprintf("[%s] %s: %s", strings.ToUpper(c.Status), c.Title, c.Statement))
		if c.Status == "contested" {
			warnings = append(warnings, fmt.Sprintf("Atenção: A configuração para '%s' possui evidências conflitantes ou relatos de incompatibilidade.", c.Title))
		}
	}

	if len(warnings) == 0 {
		warnings = append(warnings, "Recomenda-se validação em bancada antes de aplicação em roteadores em produção.")
		warnings = append(warnings, "Verifique se a versão de firmware/software em execução suporta as sintaxes especificadas.")
	}

	summary := fmt.Sprintf("Dossiê de investigação técnica para a consulta: '%s'. Foram analisadas %d passagens de evidência e consolidados %d claims técnicos.", campaign.Query, len(passages), len(claims))

	// Markdown synthesis
	var md strings.Builder
	md.WriteString(fmt.Sprintf("# Dossiê de Investigação Técnica — NIE\n\n"))
	md.WriteString(fmt.Sprintf("**Campanha:** `%s`  \n", campaign.ID))
	md.WriteString(fmt.Sprintf("**Objetivo da Investigação:** %s  \n", campaign.Query))
	md.WriteString(fmt.Sprintf("**Data:** %s  \n\n", time.Now().Format("02/01/2006 15:04:05")))

	md.WriteString("## 1. Resumo Executivo\n\n")
	md.WriteString(summary + "\n\n")

	md.WriteString("## 2. Matriz de Fabricantes e Firmware Identificados\n\n")
	if len(compatibilityMatrix) > 0 {
		md.WriteString("| Fabricante | Sistema Operacional / Release |\n|---|---|\n")
		for vendor, os := range compatibilityMatrix {
			md.WriteString(fmt.Sprintf("| **%s** | %s |\n", vendor, os))
		}
		md.WriteString("\n")
	} else {
		md.WriteString("*Nenhum fabricante específico isolado nas evidências coletadas.*\n\n")
	}

	md.WriteString("## 3. Comandos e Sintaxes CLI Identificados nas Evidências (Hipóteses Preliminares)\n\n")
	md.WriteString("> **Aviso Epistemológico:** Os comandos abaixo foram extraídos textualmente a partir dos documentos catalogados. Eles constituem **hipóteses preliminares de configuração** e **NÃO foram individualmente executados ou homologados em bancada de laboratório**. É obrigatória a validação da sintaxe exata para a release e licença do seu equipamento antes de qualquer aplicação em ambiente de produção.\n\n")
	if len(cmdProvenance) > 0 {
		md.WriteString("| Linha / Comando CLI | Domínio de Origem | Rastreabilidade |\n|---|---|---|\n")
		for _, cp := range cmdProvenance {
			md.WriteString(fmt.Sprintf("| `%s` | %s | `%s` |\n", cp.Command, cp.Domain, cp.URL))
		}
		md.WriteString("\n")
	} else if len(cliCommands) > 0 {
		md.WriteString("```text\n")
		for _, cmd := range cliCommands {
			md.WriteString(cmd + "\n")
		}
		md.WriteString("```\n\n")
	} else {
		md.WriteString("*Nenhum comando estruturado isolado diretamente nos trechos.*\n\n")
	}

	md.WriteString("## 4. Auditoria de Claims e Evidências Independentes\n\n")
	md.WriteString("A plataforma cruza evidências e agrupa fontes duplicadas em *Origin Clusters* para evitar que cópias contem como validações independentes.\n\n")
	for i, c := range claims {
		md.WriteString(fmt.Sprintf("### Claim %d: %s\n", i+1, c.Title))
		md.WriteString(fmt.Sprintf("- **Afirmação:** %s\n", c.Statement))
		md.WriteString(fmt.Sprintf("- **Status Epistemológico:** `%s`\n", c.Status))
		md.WriteString(fmt.Sprintf("- **Aplicabilidade:** `%s` | **Validação Operacional:** `%s`\n", c.Applicability, c.OperationalValidation))
		md.WriteString("- **Evidências Rastreadas:**\n")
		for _, ev := range c.Evidences {
			pageInfo := ""
			if ev.Page != nil {
				pageInfo = fmt.Sprintf(" (Pág. %d)", *ev.Page)
			}
			officialTag := ""
			if ev.IsOfficialDoc {
				officialTag = " [DOC OFICIAL]"
			}
			md.WriteString(fmt.Sprintf("  - *\"%s\"*%s%s — %s\n", strings.TrimSpace(ev.Excerpt), pageInfo, officialTag, ev.Notes))
		}
		md.WriteString("\n")
	}

	md.WriteString("## 5. Advertências e Pontos Críticos de Atenção\n\n")
	for _, w := range warnings {
		md.WriteString(fmt.Sprintf("- ⚠️ %s\n", w))
	}
	md.WriteString("\n")

	md.WriteString("## 6. Procedimento de Rollback Recomendado\n\n")
	rollbackPlan := "Em caso de falha durante aplicação das configurações:\n" +
		"1. Mantenha console/OOB ativa antes de alterar tabelas de roteamento ou firewall.\n" +
		"2. No Huawei VRP utilize 'undo <comando>' ou faça 'rollback configuration'.\n" +
		"3. No MikroTik utilize o modo Safe Mode (Ctrl+X) antes de testar regras de BGP/filtros.\n" +
		"4. No Cisco IOS-XR/XE utilize 'rollback configuration last 1' para restauração imediata.\n" +
		"5. No Datacom DmOS utilize 'no <comando>' e confirme com 'show running-config'."
	md.WriteString(rollbackPlan + "\n\n")

	md.WriteString("## 7. Checklist de Validação em Laboratório\n\n")
	checklist := []string{
		"Confirmar versão exata do software/firmware em execução via CLI.",
		"Verificar se licenças de recursos avançados (BNG, CGNAT, EVPN) estão ativas no chassi.",
		"Validar compatibilidade de MTU e tabelas de adjacência antes de fechar peering BGP.",
		"Testar contingência e comportamento de failover durante janela de manutenção autorizada.",
	}
	for _, item := range checklist {
		md.WriteString(fmt.Sprintf("- [ ] %s\n", item))
	}

	return domain.InvestigationDossier{
		Campaign:            campaign,
		Summary:             summary,
		KeyFindings:         keyFindings,
		Claims:              claims,
		CompatibilityMatrix: compatibilityMatrix,
		CLICommands:         cliCommands,
		Warnings:            warnings,
		RollbackPlan:        rollbackPlan,
		LabChecklist:        checklist,
		Markdown:            md.String(),
		GeneratedAt:         time.Now(),
	}
}

func (s *Service) callLLM(ctx context.Context, prompt string) (string, error) {
	if s.cfg.Provider == "ollama" {
		payload := map[string]any{
			"model":  s.cfg.Model,
			"prompt": prompt,
			"stream": false,
		}
		data, _ := json.Marshal(payload)
		url := fmt.Sprintf("%s/api/generate", strings.TrimRight(s.cfg.BaseURL, "/"))
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := s.httpClient.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var res struct {
			Response string `json:"response"`
		}
		_ = json.Unmarshal(body, &res)
		return res.Response, nil
	}

	if s.cfg.Provider == "openai" {
		payload := map[string]any{
			"model": s.cfg.Model,
			"messages": []map[string]string{
				{"role": "user", "content": prompt},
			},
		}
		data, _ := json.Marshal(payload)
		url := fmt.Sprintf("%s/chat/completions", strings.TrimRight(s.cfg.BaseURL, "/"))
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
		if s.cfg.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)
		}
		resp, err := s.httpClient.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var res struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		_ = json.Unmarshal(body, &res)
		if len(res.Choices) > 0 {
			return res.Choices[0].Message.Content, nil
		}
	}

	return "", fmt.Errorf("provedor de IA desconhecido ou não configurado")
}

func dedupe(list []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, item := range list {
		if !seen[item] {
			seen[item] = true
			out = append(out, item)
		}
	}
	return out
}
