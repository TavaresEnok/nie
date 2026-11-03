package evidence

import (
	"regexp"
	"strings"

	"nie/internal/classifier"
)

// Análise por oração e segmento coordenado (auditorias V5 e V6).
// Garante vinculação estrita entre sujeito (fabricante/modelo/firmware),
// predicado (afirmação/negação) e objeto (recurso/protocolo):
//   1. Suporte de outro fabricante/modelo/firmware na mesma passagem nunca é
//      atribuído ao fabricante/modelo/firmware consultado.
//   2. Negação de um recurso (ex.: BGP ou PPPoE) nunca refuta outro recurso
//      (ex.: OSPF ou BGP), mesmo quando coordenados na mesma frase.
//   3. Recursos pedidos em conjunto exigem co-afirmação no mesmo escopo.
//   4. Conflito explícito entre afirmação e refutação no mesmo escopo é
//      preservado como conflito (Conflicted), sem virar incompatibilidade.

// clauseSplitRE separa orações: fim de frase (pontuação seguida de espaço ou
// fim do texto, o que preserva versões e IPs como "v8.2" e "10.0.0.2"),
// quebras de linha e conjunções adversativas/subordinadas em EN/PT/ES.
var clauseSplitRE = regexp.MustCompile(`(?i)(?:[.!?;]+(?:\s+|$)|\n+|,?\s+(?:but|however|whereas|although|while|whilst|yet|mas|porém|porem|contudo|entretanto|embora|enquanto|pero|sin embargo|aunque|mientras)\s+)`)

func splitClauses(snippet string) []string {
	var out []string
	for _, c := range clauseSplitRE.Split(snippet, -1) {
		if c = strings.TrimSpace(c); c != "" {
			out = append(out, c)
		}
	}
	return out
}

// passageAssertion resume o que o trecho afirma sobre os recursos e o escopo pedidos.
type passageAssertion struct {
	// CoAsserted: uma mesma oração/segmento afirma, sem negação nem condição,
	// todos os recursos pedidos para o sujeito (fabricante/modelo/firmware) alvo.
	CoAsserted bool
	// Refuted: uma oração/segmento do sujeito alvo nega ou declara obsoleto/
	// incompatível ao menos um dos recursos pedidos.
	Refuted bool
	// Conflicted: o trecho apresenta afirmação e refutação concorrentes para o
	// mesmo recurso no escopo pedido, ou declara conflito explícito entre fontes.
	Conflicted bool
	// Escopo verificado na oração/passagem pertencente ao sujeito alvo.
	VendorMatch   bool
	ModelMatch    bool
	FirmwareMatch bool
}

var allKnownVendors = []struct {
	pattern string
	name    string
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
	{"nokia", "Nokia"},
	{"arista", "Arista"},
}

var allKnownModels = []string{
	"ne8000", "ne40e", "ccr2004", "ccr1036", "ccr2116", "rb4011", "rb5009",
	"asr9000", "asr1000", "dm4270", "dm4370", "dm4000", "an5516", "an6000",
	"c320", "c600", "ma5800", "ma5608t", "nsa2700",
}

var allKnownFeatures = []string{
	"bgp", "pppoe", "radius", "vlan", "ospf", "evpn", "mpls", "bfd",
	"cgnat", "flowspec", "failover", "inband", "outband", "tl1",
	"l2vpn", "pbr", "qos", "acl", "ha", "telnet", "ssh", "snmp", "vrrp",
	"isis", "ldp", "rsvp", "nat", "ipsec", "gre", "lacp", "stp", "rstp",
	"igmp", "pim", "dhcp", "netconf", "gpon", "xgs-pon",
}

// coordConjRE localiza conjunções coordenativas ("and", "e", "y") ou vírgulas
// que podem separar predicados distintos na mesma oração.
var coordConjRE = regexp.MustCompile(`(?i)(?:,\s*(?:and|e|y)\s+|\s+(?:and|e|y)\s+|,\s+)`)

// contrastiveNegSplitRE separa adjuntos negativos contrastivos ao final da frase
// (ex.: "supports BGP without PPPoE", "supports BGP, not PPPoE").
var contrastiveNegSplitRE = regexp.MustCompile(`(?i)(?:,\s*(?:not|não|nao|no)\s+|\s+(?:without|lacking|sem|sin)\s+)`)

// simpleCoordBetweenRE verifica se dois recursos estão diretamente coordenados
// por conjunção ("and", "or", "nor", "e", "ou", "nem", "y", "o", "ni", "/", ",").
var simpleCoordBetweenRE = regexp.MustCompile(`(?i)^\s*(?:,|/|(?:,\s*)?(?:and|or|nor|as well as|e|ou|nem|y|o|ni))\s*$`)

// contrastivePrefixRE identifica se uma menção a fabricante/modelo está em um
// adjunto comparativo/contrastivo ("unlike Huawei", "compared to Huawei", etc.).
var contrastivePrefixRE = regexp.MustCompile(`(?i)(?:unlike|compared\s+(?:to|with)|versus|vs\.?|instead\s+of|rather\s+than|from|than|ao\s+contr[áa]rio\s+d[eo]|comparado\s+(?:a|ao|com)|em\s+vez\s+de|a\s+diferencia\s+de|comparado\s+con|en\s+lugar\s+de)\s+$`)

func hasPredicate(lower string) bool {
	return snippetRefutes(lower) || activeNegationRE.MatchString(lower) || passiveNegationRE.MatchString(lower) ||
		configSyntaxRE.MatchString(lower) || assertiveRE.MatchString(lower)
}

// splitCoordinateSegments divide uma oração em segmentos coordenados quando
// ambos os lados possuem seu próprio predicado (ex.: "supports OSPF and does
// not support BGP") ou quando há um adjunto negativo ("without PPPoE", ", not PPPoE").
// Mantém intactas coordenações nominais sob um mesmo verbo ("supports BGP and PPPoE").
func splitCoordinateSegments(clause string) []string {
	// 1. Separa adjuntos negativos contrastivos ("... supports BGP without PPPoE")
	var stage1 []string
	locs := contrastiveNegSplitRE.FindAllStringIndex(clause, -1)
	if len(locs) == 0 {
		stage1 = []string{clause}
	} else {
		prev := 0
		for _, loc := range locs {
			left := strings.TrimSpace(clause[prev:loc[0]])
			if left != "" {
				stage1 = append(stage1, left)
			}
			prev = loc[0]
		}
		tail := strings.TrimSpace(clause[prev:])
		if tail != "" {
			stage1 = append(stage1, tail)
		}
	}

	// 2. Separa coordenações verbais ("... supports OSPF and does not support BGP")
	var out []string
	for _, part := range stage1 {
		out = append(out, splitByVerbCoordination(part)...)
	}
	return out
}

func splitByVerbCoordination(s string) []string {
	locs := coordConjRE.FindAllStringIndex(s, -1)
	if len(locs) == 0 {
		return []string{s}
	}
	for _, loc := range locs {
		left := strings.TrimSpace(s[:loc[0]])
		right := strings.TrimSpace(s[loc[1]:])
		if left != "" && right != "" && hasPredicate(strings.ToLower(left)) && hasPredicate(strings.ToLower(right)) {
			var res []string
			res = append(res, splitByVerbCoordination(left)...)
			res = append(res, splitByVerbCoordination(right)...)
			return res
		}
	}
	return []string{s}
}

// tokenIndex devolve o índice inicial de tok em lower respeitando limites de palavra, ou -1.
func tokenIndex(lower, tok string) int {
	if tok == "" {
		return -1
	}
	for from := 0; from < len(lower); {
		j := strings.Index(lower[from:], tok)
		if j < 0 {
			return -1
		}
		j += from
		end := j + len(tok)
		before := j == 0 || !isAlnumByte(lower[j-1])
		after := end == len(lower) || !isLetterByte(lower[end]) ||
			(lower[end] == 's' && (end+1 == len(lower) || !isLetterByte(lower[end+1]))) ||
			(lower[end] == 'v' && end+1 < len(lower) && lower[end+1] >= '0' && lower[end+1] <= '9')
		if before && after {
			return j
		}
		from = j + 1
	}
	return -1
}

func featureIndex(feature, snippet, lowerSnippet string) int {
	if strings.EqualFold(feature, "HA") {
		if loc := haAcronymRE.FindStringIndex(snippet); loc != nil {
			return loc[0]
		}
		if loc := haPhraseRE.FindStringIndex(lowerSnippet); loc != nil {
			return loc[0]
		}
		return -1
	}
	return tokenIndex(lowerSnippet, strings.ToLower(feature))
}

// isFeatureNegatedInSegment avalia se a negação existente no segmento recai
// especificamente sobre o recurso `feature` (e não apenas sobre outro recurso).
func isFeatureNegatedInSegment(seg, lowerSeg, feature string) bool {
	fPos := featureIndex(feature, seg, lowerSeg)
	if fPos < 0 {
		return false
	}

	// Negação curta direta ("without BGP", "not BGP", "sem BGP", "no BGP")
	shortNegRE := regexp.MustCompile(`(?i)(?:^|[,\s])\b(?:without|lacks?|lacking|not|no|sem|não|nao|sin)\s+(?:support\s+for\s+|suporte\s+(?:a|ao|para)\s+|soporte\s+para\s+)?` + regexp.QuoteMeta(strings.ToLower(feature)) + `\b`)
	if shortNegRE.MatchString(lowerSeg) {
		return true
	}

	if !snippetRefutes(lowerSeg) {
		return false
	}

	// Coleta posições de outros recursos técnicos presentes no mesmo segmento.
	type otherFeatPos struct {
		name string
		pos  int
	}
	var others []otherFeatPos
	for _, kf := range allKnownFeatures {
		if strings.EqualFold(kf, feature) {
			continue
		}
		if pos := featureIndex(kf, seg, lowerSeg); pos >= 0 {
			others = append(others, otherFeatPos{name: kf, pos: pos})
		}
	}
	if len(others) == 0 {
		return true
	}

	// 1. Verifica negação passiva ("PPPoE is not supported with BGP"):
	// na voz passiva, o recurso negado antecede o verbo passivo.
	if loc := passiveNegationRE.FindStringIndex(lowerSeg); loc != nil {
		negStart := loc[0]
		// Se outro recurso está antes do verbo passivo e `feature` está depois,
		// o sujeito negado é o outro recurso, não `feature`.
		otherBefore := false
		for _, o := range others {
			if o.pos < negStart {
				otherBefore = true
				break
			}
		}
		if otherBefore && fPos >= negStart {
			return false
		}
		// Se ambos estão antes do verbo passivo, mas `feature` vem antes em um
		// adjunto preposicional ("With BGP, PPPoE is not supported") e não
		// coordenado por "and/or/nor" com o sujeito imediato:
		if fPos < negStart {
			for _, o := range others {
				if o.pos > fPos && o.pos < negStart {
					between := lowerSeg[fPos+len(feature) : o.pos]
					if !simpleCoordBetweenRE.MatchString(between) {
						return false
					}
				}
			}
		}
		return true
	}

	// 2. Verifica negação ativa ("does not support PPPoE over BGP"):
	// na voz ativa, o recurso negado sucede o verbo negativo.
	if loc := activeNegationRE.FindStringIndex(lowerSeg); loc != nil {
		negStart, negEnd := loc[0], loc[1]
		otherAfter := false
		for _, o := range others {
			if o.pos >= negStart {
				otherAfter = true
				break
			}
		}
		// Se `feature` vem antes da negação ativa e outro recurso vem depois
		// ("With OSPF, Huawei does not support BGP"), `feature` não é o alvo.
		if fPos < negStart && otherAfter {
			return false
		}
		// Se ambos vêm depois da negação ativa, mas outro recurso é o objeto
		// imediato e `feature` vem após preposição ("does not support PPPoE over BGP"):
		if fPos >= negEnd {
			for _, o := range others {
				if o.pos >= negStart && o.pos < fPos {
					between := lowerSeg[o.pos+len(o.name) : fPos]
					if !simpleCoordBetweenRE.MatchString(between) {
						return false
					}
				}
			}
		}
		return true
	}

	// 3. Palavras de obsolescência/erro (deprecated, obsolete, incompatible):
	if loc := refuteKeywordRE.FindStringIndex(lowerSeg); loc != nil {
		negStart := loc[0]
		for _, o := range others {
			if o.pos < negStart && fPos > negStart {
				return false
			}
			if o.pos > fPos && o.pos < negStart {
				between := lowerSeg[fPos+len(feature) : o.pos]
				if !simpleCoordBetweenRE.MatchString(between) {
					return false
				}
			}
		}
	}

	return true
}

// isFeatureAffirmedInSegment verifica se `feature` é positivamente afirmado no segmento.
func isFeatureAffirmedInSegment(seg, lowerSeg, feature string, hasCLI bool) bool {
	if !snippetHasFeature(feature, seg, lowerSeg) {
		return false
	}
	if isFeatureNegatedInSegment(seg, lowerSeg, feature) {
		return false
	}
	if snippetIsQualified(lowerSeg) {
		return false
	}
	// Se há uma negação no mesmo segmento (dirigida a outro recurso), remove os
	// predicados negativos antes de testar se resta um verbo afirmativo para `feature`.
	cleaned := negatedSupportRE.ReplaceAllString(lowerSeg, " ")
	cleaned = passiveNegationRE.ReplaceAllString(cleaned, " ")
	cleaned = activeNegationRE.ReplaceAllString(cleaned, " ")
	cleaned = refuteKeywordRE.ReplaceAllString(cleaned, " ")
	return assertsContent(cleaned, hasCLI)
}

// clauseHasForeignSubject verifica se a oração/segmento atribui a ação a outro
// fabricante, outro modelo ou outro firmware diferente do solicitado em `req`.
func clauseHasForeignSubject(lower string, req TargetRequirements) (hasTargetVendor, hasForeignVendor, hasTargetModel, hasForeignModel, hasTargetFW, hasForeignFW bool) {
	if len(req.Vendors) > 0 {
		targetSet := make(map[string]bool, len(req.Vendors))
		for _, v := range req.Vendors {
			targetSet[strings.ToLower(v)] = true
		}
		firstTargetPos := -1
		targetInContrast := false
		firstForeignPos := -1
		for _, kv := range allKnownVendors {
			pos := tokenIndex(lower, kv.pattern)
			if pos < 0 {
				continue
			}
			if targetSet[strings.ToLower(kv.name)] || targetSet[kv.pattern] {
				hasTargetVendor = true
				if firstTargetPos < 0 || pos < firstTargetPos {
					firstTargetPos = pos
				}
				prefix := lower[:pos]
				if contrastivePrefixRE.MatchString(prefix) {
					targetInContrast = true
				}
			} else {
				hasForeignVendor = true
				if firstForeignPos < 0 || pos < firstForeignPos {
					firstForeignPos = pos
				}
			}
		}
		// Se ambos aparecem no mesmo segmento (ex.: "Compared to Huawei NE8000, Cisco supports BGP"
		// ou "Huawei NE8000 interop: Cisco supports BGP"), verifica se o fabricante estrangeiro
		// é o verdadeiro sujeito do predicado.
		if hasTargetVendor && hasForeignVendor {
			if targetInContrast || firstForeignPos > firstTargetPos {
				hasTargetVendor = false
			}
		}
	}

	if len(req.Models) > 0 {
		targetModels := make(map[string]bool, len(req.Models))
		for _, m := range req.Models {
			targetModels[strings.ToLower(m)] = true
		}
		for _, km := range allKnownModels {
			if tokenIndex(lower, km) >= 0 {
				if targetModels[km] {
					hasTargetModel = true
				} else {
					hasForeignModel = true
				}
			}
		}
		if hasTargetModel && hasForeignModel && hasForeignVendor && !hasTargetVendor {
			hasTargetModel = false
		}
	}

	if len(req.Firmwares) > 0 {
		for _, fw := range req.Firmwares {
			if mentionsFirmware(lower, fw) {
				hasTargetFW = true
			}
			if mentionsConflictingFirmware(lower, fw) {
				hasForeignFW = true
			}
		}
	}

	return
}

// analyzeAssertion mantém compatibilidade com chamadas legadas que informam apenas features.
func analyzeAssertion(snippet string, features, cliCommands []string) passageAssertion {
	return analyzePassage(snippet, TargetRequirements{Features: features}, cliCommands)
}

// analyzePassage avalia o trecho oração por oração e segmento por segmento,
// vinculando fabricante, modelo, firmware e recursos na mesma proposição.
func analyzePassage(snippet string, req TargetRequirements, cliCommands []string) passageAssertion {
	var a passageAssertion
	lowerSnippet := strings.ToLower(snippet)

	// Verifica no trecho inteiro se há múltiplos fabricantes/modelos/firmwares
	_, passageForeignVendor, _, passageForeignModel, _, passageForeignFW := clauseHasForeignSubject(lowerSnippet, req)
	singleEntityPassage := !passageForeignVendor && !passageForeignModel && !passageForeignFW

	// Em passagens de entidade única, o escopo explícito no trecho vale para suas orações.
	var passageVendorOK, passageModelOK, passageFirmwareOK bool
	if len(req.Vendors) > 0 {
		cl := classifier.Classify(snippet, "")
		for _, v := range req.Vendors {
			if hasToken(lowerSnippet, strings.ToLower(v)) {
				passageVendorOK = true
				break
			}
			if singleEntityPassage {
				for _, dv := range cl.Vendors {
					if strings.EqualFold(dv.Name, v) {
						passageVendorOK = true
						break
					}
				}
			}
		}
	}
	if len(req.Models) > 0 {
		for _, m := range req.Models {
			if hasToken(lowerSnippet, strings.ToLower(m)) {
				passageModelOK = true
				break
			}
		}
	}
	if len(req.Firmwares) > 0 {
		for _, fw := range req.Firmwares {
			if mentionsFirmware(lowerSnippet, fw) && !mentionsConflictingFirmware(lowerSnippet, fw) {
				passageFirmwareOK = true
				break
			}
		}
	}

	if singleEntityPassage {
		a.VendorMatch = passageVendorOK
		a.ModelMatch = passageModelOK
		a.FirmwareMatch = passageFirmwareOK
	}

	if len(req.Features) == 0 {
		return a
	}

	// Contexto acumulado entre orações para pronomes/sujeito oculto.
	ctxVendorOK := singleEntityPassage && passageVendorOK
	ctxModelOK := singleEntityPassage && passageModelOK
	ctxFirmwareOK := singleEntityPassage && passageFirmwareOK
	ctxForeignActive := false

	for _, clause := range splitClauses(snippet) {
		lowerClause := strings.ToLower(clause)
		cTargetV, cForeignV, cTargetM, cForeignM, cTargetFW, cForeignFW := clauseHasForeignSubject(lowerClause, req)

		if cForeignV && !cTargetV {
			ctxForeignActive = true
			ctxVendorOK = false
			ctxModelOK = false
			ctxFirmwareOK = false
		} else if cTargetV {
			ctxForeignActive = false
			ctxVendorOK = true
		}
		if cForeignM && !cTargetM {
			ctxModelOK = false
		} else if cTargetM && !ctxForeignActive {
			ctxModelOK = true
		}
		if cForeignFW && !cTargetFW {
			ctxFirmwareOK = false
		} else if cTargetFW && !ctxForeignActive {
			ctxFirmwareOK = true
		}

		for _, seg := range splitCoordinateSegments(clause) {
			lowerSeg := strings.ToLower(seg)
			sTargetV, sForeignV, sTargetM, sForeignM, sTargetFW, sForeignFW := clauseHasForeignSubject(lowerSeg, req)

			// Se o segmento trata de outro fabricante, modelo ou release, não prova nem refuta o alvo.
			if (len(req.Vendors) > 0 && ((sForeignV && !sTargetV) || (!sTargetV && ctxForeignActive))) ||
				(len(req.Models) > 0 && sForeignM && !sTargetM) ||
				(len(req.Firmwares) > 0 && sForeignFW && !sTargetFW) {
				continue
			}

			segVendorOK := sTargetV || (!sForeignV && (cTargetV || ctxVendorOK || (singleEntityPassage && passageVendorOK)))
			segModelOK := sTargetM || (!sForeignM && (cTargetM || ctxModelOK || (singleEntityPassage && passageModelOK)))
			segFirmwareOK := sTargetFW || (!sForeignFW && (cTargetFW || ctxFirmwareOK || (singleEntityPassage && passageFirmwareOK)))

			// Em passagens com múltiplos fabricantes, exige vínculo com o fabricante pedido.
			if len(req.Vendors) > 0 && !singleEntityPassage && !segVendorOK {
				continue
			}

			mentionsAny := false
			for _, f := range req.Features {
				if snippetHasFeature(f, seg, lowerSeg) {
					mentionsAny = true
					break
				}
			}
			if !mentionsAny {
				continue
			}

			hasCLI := false
			for _, cmd := range cliCommands {
				if cmd != "" && strings.Contains(lowerSeg, strings.ToLower(cmd)) {
					hasCLI = true
					break
				}
			}

			// Verifica se algum dos recursos pedidos é especificamente negado neste segmento.
			segRefutesRequested := false
			for _, f := range req.Features {
				if isFeatureNegatedInSegment(seg, lowerSeg, f) {
					segRefutesRequested = true
					break
				}
			}
			if segRefutesRequested {
				a.Refuted = true
				if segVendorOK {
					a.VendorMatch = true
				}
				if segModelOK {
					a.ModelMatch = true
				}
				if segFirmwareOK {
					a.FirmwareMatch = true
				}
				continue
			}

			// Verifica se TODOS os recursos pedidos são co-afirmados neste segmento.
			if snippetIsQualified(lowerClause) && len(splitCoordinateSegments(clause)) == 1 {
				continue
			}
			allAffirmed := true
			for _, f := range req.Features {
				if !isFeatureAffirmedInSegment(seg, lowerSeg, f, hasCLI) {
					allAffirmed = false
					break
				}
			}
			if allAffirmed {
				a.CoAsserted = true
				if segVendorOK {
					a.VendorMatch = true
				}
				if segModelOK {
					a.ModelMatch = true
				}
				if segFirmwareOK {
					a.FirmwareMatch = true
				}
			}
		}
	}

	// Se o mesmo trecho tanto afirma quanto refuta o recurso para o sujeito alvo,
	// ou se declara explicitamente conflito/contradição sobre uma afirmação/refutação,
	// preserva o estado de conflito (Regra 9 da Seção 10.3).
	if (a.CoAsserted && a.Refuted) || (conflictMarkerRE.MatchString(lowerSnippet) && (a.CoAsserted || a.Refuted)) {
		a.Conflicted = true
	}

	return a
}

// firmwareAliases lista as grafias aceitas como menção explícita de cada
// release. O firmware nunca é inferido a partir do modelo: "NE8000" sozinho
// não prova "VRP v8".
var firmwareAliases = map[string][]string{
	"VRP v8":      {"vrp v8", "vrpv8", "vrp-v8", "vrp8", "vrp v800", "v800r"},
	"VRP v5":      {"vrp v5", "vrpv5", "vrp-v5", "vrp5"},
	"RouterOS v7": {"routeros v7", "routeros 7", "routerosv7", "ros v7", "ros7"},
	"RouterOS v6": {"routeros v6", "routeros 6", "routerosv6", "ros v6", "ros6"},
	"IOS-XR":      {"ios-xr", "ios xr", "iosxr"},
	"Junos":       {"junos"},
	"TiMOS":       {"timos"},
}

// conflictingFirmware: releases que, se citadas explicitamente, indicam que o
// trecho trata de outra versão do mesmo sistema.
var conflictingFirmware = map[string]string{
	"VRP v8":      "VRP v5",
	"VRP v5":      "VRP v8",
	"RouterOS v7": "RouterOS v6",
	"RouterOS v6": "RouterOS v7",
}

// mentionsFirmware informa se o trecho cita explicitamente a release.
func mentionsFirmware(lowerSnippet, firmware string) bool {
	aliases, ok := firmwareAliases[firmware]
	if !ok {
		aliases = []string{strings.ToLower(firmware)}
	}
	for _, a := range aliases {
		if hasToken(lowerSnippet, a) {
			return true
		}
	}
	return false
}

// mentionsConflictingFirmware informa se o trecho cita uma release diferente
// da pedida (e não cita a pedida).
func mentionsConflictingFirmware(lowerSnippet, firmware string) bool {
	other, ok := conflictingFirmware[firmware]
	return ok && mentionsFirmware(lowerSnippet, other) && !mentionsFirmware(lowerSnippet, firmware)
}
