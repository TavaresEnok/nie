package classifier

import (
	"net"
	"net/url"
	"regexp"
	"strings"
)

type DetectedVendor struct {
	Slug       string   `json:"slug"`
	Name       string   `json:"name"`
	OS         string   `json:"os,omitempty"`
	Confidence float64  `json:"confidence"`
	Commands   []string `json:"commands,omitempty"`
}

type ClassificationResult struct {
	Vendors       []DetectedVendor `json:"vendors"`
	IsOfficialDoc bool             `json:"is_official_doc"`
	CLICommands   []string         `json:"cli_commands"`
	Tags          []string         `json:"tags"`
}

var officialDomains = []string{
	"huawei.com",
	"mikrotik.com",
	"cisco.com",
	"datacom.com.br",
	"zte.com.cn",
	"fiberhome.com",
	"vsolcn.com",
	"sonicwall.com",
	"hillstonenet.com",
}

var (
	reHuaweiCLI   = regexp.MustCompile(`(?mi)^\s*(?:display\s+[\w\-]+|system-view|peer\s+[\d\.]+\s+as-number|undo\s+[\w\-]+|commit|traffic\s+policy)`)
	reMikroTikCLI = regexp.MustCompile(`(?mi)^\s*(?:\/(?:ip|routing|interface|certificate|system|queue|radius)\b[^\n\r]+)`)
	reCiscoCLI    = regexp.MustCompile(`(?mi)^\s*(?:show\s+[\w\-]+|router\s+bgp\s+\d+|crypto\s+ikev2|interface\s+(?:GigabitEthernet|TenGigE|Loopback)[\w\/\.]+|no\s+shutdown|ip\s+route\s+[\d\.]+)`)
	reDatacomCLI  = regexp.MustCompile(`(?mi)^\s*(?:route-map\s+[\w\-]+|ip\s+access-list|ip\s+policy\s+route-map|dmos\b)`)
	reZTECLI      = regexp.MustCompile(`(?mi)^\s*(?:gpon-onu-profile|interface\s+gpon-olt|tcont\s+\d+|gemport\s+\d+)`)
	reGenericCLI  = regexp.MustCompile(`(?mi)^\s*(?:#|\$|>)\s*([a-zA-Z0-9_\-\.\/]{2,}\s+[^\n\r]+)`)
)

// CanonicalHost extrai o hostname limpo em minúsculas sem porta, userinfo ou trailing dot.
func CanonicalHost(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, "://") {
		if u, err := url.Parse(raw); err == nil && u.Hostname() != "" {
			raw = u.Hostname()
		}
	} else if strings.Contains(raw, "/") {
		raw = strings.SplitN(raw, "/", 2)[0]
	}
	if idx := strings.LastIndex(raw, "@"); idx != -1 {
		raw = raw[idx+1:]
	}
	if host, _, err := net.SplitHostPort(raw); err == nil {
		raw = host
	}
	return strings.ToLower(strings.TrimRight(raw, "."))
}

// IsOfficialDomain valida se o host corresponde estritamente ao domínio oficial ou a um subdomínio legítimo.
// Previne vulnerabilidades onde strings.Contains(lower, "huawei.com") aceitava "huawei.com.attacker.com".
func IsOfficialDomain(domainOrURL string) bool {
	host := CanonicalHost(domainOrURL)
	if host == "" {
		return false
	}
	for _, off := range officialDomains {
		offCanonical := strings.ToLower(strings.TrimSpace(off))
		if host == offCanonical || strings.HasSuffix(host, "."+offCanonical) {
			return true
		}
	}
	return false
}

func Classify(text string, domain string) ClassificationResult {
	res := ClassificationResult{
		Vendors:       []DetectedVendor{},
		CLICommands:   []string{},
		Tags:          []string{},
		IsOfficialDoc: IsOfficialDomain(domain),
	}

	lowerText := strings.ToLower(text)

	// Detect Huawei
	if strings.Contains(lowerText, "huawei") || strings.Contains(lowerText, "ne8000") || strings.Contains(lowerText, "netengine") || strings.Contains(lowerText, "vrp") {
		os := "VRP"
		if strings.Contains(lowerText, "vrpv8") || strings.Contains(lowerText, "vrp v8") || strings.Contains(lowerText, "ne8000") || strings.Contains(lowerText, "netengine 8000") {
			os = "VRP v8"
		}
		cmds := extractMatches(reHuaweiCLI, text)
		res.Vendors = append(res.Vendors, DetectedVendor{
			Slug:       "huawei",
			Name:       "Huawei",
			OS:         os,
			Confidence: 0.9,
			Commands:   cmds,
		})
		res.CLICommands = append(res.CLICommands, cmds...)
		res.Tags = append(res.Tags, "huawei", strings.ToLower(os))
	}

	// Detect MikroTik
	if strings.Contains(lowerText, "mikrotik") || strings.Contains(lowerText, "routeros") || strings.Contains(lowerText, "routerboard") {
		os := "RouterOS"
		if strings.Contains(lowerText, "routeros v7") || strings.Contains(lowerText, "ros v7") || strings.Contains(lowerText, "v7.") {
			os = "RouterOS v7"
		} else if strings.Contains(lowerText, "routeros v6") || strings.Contains(lowerText, "ros v6") || strings.Contains(lowerText, "v6.") {
			os = "RouterOS v6"
		}
		cmds := extractMatches(reMikroTikCLI, text)
		res.Vendors = append(res.Vendors, DetectedVendor{
			Slug:       "mikrotik",
			Name:       "MikroTik",
			OS:         os,
			Confidence: 0.9,
			Commands:   cmds,
		})
		res.CLICommands = append(res.CLICommands, cmds...)
		res.Tags = append(res.Tags, "mikrotik", strings.ToLower(os))
	}

	// Detect Cisco
	if strings.Contains(lowerText, "cisco") || strings.Contains(lowerText, "ios-xr") || strings.Contains(lowerText, "ios-xe") {
		os := "IOS"
		if strings.Contains(lowerText, "ios-xr") || strings.Contains(lowerText, "asr9000") || strings.Contains(lowerText, "asr 9000") {
			os = "IOS-XR"
		} else if strings.Contains(lowerText, "ios-xe") || strings.Contains(lowerText, "catalyst 8") {
			os = "IOS-XE"
		}
		cmds := extractMatches(reCiscoCLI, text)
		res.Vendors = append(res.Vendors, DetectedVendor{
			Slug:       "cisco",
			Name:       "Cisco",
			OS:         os,
			Confidence: 0.9,
			Commands:   cmds,
		})
		res.CLICommands = append(res.CLICommands, cmds...)
		res.Tags = append(res.Tags, "cisco", strings.ToLower(os))
	}

	// Detect Datacom
	if strings.Contains(lowerText, "datacom") || strings.Contains(lowerText, "dmos") || strings.Contains(lowerText, "dm4270") {
		cmds := extractMatches(reDatacomCLI, text)
		res.Vendors = append(res.Vendors, DetectedVendor{
			Slug:       "datacom",
			Name:       "Datacom",
			OS:         "DmOS",
			Confidence: 0.9,
			Commands:   cmds,
		})
		res.CLICommands = append(res.CLICommands, cmds...)
		res.Tags = append(res.Tags, "datacom", "dmos")
	}

	// Detect ZTE
	if strings.Contains(lowerText, "zte") || strings.Contains(lowerText, "zxros") || strings.Contains(lowerText, "c320") || strings.Contains(lowerText, "c300") {
		cmds := extractMatches(reZTECLI, text)
		res.Vendors = append(res.Vendors, DetectedVendor{
			Slug:       "zte",
			Name:       "ZTE",
			OS:         "ZXROS",
			Confidence: 0.85,
			Commands:   cmds,
		})
		res.CLICommands = append(res.CLICommands, cmds...)
		res.Tags = append(res.Tags, "zte", "zxros")
	}

	// Detect FiberHome
	if strings.Contains(lowerText, "fiberhome") || strings.Contains(lowerText, "an5516") {
		res.Vendors = append(res.Vendors, DetectedVendor{
			Slug:       "fiberhome",
			Name:       "FiberHome",
			OS:         "Fiberhome OS",
			Confidence: 0.85,
		})
		res.Tags = append(res.Tags, "fiberhome")
	}

	// Detect VSOL
	if strings.Contains(lowerText, "vsol") || strings.Contains(lowerText, "v1600g") {
		res.Vendors = append(res.Vendors, DetectedVendor{
			Slug:       "vsol",
			Name:       "VSOL",
			OS:         "VSOL CLI",
			Confidence: 0.85,
		})
		res.Tags = append(res.Tags, "vsol")
	}

	// Detect SonicWall
	if strings.Contains(lowerText, "sonicwall") || strings.Contains(lowerText, "sonicos") {
		res.Vendors = append(res.Vendors, DetectedVendor{
			Slug:       "sonicwall",
			Name:       "SonicWall",
			OS:         "SonicOS",
			Confidence: 0.85,
		})
		res.Tags = append(res.Tags, "sonicwall", "sonicos")
	}

	// Detect Hillstone
	if strings.Contains(lowerText, "hillstone") || strings.Contains(lowerText, "stoneos") {
		res.Vendors = append(res.Vendors, DetectedVendor{
			Slug:       "hillstone",
			Name:       "Hillstone",
			OS:         "StoneOS",
			Confidence: 0.85,
		})
		res.Tags = append(res.Tags, "hillstone", "stoneos")
	}

	// Generic CLI prompt matches if no vendor-specific matches
	if len(res.CLICommands) == 0 {
		genericMatches := reGenericCLI.FindAllStringSubmatch(text, 10)
		for _, m := range genericMatches {
			if len(m) > 1 {
				res.CLICommands = append(res.CLICommands, strings.TrimSpace(m[1]))
			}
		}
	}

	res.CLICommands = dedupe(res.CLICommands)
	res.Tags = dedupe(res.Tags)
	return res
}

func extractMatches(re *regexp.Regexp, text string) []string {
	matches := re.FindAllString(text, 20)
	var out []string
	for _, m := range matches {
		cleaned := strings.TrimSpace(m)
		if len(cleaned) > 3 {
			out = append(out, cleaned)
		}
	}
	return out
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
