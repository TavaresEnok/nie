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

// AdaptiveInvestigationLead estabelece o contrato cognitivo:
// research_question -> hypothesis -> search_lead -> evidence_gap -> next_action (P1-09)
type AdaptiveInvestigationLead struct {
	ResearchQuestion string `json:"research_question"`
	Hypothesis       string `json:"hypothesis"`
	SearchLead       string `json:"search_lead"`
	EvidenceGap      string `json:"evidence_gap"`
	NextAction       string `json:"next_action"`
}

// PlanAdaptiveLeads replaneja a investigação com base nas contradições, lacunas de firmware e nível de corroboração.
func (s *Service) PlanAdaptiveLeads(ctx context.Context, query string, existingClaims []domain.Claim) []AdaptiveInvestigationLead {
	var leads []AdaptiveInvestigationLead

	for _, c := range existingClaims {
		switch c.Status {
		case "contested":
			// Contradição detectada: formula hipótese de incompatibilidade entre releases
			leads = append(leads, AdaptiveInvestigationLead{
				ResearchQuestion: query,
				Hypothesis:       fmt.Sprintf("A sintaxe ou procedimento para %s sofreu depreciação ou alteração entre releases", c.Title),
				SearchLead:       fmt.Sprintf("%s syntax change deprecated release notes migration", query),
				EvidenceGap:      "Conflito entre fontes ou aviso de comando obsoleto não resolvido por documento canônico da versão",
				NextAction:       "Consultar notas de versão oficiais e manuais de migração do fabricante para datar a quebra de compatibilidade",
			})
		case "documented":
			// Documento oficial encontrado, mas sem comprovação de firmware/modelo específico
			leads = append(leads, AdaptiveInvestigationLead{
				ResearchQuestion: query,
				Hypothesis:       fmt.Sprintf("O comando é suportado oficialmente em %s, mas requer confirmação de versão mínima de OS", c.VendorName),
				SearchLead:       fmt.Sprintf("%s %s release requirement minimum firmware compatibility", c.VendorName, query),
				EvidenceGap:      "Ausência de especificação exata de modelo ou versão de firmware na evidência oficial coletada",
				NextAction:       "Buscar matriz de compatibilidade de hardware e software do fabricante",
			})
		case "unconfirmed":
			// Menos de 2 origens independentes
			leads = append(leads, AdaptiveInvestigationLead{
				ResearchQuestion: query,
				Hypothesis:       fmt.Sprintf("Existem relatos de procedimento para %s, mas dependem de corroboração multi-origem", c.Title),
				SearchLead:       fmt.Sprintf("%s configuration example practical guide forum ISP", query),
				EvidenceGap:      "Apenas 1 origem ou réplicas do mesmo artigo (falta segunda fonte independente não-cópia)",
				NextAction:       "Localizar discussão independente de operadores ou repositório de configuração real",
			})
		}
	}

	if len(leads) == 0 {
		leads = append(leads, AdaptiveInvestigationLead{
			ResearchQuestion: query,
			Hypothesis:       fmt.Sprintf("Procedimento padrão documentado para %s", query),
			SearchLead:       fmt.Sprintf("%s official guide configuration syntax", query),
			EvidenceGap:      "Nenhuma claim formulada ainda para a consulta",
			NextAction:       "Iniciar coleta exploratória a partir de seeds e metabusca",
		})
	}
	return leads
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

	// Afirmações sem evidência suficiente nunca aparecem como conclusões: cada
	// uma recebe uma advertência explícita, e o resumo diz quantas há.
	supported, documented, insufficient, contested := 0, 0, 0, 0
	for _, c := range claims {
		keyFindings = append(keyFindings, fmt.Sprintf("[%s] %s: %s", strings.ToUpper(c.Status), c.Title, c.Statement))
		switch c.Status {
		case "contested":
			contested++
			warnings = append(warnings, fmt.Sprintf("Atenção: A configuração para '%s' possui evidências conflitantes ou relatos de incompatibilidade.", c.Title))
		case "documented":
			documented++
			warnings = append(warnings, fmt.Sprintf("'%s' consta em documentação, mas a aplicabilidade ao modelo, firmware e recurso pedidos não foi comprovada.", c.Title))
		case "sustained", "corroborated", "validated_lab":
			supported++
		default:
			insufficient++
			warnings = append(warnings, fmt.Sprintf("Evidência insuficiente: '%s' não tem documentação oficial nem origens independentes que a sustentem.", c.Title))
		}
	}

	if len(warnings) == 0 {
		warnings = append(warnings, "Recomenda-se validação em bancada antes de aplicação em roteadores em produção.")
		warnings = append(warnings, "Verifique se a versão de firmware/software em execução suporta as sintaxes especificadas.")
	}

	summary := fmt.Sprintf("Dossiê de investigação técnica para a consulta: '%s'. Foram analisadas %d passagens de evidência e consolidados %d claims técnicos "+
		"(%d sustentados ou corroborados, %d documentados sem aplicabilidade comprovada, %d sem evidência suficiente, %d contestados).",
		campaign.Query, len(passages), len(claims), supported, documented, insufficient, contested)
	if supported == 0 {
		summary += " Nenhuma afirmação atingiu evidência suficiente para ser tratada como conclusão."
	}

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

	// Rollback e checklist são orientação genérica de redes, não extraída das
	// evidências: só entram quando há fabricante de rede identificado.
	networkContext := len(compatibilityMatrix) > 0
	md.WriteString("## 6. Procedimento de Rollback Recomendado\n\n")
	if networkContext {
		md.WriteString("> Orientação genérica do NIE, **não extraída das evidências desta campanha**.\n\n")
	} else {
		md.WriteString("*Não aplicável: nenhum fabricante de rede identificado nas evidências.*\n\n")
	}
	rollbackPlan :="Em caso de falha durante aplicação das configurações:\n" +
		"1. Mantenha console/OOB ativa antes de alterar tabelas de roteamento ou firewall.\n" +
		"2. No Huawei VRP utilize 'undo <comando>' ou faça 'rollback configuration'.\n" +
		"3. No MikroTik utilize o modo Safe Mode (Ctrl+X) antes de testar regras de BGP/filtros.\n" +
		"4. No Cisco IOS-XR/XE utilize 'rollback configuration last 1' para restauração imediata.\n" +
		"5. No Datacom DmOS utilize 'no <comando>' e confirme com 'show running-config'."
	if networkContext {
		md.WriteString(rollbackPlan + "\n\n")
	} else {
		rollbackPlan = ""
	}

	md.WriteString("## 7. Checklist de Validação em Laboratório\n\n")
	checklist := []string{
		"Confirmar versão exata do software/firmware em execução via CLI.",
		"Verificar se licenças de recursos avançados (BNG, CGNAT, EVPN) estão ativas no chassi.",
		"Validar compatibilidade de MTU e tabelas de adjacência antes de fechar peering BGP.",
		"Testar contingência e comportamento de failover durante janela de manutenção autorizada.",
	}
	if !networkContext {
		checklist = nil
		md.WriteString("*Não aplicável: nenhum fabricante de rede identificado nas evidências.*\n")
	}
	for _, item := range checklist {
		md.WriteString(fmt.Sprintf("- [ ] %s\n", item))
	}
	md.WriteString("\n")

	// P1-09: Replanejamento cognitivo contínuo guiado por contradições e lacunas
	adaptiveLeads := s.PlanAdaptiveLeads(ctx, campaign.Query, claims)
	var domainLeads []domain.InvestigationLead
	md.WriteString("## 8. Próximas Pistas Investigativas (Ciclo Adaptativo Guiado por Lacunas)\n\n")
	md.WriteString("| Hipótese Técnica | Lacuna de Evidência Identificada | Próxima Ação Investigativa | Consulta de Descoberta |\n|---|---|---|---|\n")
	for _, l := range adaptiveLeads {
		domainLeads = append(domainLeads, domain.InvestigationLead{
			ResearchQuestion: l.ResearchQuestion,
			Hypothesis:       l.Hypothesis,
			SearchLead:       l.SearchLead,
			EvidenceGap:      l.EvidenceGap,
			NextAction:       l.NextAction,
		})
		md.WriteString(fmt.Sprintf("| %s | %s | %s | `%s` |\n", l.Hypothesis, l.EvidenceGap, l.NextAction, l.SearchLead))
	}
	md.WriteString("\n")

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
		InvestigationLeads:  domainLeads,
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
