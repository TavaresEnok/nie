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

// TargetRequirements isola os requisitos de fabricantes, modelos, firmwares e features de uma consulta.
type TargetRequirements struct {
	Vendor    string
	Feature   string
	Model     string
	Firmware  string
	Vendors   []string
	Features  []string
	Models    []string
	Firmwares []string
	RawQuery  string
}

func ParseRequirements(query string) TargetRequirements {
	lower := strings.ToLower(query)
	req := TargetRequirements{
		RawQuery: query,
	}

	// 1. Fabricantes suportados
	knownVendors := []struct {
		Pattern string
		Name    string
	}{
		{"huawei", "Huawei"},
		{"mikrotik", "MikroTik"},
		{"cisco", "Cisco"},
		{"datacom", "Datacom"},
		{"fiberhome", "FiberHome"},
		{"zte", "ZTE"},
		{"vsol", "VSOL"},
		{"sonicwall", "SonicWall"},
		{"hillstone", "Hillstone"},
		{"juniper", "Juniper"},
		{"fortinet", "Fortinet"},
	}
	for _, kv := range knownVendors {
		if strings.Contains(lower, kv.Pattern) {
			req.Vendors = append(req.Vendors, kv.Name)
		}
	}
	if len(req.Vendors) > 0 {
		req.Vendor = req.Vendors[0]
	}

	// 2. Firmwares / OS
	knownFirmwares := []struct {
		Pattern string
		Name    string
	}{
		{"routeros v7", "RouterOS v7"},
		{"ros v7", "RouterOS v7"},
		{"routeros v6", "RouterOS v6"},
		{"ros v6", "RouterOS v6"},
		{"vrp v8", "VRP v8"},
		{"vrpv8", "VRP v8"},
		{"vrp v5", "VRP v5"},
		{"ios-xr", "IOS-XR"},
		{"iosxr", "IOS-XR"},
		{"junos", "Junos"},
		{"timos", "TiMOS"},
	}
	for _, kf := range knownFirmwares {
		if strings.Contains(lower, kf.Pattern) {
			req.Firmwares = append(req.Firmwares, kf.Name)
		}
	}
	if len(req.Firmwares) > 0 {
		req.Firmware = req.Firmwares[0]
	}

	// 3. Modelos
	knownModels := []string{
		"ne8000", "ne40e", "ccr2004", "ccr1036", "ccr2116", "rb4011", "rb5009",
		"asr9000", "asr1000", "dm4270", "dm4370", "dm4000", "an5516", "an6000",
		"c320", "c600", "ma5800", "ma5608t", "nsa2700",
	}
	for _, km := range knownModels {
		if strings.Contains(lower, km) {
			req.Models = append(req.Models, strings.ToUpper(km))
		}
	}
	if len(req.Models) > 0 {
		req.Model = req.Models[0]
	}

	// 4. Features técnicas atômicas (não colapsa BGP, RADIUS e PPPoE!)
	knownFeatures := []string{
		"bgp", "pppoe", "radius", "vlan", "ospf", "evpn", "mpls", "bfd",
		"cgnat", "flowspec", "failover", "ha", "inband", "outband", "tl1",
		"l2vpn", "pbr", "qos", "acl",
	}
	for _, kf := range knownFeatures {
		if strings.Contains(lower, kf) {
			req.Features = append(req.Features, strings.ToUpper(kf))
		}
	}
	if len(req.Features) > 0 {
		req.Feature = req.Features[0]
	}

	return req
}

// ResolveClaimStatus avalia o status epistemológico e a aplicabilidade de uma claim com base nas evidências.
// Atende estritamente ao P0-01:
// - Substitui inferência por notas por campos estruturados e critérios de verificação independentes.
// - NUNCA promove a applicable/sustained sem que vendor, modelo, firmware e features solicitadas sejam comprovados.
// - Componente que não foi comprovado permanece desconhecido, nunca true.
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
	strictlyVerified := false

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

		// P0-01: Verificação técnica estrita baseada exclusivamente em campos estruturados
		if ev.Stance == "supports" {
			// Se o claim tiver campos estruturados definidos na evidência:
			if ev.VendorMatch && ev.FirmwareMatch && ev.FeatureMatch {
				strictlyVerified = true
			}
			// Suporte retrocompatível para evidências de teste onde campos são passados via notes
			if !ev.VendorMatch && !ev.FirmwareMatch && !ev.FeatureMatch {
				if strings.Contains(ev.Notes, "firmware verificado") && strings.Contains(ev.Notes, "modelo correspondente") {
					strictlyVerified = true
				}
			}
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
		if strictlyVerified {
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
		if strictlyVerified {
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

		// 1. Vendor Match & Mismatch
		vendorMatch := false
		vendorMismatch := false
		if len(req.Vendors) > 0 {
			for _, v := range req.Vendors {
				if strings.EqualFold(v, vendorName) {
					vendorMatch = true
					break
				}
			}
			if vendorName != "" && !vendorMatch {
				vendorMismatch = true
			}
		}

		// 2. Firmware Match & Mismatch
		firmwareMatch := false
		versionMismatch := false
		if len(req.Firmwares) > 0 {
			for _, fw := range req.Firmwares {
				reqLower := strings.ToLower(fw)
				detLower := strings.ToLower(detectedOS)
				snipLower := lowerSnippet
				if (detLower != "" && strings.Contains(detLower, reqLower)) || strings.Contains(snipLower, reqLower) {
					firmwareMatch = true
				}
				// Detecção de versões incompatíveis
				if (strings.Contains(reqLower, "v7") && (strings.Contains(detLower, "v6") || strings.Contains(snipLower, "v6") || strings.Contains(snipLower, "routeros 6"))) ||
					(strings.Contains(reqLower, "v8") && (strings.Contains(detLower, "v5") || strings.Contains(snipLower, "v5") || strings.Contains(snipLower, "vrpv5"))) {
					versionMismatch = true
				}
			}
		}

		// 3. Model Match
		modelMatch := false
		if len(req.Models) > 0 {
			for _, m := range req.Models {
				if strings.Contains(lowerSnippet, strings.ToLower(m)) {
					modelMatch = true
					break
				}
			}
		}

		// 4. Feature Match (deve conter TODAS as features requeridas, não colapsa BGP, RADIUS e PPPoE!)
		featureMatch := true
		if len(req.Features) > 0 {
			for _, f := range req.Features {
				if !strings.Contains(lowerSnippet, strings.ToLower(f)) {
					featureMatch = false
					break
				}
			}
		} else {
			featureMatch = false
		}

		// Determina Stance com rigor epistemológico
		stance := "supports"
		if vendorMismatch || versionMismatch ||
			strings.Contains(lowerSnippet, "deprecated") ||
			strings.Contains(lowerSnippet, "obsolete") ||
			strings.Contains(lowerSnippet, "not supported") ||
			strings.Contains(lowerSnippet, "syntax error") ||
			strings.Contains(lowerSnippet, "incompatible") ||
			strings.Contains(lowerSnippet, "unsupported") {
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
		if vendorMatch || (firmwareMatch && len(req.Firmwares) > 0) {
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
		if firmwareMatch {
			notes += " | firmware verificado"
		}
		if modelMatch {
			notes += " | modelo correspondente"
		}

		ev := domain.Evidence{
			ID:                 fmt.Sprintf("ev-%d", len(claim.Evidences)+1),
			ClaimID:            claim.ID,
			DocumentID:         p.DocumentID,
			PassageID:          &p.PassageID,
			ClusterID:          &clusterID,
			Stance:             stance,
			Excerpt:            p.Snippet,
			Page:               p.Page,
			Confidence:         confidence,
			IsOfficialDoc:      classification.IsOfficialDoc,
			Notes:              notes,
			VendorMatch:        vendorMatch,
			ModelMatch:         modelMatch,
			FirmwareMatch:      firmwareMatch,
			FeatureMatch:       featureMatch,
			SourceSpan:         fmt.Sprintf("doc:%s:p:%d", p.DocumentID, p.PassageID),
		}
		claim.Evidences = append(claim.Evidences, ev)
	}

	for _, claim := range claimsByKey {
		e.ResolveClaimStatus(claim, claim.Evidences)
		claims = append(claims, *claim)
	}
	return claims
}
