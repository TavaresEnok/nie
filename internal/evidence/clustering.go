package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"sync"
)

var reWhitespace = regexp.MustCompile(`\s+`)
var reNonAlpha = regexp.MustCompile(`[^a-z0-9]+`)

// Fingerprint generates a normalized content hash for duplicate/mirror detection.
// It removes non-alphanumeric noise and whitespace so slight reformatting or
// copy-pasting still produces the exact same fingerprint.
func Fingerprint(text string) string {
	lower := strings.ToLower(text)
	normalized := reNonAlpha.ReplaceAllString(lower, " ")
	compact := strings.TrimSpace(reWhitespace.ReplaceAllString(normalized, " "))
	if len(compact) == 0 {
		return ""
	}
	// Limit to leading 500 chars to match paragraphs even with trailing comments
	if len(compact) > 500 {
		compact = compact[:500]
	}
	h := sha256.Sum256([]byte(compact))
	return hex.EncodeToString(h[:])
}

// ClusterTracker groups duplicate text occurrences by fingerprint to prevent
// 3 identical copied blogs from counting as 3 independent evidences.
// Thread-safe via sync.RWMutex conforme exigido em P1-06 (go test -race).
type ClusterTracker struct {
	mu       sync.RWMutex
	clusters map[string][]string // fingerprint -> list of replica domains
}

func NewClusterTracker() *ClusterTracker {
	return &ClusterTracker{
		clusters: make(map[string][]string),
	}
}

// Add registra uma ocorrência de um fingerprint associado a um domínio.
// Retorna a quantidade de réplicas registradas para esse mesmo conteúdo.
func (c *ClusterTracker) Add(fingerprint, domain string) int {
	if fingerprint == "" {
		return 1
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	domain = strings.ToLower(strings.TrimSpace(domain))
	domains, exists := c.clusters[fingerprint]
	if !exists {
		c.clusters[fingerprint] = []string{domain}
		return 1
	}
	for _, d := range domains {
		if d == domain {
			return len(domains)
		}
	}
	c.clusters[fingerprint] = append(domains, domain)
	return len(c.clusters[fingerprint])
}

// ReplicaCount retorna quantas republicações/cópias existem do mesmo fingerprint.
func (c *ClusterTracker) ReplicaCount(fingerprint string) int {
	if fingerprint == "" {
		return 1
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.clusters[fingerprint])
}

// IndependentCount mantido por compatibilidade legada, retorna réplicas do fingerprint.
func (c *ClusterTracker) IndependentCount(fingerprint string) int {
	return c.ReplicaCount(fingerprint)
}

// CountIndependentOrigins calcula a quantidade real de origens substantivamente independentes.
// Se 3 domínios diferentes possuem o mesmo fingerprint, contam como 1 ÚNICA origem (réplicas do mesmo texto).
// Se 3 trechos diferentes vêm do mesmo domínio, contam como 1 ÚNICO domínio de origem.
// Retorna a contagem efetiva de origens independentes.
func (c *ClusterTracker) CountIndependentOrigins(fingerprints []string, domains []string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	seenClusters := make(map[string]bool)
	seenDomains := make(map[string]bool)

	for i := 0; i < len(fingerprints) && i < len(domains); i++ {
		fp := fingerprints[i]
		dom := strings.ToLower(strings.TrimSpace(domains[i]))
		if fp == "" || dom == "" {
			continue
		}
		seenClusters[fp] = true
		seenDomains[dom] = true
	}

	clusterCount := len(seenClusters)
	domainCount := len(seenDomains)

	if clusterCount == 0 || domainCount == 0 {
		return 0
	}
	if clusterCount < domainCount {
		return clusterCount
	}
	return domainCount
}
