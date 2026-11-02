package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
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

// ClusterMap groups duplicate text occurrences by fingerprint to prevent
// 3 identical copied blogs from counting as 3 independent evidences.
type ClusterTracker struct {
	clusters map[string][]string // fingerprint -> list of unique domains
}

func NewClusterTracker() *ClusterTracker {
	return &ClusterTracker{
		clusters: make(map[string][]string),
	}
}

func (c *ClusterTracker) Add(fingerprint, domain string) int {
	if fingerprint == "" {
		return 1
	}
	domains, exists := c.clusters[fingerprint]
	if !exists {
		c.clusters[fingerprint] = []string{domain}
		return 1
	}
	for _, d := range domains {
		if d == domain {
			// Same domain duplicate
			return len(domains)
		}
	}
	c.clusters[fingerprint] = append(domains, domain)
	return len(c.clusters[fingerprint])
}

func (c *ClusterTracker) IndependentCount(fingerprint string) int {
	if fingerprint == "" {
		return 1
	}
	return len(c.clusters[fingerprint])
}
