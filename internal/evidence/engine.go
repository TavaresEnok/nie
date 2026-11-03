package evidence

import (
	"fmt"
	"regexp"
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
		if hasToken(lower, kv.Pattern) {
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
		if hasToken(lower, kf.Pattern) {
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
		if hasToken(lower, km) {
			req.Models = append(req.Models, strings.ToUpper(km))
		}
	}
	if len(req.Models) > 0 {
		req.Model = req.Models[0]
	}

	// 4. Features técnicas atômicas (não colapsa BGP, RADIUS e PPPoE!)
	knownFeatures := []string{
		"bgp", "pppoe", "radius", "vlan", "ospf", "evpn", "mpls", "bfd",
		"cgnat", "flowspec", "failover", "inband", "outband", "tl1",
		"l2vpn", "pbr", "qos", "acl",
	}
	for _, kf := range knownFeatures {
		if hasToken(lower, kf) {
			req.Features = append(req.Features, strings.ToUpper(kf))
		}
	}
	// "ha" é uma palavra comum (what, há, ha): só vale a sigla ou a expressão.
	if requestsHA(query) {
		req.Features = append(req.Features, "HA")
	}
	if len(req.Features) > 0 {
		req.Feature = req.Features[0]
	}

	return req
}

// ResolveClaimStatus avalia o status epistemológico e a aplicabilidade de uma claim.
//
// Regras (cada evidência é avaliada isoladamente; flags de evidências diferentes
// nunca são combinadas):
//   - Só evidências com Stance "supports" sustentam ou corroboram. "mentions" é
//     recuperação lexical, não prova, e nunca conta como origem.
//   - Uma evidência está "escopada" quando ela mesma comprova fabricante, modelo,
//     firmware e recurso (campos estruturados; notas de texto são ignoradas).
//   - sustained/applicable: uma evidência oficial que é, ela própria, escopada.
//   - corroborated/applicable: >= 2 origens independentes, cada uma com
//     evidência escopada (a origem oficial escopada já basta pela regra acima).
//   - corroborated/indeterminate: >= 2 origens independentes que sustentam, mas
//     sem escopo comprovado.
//   - documented/indeterminate: há documento oficial que sustenta, sem escopo.
//   - Uma refutação só vale quando trata do fabricante e do recurso pedidos:
//     refutação fora desse escopo é ignorada (não contesta nem incompatibiliza).
//     contested/incompatible exige ainda modelo e firmware; com fabricante e
//     recurso apenas, contested/indeterminate.
//   - unconfirmed/indeterminate: o restante, inclusive só menções.
//
// Três cópias do mesmo texto são uma única origem (ClusterTracker).
func (e *Engine) ResolveClaimStatus(claim *domain.Claim, evidences []domain.Evidence) {
	claim.Status = "unconfirmed"
	claim.Applicability = "indeterminate"
	if len(evidences) == 0 {
		return
	}

	hasContradiction := false // refutação do fabricante+recurso pedidos
	fullyRefuted := false     // refutação que também casa modelo e firmware
	hasOfficialSupport := false
	officialScoped := false
	hasMatchingSupport := false // suporte que casa fabricante+recurso pedidos

	var supportFP, supportDom []string
	var scopedFP, scopedDom []string
	var firstSupportID, firstRefuteID string

	for _, ev := range evidences {
		switch ev.Stance {
		case "refutes":
			if ev.VendorMatch && ev.FeatureMatch {
				hasContradiction = true
				if firstRefuteID == "" {
					firstRefuteID = ev.ID
				}
				if ev.ModelMatch && ev.FirmwareMatch {
					fullyRefuted = true
				}
			}
			continue
		case "supports":
			if ev.VendorMatch && ev.FeatureMatch {
				hasMatchingSupport = true
				if firstSupportID == "" {
					firstSupportID = ev.ID
				}
			}
		default: // "mentions" e qualquer outro valor: relevante, mas não é prova
			continue
		}

		fp := ev.ID
		if ev.ClusterID != nil && *ev.ClusterID != "" {
			fp = *ev.ClusterID
		}
		dom := ev.ID
		if strings.HasPrefix(ev.Notes, "Fonte: ") {
			if parts := strings.Fields(ev.Notes); len(parts) >= 2 {
				dom = parts[1]
			}
		}
		// Uma fonte só pode corroborar a consulta quando o suporte se refere
		// ao fabricante e ao recurso consultados. Outros suportes podem ser
		// documentação relacionada, mas não são votos para esta afirmação.
		if ev.VendorMatch && ev.FeatureMatch {
			supportFP, supportDom = append(supportFP, fp), append(supportDom, dom)
		}

		scoped := ev.VendorMatch && ev.ModelMatch && ev.FirmwareMatch && ev.FeatureMatch
		if scoped {
			scopedFP, scopedDom = append(scopedFP, fp), append(scopedDom, dom)
		}
		if ev.IsOfficialDoc {
			hasOfficialSupport = true
			if scoped {
				officialScoped = true
			}
		}
	}

	if hasContradiction && hasMatchingSupport && firstSupportID != "" && firstRefuteID != "" && len(claim.Contradictions) == 0 {
		claim.Contradictions = append(claim.Contradictions, domain.Contradiction{
			ID:          fmt.Sprintf("contradiction-%s-%s", firstSupportID, firstRefuteID),
			ClaimID:     claim.ID,
			EvidenceAID: firstSupportID,
			EvidenceBID: firstRefuteID,
			Explanation: "Conflito explícito entre evidências de suporte e refutação para o escopo consultado",
		})
	}

	switch {
	case fullyRefuted && !hasMatchingSupport:
		claim.Status = "contested"
		claim.Applicability = "incompatible"
	case hasContradiction:
		// Regra 9 (Seção 10.3): conflito entre suporte e refutação permanece
		// contestado/indeterminado, não se converte automaticamente em incompatible.
		claim.Status = "contested"
		claim.Applicability = "indeterminate"
	case officialScoped:
		claim.Status = "sustained"
		claim.Applicability = "applicable"
	case e.tracker.CountIndependentOrigins(scopedFP, scopedDom) >= 2:
		claim.Status = "corroborated"
		claim.Applicability = "applicable"
	case hasOfficialSupport:
		// Manual legítimo, sem comprovar o escopo (modelo/firmware/recurso) pedido.
		claim.Status = "documented"
	case e.tracker.CountIndependentOrigins(supportFP, supportDom) >= 2:
		claim.Status = "corroborated"
	}
}

var (
	// Trechos que só listam assuntos ou declaram a ausência de conteúdo.
	nonAssertiveRE = regexp.MustCompile(`(?i)\b(index of|table of contents|contents:|sitemap|title only|no (configuration|compatibility|command|support|example)s?\b|not (documented|described|covered))`)
	// Verbos e marcadores de afirmação técnica.
	// Sintaxe de configuração/CLI (o classificador nem sempre a reconhece como comando).
	configSyntaxRE = regexp.MustCompile(`(?i)(\bpeer\s+\S+\s+as-number\b|\bas-number\s+\d+|\bbgp\s+\d{1,10}\b|\brouter\s+bgp\s+\d+|(^|\s)/(routing|interface|ip|system)\s+\w+|\bip\s+route(-static)?\s+\S+|\b(display|show|undo)\s+\w+)`)
	assertiveRE    = regexp.MustCompile(`(?i)\b(supports?|supported|configure[sd]?|configuring|enable[sd]?|use the command|run the command|is available|can be (configured|used|enabled)|example|procedure|steps?|suporta\p{L}*|suportad[oa]s?|configura\p{L}*|habilita\p{L}*|dispon[íi]vel|soporta\p{L}*|soportad[oa]s?|admite\p{L}*|admitid[oa]s?|disponibles?)\b`)
)

// assertsContent informa se o trecho afirma algo (comando CLI ou frase de
// suporte/procedimento), em vez de apenas conter as palavras da consulta.
func assertsContent(lowerSnippet string, hasCLI bool) bool {
	if nonAssertiveRE.MatchString(lowerSnippet) {
		return false
	}
	return hasCLI || configSyntaxRE.MatchString(lowerSnippet) || assertiveRE.MatchString(lowerSnippet)
}

// BuildClaimsFromPassages analisa as passagens de uma campanha e sintetiza claims atômicas estruturadas.
func (e *Engine) BuildClaimsFromPassages(campaignID string, query string, passages []domain.SearchResult) []domain.Claim {
	var claims []domain.Claim
	req := ParseRequirements(query)
	claimsByKey := make(map[string]*domain.Claim)

	for _, p := range passages {
		classification := classifier.Classify(p.Snippet, p.Domain)

		// Fabricante da passagem: prioriza o fabricante requerido quando verificado,
		// evitando que um fabricante secundário citado no texto sequestre o sujeito.
		vendorName := req.Vendor
		if vendorName == "" && len(classification.Vendors) > 0 {
			vendorName = classification.Vendors[0].Name
		}

		// Avaliação oracional vinculando sujeito (fabricante/modelo/firmware),
		// predicado (afirmação/negação) e objeto (recursos pedidos).
		assertion := analyzePassage(p.Snippet, req, classification.CLICommands)
		vendorMatch := assertion.VendorMatch
		modelMatch := assertion.ModelMatch
		firmwareMatch := assertion.FirmwareMatch
		featureMatch := assertion.CoAsserted || assertion.Refuted

		// Se a consulta exige fabricante/firmware e a oração assertiva/refutativa
		// não pertence a ele, a passagem é apenas menção fora de escopo.
		vendorMismatch := len(req.Vendors) > 0 && !vendorMatch
		versionMismatch := false
		lowerSnippet := strings.ToLower(p.Snippet)
		for _, fw := range req.Firmwares {
			if mentionsConflictingFirmware(lowerSnippet, fw) && !firmwareMatch {
				versionMismatch = true
			}
		}

		stance := "mentions"
		switch {
		case vendorMismatch || versionMismatch:
			stance = "mentions"
		case assertion.Conflicted:
			// Conflito explícito dentro do próprio trecho: registra a contestação,
			// mas sem permitir que vire incompatibilidade unilateral.
			stance = "refutes"
			modelMatch = false
			firmwareMatch = false
		case assertion.Refuted:
			stance = "refutes"
		case assertion.CoAsserted:
			stance = "supports"
		}

		// Identifica o tópico atômico do claim de forma consistente por consulta,
		// garantindo que evidências de suporte e refutação da mesma consulta sejam
		// confrontadas no mesmo claim.
		claimTitle := query
		claimStatement := fmt.Sprintf("Procedimento e diretrizes documentadas para %s", query)
		if vendorName != "" {
			claimTitle = fmt.Sprintf("%s — %s", vendorName, query)
		}
		if query == "" && len(classification.CLICommands) > 0 {
			firstCmd := classification.CLICommands[0]
			claimTitle = fmt.Sprintf("Configuração CLI: %s", firstCmd)
			claimStatement = fmt.Sprintf("Comandos CLI citados nas fontes para o contexto %s (não validados)", vendorName)
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
			VendorMatch:   vendorMatch,
			ModelMatch:    modelMatch,
			FirmwareMatch: firmwareMatch,
			FeatureMatch:  featureMatch,
			SourceSpan:    fmt.Sprintf("doc:%s:p:%d", p.DocumentID, p.PassageID),
		}
		claim.Evidences = append(claim.Evidences, ev)
	}

	for _, claim := range claimsByKey {
		e.ResolveClaimStatus(claim, claim.Evidences)
		claims = append(claims, *claim)
	}
	return claims
}
