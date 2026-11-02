package extract

import (
	"regexp"
	"sort"
	"strings"
)

// MentionPattern representa uma regra de extração de termos técnicos sem hyperlink.
type MentionPattern struct {
	Category string
	Regexp   *regexp.Regexp
}

var mentionPatterns = []MentionPattern{
	// Referências genéricas, válidas para qualquer assunto público (o vocabulário
	// de telecom abaixo é uma especialização, não a base da descoberta).
	{
		Category: "cve",
		Regexp:   regexp.MustCompile(`(?i)\bCVE-\d{4}-\d{4,7}\b`),
	},
	{
		Category: "doi",
		Regexp:   regexp.MustCompile(`\b10\.\d{4,9}/[-._;()/:A-Za-z0-9]+[A-Za-z0-9]`),
	},
	{
		Category: "arxiv",
		Regexp:   regexp.MustCompile(`(?i)\barXiv:\s?\d{4}\.\d{4,5}(?:v\d+)?\b`),
	},
	{
		Category: "standard",
		Regexp:   regexp.MustCompile(`\b(?:ISO(?:/IEC)?\s?\d{3,5}(?:-\d+)?(?::\d{4})?|IEC\s?\d{3,5}(?:-\d+)*|IEEE\s?\d{3,4}(?:\.\d+[a-z]*)*|ITU-T\s?[A-Z]\.\d{1,4}(?:\.\d+)?|NIST\s?SP\s?\d{3}-\d{1,3}[A-Z]?|ABNT\s?NBR\s?\d{3,5}(?:-\d+)?)\b`),
	},
	// RFCs e Especificações de Engenharia de Internet
	{
		Category: "rfc",
		Regexp:   regexp.MustCompile(`(?i)\bRFC\s*([0-9]{3,5})\b`),
	},
	// Roteadores e Equipamentos de Borda / Core / Acesso
	{
		Category: "model_huawei",
		Regexp:   regexp.MustCompile(`(?i)\b(?:NetEngine\s*8000|NE8000|NE40E|NE20E|MA5800(?:-X\d+)?|MA5608T|MA5600T|CX600|S6730[\w-]*)\b`),
	},
	{
		Category: "model_mikrotik",
		Regexp:   regexp.MustCompile(`(?i)\b(?:CCR2004(?:-[0-9a-z-]+)?(?:\+)?|CCR2116(?:-[0-9a-z-]+)?(?:\+)?|CCR1036(?:-[0-9a-z-]+)?(?:\+)?|CCR1072|CCR2216|CRS326[\w-]*|CRS328[\w-]*|CRS317[\w-]*|RB5009[\w-]*|RB4011[\w-]*)`),
	},
	{
		Category: "model_cisco",
		Regexp:   regexp.MustCompile(`(?i)\b(?:ASR\s*(?:9000|9010|9006|9904|1001-X|1002-X|1000)|NCS\s*(?:5500|540|500)|Catalyst\s*(?:9300|9500)|Nexus\s*(?:9300|3000))\b`),
	},
	{
		Category: "model_datacom",
		Regexp:   regexp.MustCompile(`(?i)\b(?:DM4270|DM4380|DM4000|DM4100|DM4250|DM4770|DM2100|DM2300)\b`),
	},
	{
		Category: "model_gpon_olt",
		Regexp:   regexp.MustCompile(`(?i)\b(?:ZXA10\s*(?:C300|C320|C600|C650)|C320|C300|C600|AN5516(?:-[\w]+)?|V1600(?:G\d|D\d)?)\b`),
	},
	{
		Category: "model_nokia_juniper",
		Regexp:   regexp.MustCompile(`(?i)\b(?:7750\s*SR(?:-[\w]+)?|7250\s*IXR|MX960|MX480|MX204|PTX1000(?:1|2)?)\b`),
	},
	// Protocolos e Arquiteturas Críticas de Telecom
	{
		Category: "protocol",
		Regexp:   regexp.MustCompile(`(?i)\b(?:BGP\s+(?:EVPN|confederation|add-path|flowspec|route-reflector|communities)|EVPN-VXLAN|Segment\s+Routing(?:\s+SRv6)?|PPPoE\s+(?:dual-stack|BRAS|BNG)|MPLS\s+(?:L2VPN|L3VPN|VPLS|SR)|Option\s+82|RPKI\s+ROA|BFD\s+(?:multihop|single-hop)|DHCP\s+snooping)\b`),
	},
	// Comandos de Sintaxe Operacional (CLI)
	{
		Category: "cli_command",
		Regexp:   regexp.MustCompile(`(?i)\b(?:(?:display|show)\s+(?:bgp(?:\s+vpnv4)?|ip\s+routing-table|interface\s+description|pppoe-server|mpls\s+lsp)|ip\s+route-static\s+[0-9./]+)\b`),
	},
}

// ExtractMentions analisa o texto bruto e extrai termos técnicos representativos não linkados.
// Retorna uma lista deduplicada de termos técnicos padronizados, ordenados e limitados a maxCount.
func ExtractMentions(text string, maxCount ...int) []string {
	limit := 25
	if len(maxCount) > 0 && maxCount[0] > 0 {
		limit = maxCount[0]
	}

	if strings.TrimSpace(text) == "" {
		return nil
	}

	seen := make(map[string]struct{})
	var results []string

	for _, p := range mentionPatterns {
		matches := p.Regexp.FindAllString(text, -1)
		for _, m := range matches {
			cleaned := cleanMention(m)
			if cleaned == "" {
				continue
			}
			normKey := strings.ToLower(cleaned)
			if _, exists := seen[normKey]; !exists {
				seen[normKey] = struct{}{}
				results = append(results, cleaned)
			}
		}
	}

	// Ordena por relevância e tamanho do termo
	sort.Slice(results, func(i, j int) bool {
		// Prefer RFCs e termos mais específicos primeiro
		if strings.HasPrefix(results[i], "RFC ") && !strings.HasPrefix(results[j], "RFC ") {
			return true
		}
		if !strings.HasPrefix(results[i], "RFC ") && strings.HasPrefix(results[j], "RFC ") {
			return false
		}
		return len(results[i]) > len(results[j])
	})

	if len(results) > limit {
		results = results[:limit]
	}
	return results
}

func cleanMention(s string) string {
	s = strings.TrimSpace(s)
	// Normaliza múltiplos espaços
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	res := strings.Join(fields, " ")

	// Normaliza RFC (e.g. "rfc1997" -> "RFC 1997")
	if strings.HasPrefix(strings.ToUpper(res), "RFC") {
		num := strings.TrimSpace(res[3:])
		return "RFC " + num
	}

	// Normaliza maiúsculas em siglas comuns
	switch strings.ToLower(res) {
	case "ne8000":
		return "NE8000"
	case "ccr2004":
		return "CCR2004"
	case "dm4270":
		return "DM4270"
	case "ma5800":
		return "MA5800"
	}

	return res
}
