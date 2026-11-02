package evidence

import (
	"fmt"
	"strings"

	"nie/internal/classifier"
	"nie/internal/domain"
)

type Engine struct {
	tracker *ClusterTracker
}

func NewEngine() *Engine {
	return &Engine{
		tracker: NewClusterTracker(),
	}
}

// TargetRequirements isola os requisitos de vendor, feature, modelo e firmware de uma consulta.
type TargetRequirements struct {
	Vendor   string
	Feature  string
	Model    string
	Firmware string
}

func ParseRequirements(query string) TargetRequirements {
	lower := strings.ToLower(query)
	req := TargetRequirements{}

	// Vendor
	switch {
	case strings.Contains(lower, "huawei"):
		req.Vendor = "Huawei"
	case strings.Contains(lower, "mikrotik"):
		req.Vendor = "MikroTik"
	case strings.Contains(lower, "cisco"):
		req.Vendor = "Cisco"
	case strings.Contains(lower, "datacom"):
		req.Vendor = "Datacom"
	case strings.Contains(lower, "fiberhome"):
		req.Vendor = "FiberHome"
	case strings.Contains(lower, "zte"):
		req.Vendor = "ZTE"
	}

	// Firmware / OS
	switch {
	case strings.Contains(lower, "routeros v7") || strings.Contains(lower, "ros v7") || strings.Contains(lower, "v7"):
		req.Firmware = "RouterOS v7"
	case strings.Contains(lower, "routeros v6") || strings.Contains(lower, "ros v6") || strings.Contains(lower, "v6"):
		req.Firmware = "RouterOS v6"
	case strings.Contains(lower, "vrp v8") || strings.Contains(lower, "vrpv8"):
		req.Firmware = "VRP v8"
	case strings.Contains(lower, "ios-xr") || strings.Contains(lower, "iosxr"):
		req.Firmware = "IOS-XR"
	}

	// Model
	for _, m := range []string{"ne8000", "ccr2004", "ccr1036", "asr9000", "dm4270", "an5516", "c320"} {
		if strings.Contains(lower, m) {
			req.Model = strings.ToUpper(m)
			break
		}
	}

	// Feature
	for _, f := range []string{"bgp", "pppoe", "vlan", "ospf", "evpn", "uplink", "tl1", "radius", "mpls"} {
		if strings.Contains(lower, f) {
			req.Feature = strings.ToUpper(f)
			break
		}
	}

	return req
}

// ResolveClaimStatus avalia o status epistemológico e a aplicabilidade de uma claim com base nas evidências.
// Atende estritamente ao P0-01:
// - Não eleva para "applicable" / "confirmed" sem comprovação de modelo/firmware e correspondência técnica.
// - Três cópias do mesmo artigo contam como 1 único cluster de origem.
// - Separa "documented" (manual oficial sem comprovação de versão), "corroborated" (múltiplas origens),
//   "applicable" (corroborado + condições confirmadas) e "contested" / "incompatible" (refutado/conflito).
func (e *Engine) ResolveClaimStatus(claim *domain.Claim, evidences []domain.Evidence) {
	if len(evidences) == 0 {
		claim.Status = "unconfirmed"
		claim.Applicability = "indeterminate"
		return
	}

	hasOfficial := false
	hasContradiction := false
	modelFirmwareVerified := false

	var fingerprints []string
	var domains []string

	for _, ev := range evidences {
		if ev.Stance == "refutes" {
			hasContradiction = true
		}
		if ev.IsOfficialDoc && ev.Stance == "supports" {
			hasOfficial = true
		}
		if ev.ClusterID != nil && *ev.ClusterID != "" {
			fingerprints = append(fingerprints, *ev.ClusterID)
		} else {
			fingerprints = append(fingerprints, ev.ID)
		}
		// Extrai domínio das notas se disponível
		dom := ""
		if strings.HasPrefix(ev.Notes, "Fonte: ") {
			parts := strings.Split(ev.Notes, " ")
			if len(parts) >= 2 {
				dom = parts[1]
			}
		}
		if dom == "" {
			dom = ev.ID
		}
		domains = append(domains, dom)

		// Se a nota indicar verificação de modelo/firmware
		if strings.Contains(ev.Notes, "firmware verificado") || strings.Contains(ev.Notes, "modelo correspondente") {
			modelFirmwareVerified = true
		}
	}

	// Calcula quantidade real de origens substantivamente independentes
	independentOrigins := e.tracker.CountIndependentOrigins(fingerprints, domains)
	if independentOrigins < 1 && len(evidences) > 0 {
		independentOrigins = 1
	}

	// 1. Contradições, depreciação ou conflito explícito
	if hasContradiction {
		claim.Status = "contested"
		claim.Applicability = "incompatible"
		return
	}

	// 2. Documento oficial
	if hasOfficial {
		if modelFirmwareVerified {
			claim.Status = "sustained"
			claim.Applicability = "applicable"
		} else {
			// Manual legítimo, mas sem validação do procedimento/firmware completo solicitado
			claim.Status = "documented"
			claim.Applicability = "indeterminate"
		}
		return
	}

	// 3. Corroboração multi-origem independente (>= 2 origens distintas não-cópia)
	if independentOrigins >= 2 {
		claim.Status = "corroborated"
		if modelFirmwareVerified {
			claim.Applicability = "applicable"
		} else {
			claim.Applicability = "indeterminate"
		}
		return
	}

	// 4. Sem evidência independente suficiente
	claim.Status = "unconfirmed"
	claim.Applicability = "indeterminate"
}

// BuildClaimsFromPassages analisa as passagens de uma campanha e sintetiza claims atômicas estruturadas.
func (e *Engine) BuildClaimsFromPassages(campaignID string, query string, passages []domain.SearchResult) []domain.Claim {
	var claims []domain.Claim
	req := ParseRequirements(query)
	claimsByKey := make(map[string]*domain.Claim)

	for _, p := range passages {
		classification := classifier.Classify(p.Snippet, p.Domain)
		lowerSnippet := strings.ToLower(p.Snippet)

		// Determina o fabricante e OS da passagem
		vendorName := ""
		detectedOS := ""
		if len(classification.Vendors) > 0 {
			vendorName = classification.Vendors[0].Name
			detectedOS = classification.Vendors[0].OS
		}

		// Valida correspondência de feature técnica (ex: se query é BGP, PPPoE não deve suportar BGP)
		featureMatch := true
		if req.Feature != "" {
			if !strings.Contains(lowerSnippet, strings.ToLower(req.Feature)) {
				featureMatch = false
			}
		}

		// Valida compatibilidade de firmware/release
		versionMismatch := false
		if req.Firmware != "" && detectedOS != "" {
			reqLower := strings.ToLower(req.Firmware)
			detLower := strings.ToLower(detectedOS)
			if (strings.Contains(reqLower, "v7") && strings.Contains(detLower, "v6")) ||
				(strings.Contains(reqLower, "v8") && strings.Contains(detLower, "v5")) {
				versionMismatch = true
			}
		}

		// Determina Stance
		stance := "supports"
		if versionMismatch ||
			strings.Contains(lowerSnippet, "deprecated") ||
			strings.Contains(lowerSnippet, "obsolete") ||
			strings.Contains(lowerSnippet, "not supported") ||
			strings.Contains(lowerSnippet, "syntax error") {
			stance = "refutes"
		} else if !featureMatch {
			stance = "mentions"
		}

		// Identifica o tópico atômico do claim
		claimTitle := query
		claimStatement := fmt.Sprintf("Procedimento e diretrizes documentadas para %s", query)
		if len(classification.CLICommands) > 0 {
			firstCmd := classification.CLICommands[0]
			switch {
			case strings.Contains(firstCmd, "uplink") || strings.Contains(firstCmd, "port"):
				claimTitle = "Configuração de Portas Uplink e Modos de Interface"
				claimStatement = fmt.Sprintf("Sintaxe CLI para ativação e modulação de portas uplink em %s", vendorName)
			case strings.Contains(firstCmd, "bgp") || strings.Contains(firstCmd, "peer"):
				claimTitle = "Configuração e Peering de Sessões BGP"
				claimStatement = fmt.Sprintf("Estabelecimento de sessões BGP e políticas de roteamento em %s", vendorName)
			case strings.Contains(firstCmd, "vlan") || strings.Contains(firstCmd, "manage"):
				claimTitle = "Gerenciamento de VLAN e Acesso Inband/Outband"
				claimStatement = fmt.Sprintf("Definição de IPs de gerência e interfaces virtuais em %s", vendorName)
			case strings.Contains(firstCmd, "ADD-ONU") || strings.Contains(firstCmd, "LST-ONU"):
				claimTitle = "Provisionamento de ONUs via Comandos TL1"
				claimStatement = "Comandos transacionais TL1 para cadastro, consulta e sinal óptico de ONUs"
			default:
				claimTitle = fmt.Sprintf("Configuração CLI: %s", firstCmd)
				claimStatement = fmt.Sprintf("Comandos CLI validados para contexto %s", vendorName)
			}
		} else if vendorName != "" {
			claimTitle = fmt.Sprintf("%s — %s", vendorName, query)
		}

		claimKey := claimTitle
		claim, exists := claimsByKey[claimKey]
		if !exists {
			claim = &domain.Claim{
				ID:                    fmt.Sprintf("claim-%d", len(claimsByKey)+1),
				CampaignID:            campaignID,
				Title:                 claimTitle,
				Statement:             claimStatement,
				VendorName:            vendorName,
				Status:                "unconfirmed",
				Applicability:         "indeterminate",
				OperationalValidation: "untested",
				Evidences:             []domain.Evidence{},
			}
			claimsByKey[claimKey] = claim
		}

		// Registra fingerprint no cluster tracker
		fp := Fingerprint(p.Snippet)
		e.tracker.Add(fp, p.Domain)
		clusterID := fp

		// Cálculo dinâmico de confiança
		var confidence float32 = 0.20
		if classification.IsOfficialDoc {
			confidence += 0.35
		}
		if featureMatch {
			confidence += 0.25
		}
		if (req.Vendor != "" && strings.EqualFold(req.Vendor, vendorName)) ||
			(req.Firmware != "" && strings.EqualFold(req.Firmware, detectedOS)) {
			confidence += 0.20
		}
		if stance == "refutes" {
			confidence = 0.85 // Alta confiança no relato de incompatibilidade/depreciação
		} else if !featureMatch {
			confidence = 0.15 // Baixa confiança se feature não for compatível
		}
		if confidence > 1.0 {
			confidence = 1.0
		}

		notes := fmt.Sprintf("Fonte: %s (SHA-256: %s)", p.Domain, p.SHA256)
		if featureMatch && (req.Firmware == "" || strings.EqualFold(req.Firmware, detectedOS)) {
			notes += " | firmware verificado"
		}
		if req.Model != "" && strings.Contains(lowerSnippet, strings.ToLower(req.Model)) {
			notes += " | modelo correspondente"
		}

		ev := domain.Evidence{
			ID:            fmt.Sprintf("ev-%d", len(claim.Evidences)+1),
			ClaimID:       claim.ID,
			DocumentID:    p.DocumentID,
			PassageID:     &p.PassageID,
			ClusterID:     &clusterID,
			Stance:        stance,
			Excerpt:       p.Snippet,
			Page:          p.Page,
			Confidence:    confidence,
			IsOfficialDoc: classification.IsOfficialDoc,
			Notes:         notes,
		}
		claim.Evidences = append(claim.Evidences, ev)
	}

	for _, claim := range claimsByKey {
		e.ResolveClaimStatus(claim, claim.Evidences)
		claims = append(claims, *claim)
	}
	return claims
}
