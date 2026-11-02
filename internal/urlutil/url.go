package urlutil

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"net/url"
	"path"
	"sort"
	"strings"

	"nie/internal/netguard"
)

var (
	ErrInvalidURL  = errors.New("url inválida")
	ErrUnsupported = errors.New("somente http/https é permitido")
	ErrPrivateHost = errors.New("host privado ou local não permitido")
)

// Normalize devolve uma forma canônica para deduplicação.
func Normalize(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ErrInvalidURL
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", ErrInvalidURL
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", ErrUnsupported
	}
	if u.Host == "" || u.Hostname() == "" {
		return "", ErrInvalidURL
	}
	if u.User != nil {
		return "", ErrInvalidURL
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if host == "localhost" || strings.HasSuffix(host, ".localhost") ||
		strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") ||
		strings.HasSuffix(host, ".home.arpa") {
		return "", ErrPrivateHost
	}
	port := u.Port()
	if (u.Scheme == "http" && port == "80") || (u.Scheme == "https" && port == "443") {
		port = ""
	}
	if port != "" {
		host = host + ":" + port
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = host
	u.Fragment = ""
	u.RawFragment = ""

	p := u.EscapedPath()
	if p == "" {
		p = "/"
	}
	if dec, err := url.PathUnescape(p); err == nil {
		p = path.Clean("/" + dec)
		if dec == "" || strings.HasSuffix(dec, "/") {
			if p != "/" {
				p += "/"
			}
		}
	}
	u.Path = p
	u.RawPath = ""

	q := u.Query()
	for key := range q {
		lk := strings.ToLower(key)
		if strings.HasPrefix(lk, "utm_") || lk == "fbclid" || lk == "gclid" || lk == "mc_cid" || lk == "mc_eid" {
			q.Del(key)
		}
	}
	keys := make([]string, 0, len(q))
	for key := range q {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var parts []string
	for _, key := range keys {
		vals := q[key]
		sort.Strings(vals)
		for _, val := range vals {
			parts = append(parts, url.QueryEscape(key)+"="+url.QueryEscape(val))
		}
	}
	u.RawQuery = strings.Join(parts, "&")
	return u.String(), nil
}

func ParseNormalized(raw string) (*url.URL, string, error) {
	norm, err := Normalize(raw)
	if err != nil {
		return nil, "", err
	}
	u, err := url.Parse(norm)
	if err != nil {
		return nil, "", err
	}
	return u, norm, nil
}

func Domain(normalized string) string {
	u, err := url.Parse(normalized)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

func Hash(normalized string) string {
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

// Resolve resolve um href relativo e devolve apenas URLs http(s) normalizadas.
func Resolve(base *url.URL, href string) (string, bool) {
	href = strings.TrimSpace(href)
	if href == "" || strings.HasPrefix(href, "#") {
		return "", false
	}
	ref, err := url.Parse(href)
	if err != nil {
		return "", false
	}
	switch strings.ToLower(ref.Scheme) {
	case "mailto", "tel", "javascript", "data", "ftp", "file":
		return "", false
	}
	abs := base.ResolveReference(ref)
	norm, err := Normalize(abs.String())
	if err != nil {
		return "", false
	}
	return norm, true
}

// ValidateCrawlURL aplica restrições de esquema, porta e host antes da resolução DNS.
func ValidateCrawlURL(u *url.URL) error {
	if u == nil {
		return ErrInvalidURL
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ErrUnsupported
	}
	if u.Host == "" || u.Hostname() == "" || u.User != nil {
		return ErrInvalidURL
	}
	if p := u.Port(); p != "" && p != "80" && p != "443" {
		return ErrUnsupported
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") ||
		strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") ||
		strings.HasSuffix(host, ".home.arpa") {
		return ErrPrivateHost
	}
	if ip := net.ParseIP(host); ip != nil && !netguard.IsPublicIP(ip) {
		return ErrPrivateHost
	}
	return nil
}
