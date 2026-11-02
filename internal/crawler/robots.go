package crawler

import (
	"context"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/temoto/robotstxt"
)

type robotsEntry struct {
	rules     *robotstxt.RobotsData
	allowAll  bool
	denyAll   bool
	expiresAt time.Time
}

type Robots struct {
	fetcher   *Fetcher
	userAgent string
	logger    *slog.Logger
	ttl       time.Duration

	mu    sync.Mutex
	cache map[string]robotsEntry
}

func NewRobots(fetcher *Fetcher, userAgent string, logger *slog.Logger) *Robots {
	return &Robots{
		fetcher:   fetcher,
		userAgent: userAgent,
		logger:    logger,
		ttl:       time.Hour,
		cache:     make(map[string]robotsEntry),
	}
}

func (r *Robots) Allowed(ctx context.Context, u *url.URL) bool {
	key := u.Scheme + "://" + u.Host
	now := time.Now()

	r.mu.Lock()
	if entry, ok := r.cache[key]; ok && now.Before(entry.expiresAt) {
		r.mu.Unlock()
		return r.test(entry, u)
	}
	r.mu.Unlock()

	entry := r.load(ctx, u)
	r.mu.Lock()
	r.cache[key] = entry
	r.mu.Unlock()
	return r.test(entry, u)
}

func (r *Robots) test(entry robotsEntry, u *url.URL) bool {
	if entry.allowAll {
		return true
	}
	if entry.denyAll || entry.rules == nil {
		return false
	}
	group := entry.rules.FindGroup(robotToken(r.userAgent))
	return group.Test(u.RequestURI())
}

func (r *Robots) load(ctx context.Context, u *url.URL) robotsEntry {
	robotsURL := u.Scheme + "://" + u.Host + "/robots.txt"
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	resp, err := r.fetcher.Fetch(ctx, robotsURL)
	if err != nil {
		r.logger.Debug("robots indisponível; permitindo com cache curto", "host", u.Host, "err", err)
		return robotsEntry{allowAll: true, expiresAt: time.Now().Add(5 * time.Minute)}
	}
	return robotsEntryFromResponse(resp.StatusCode, resp.Body, r.ttl)
}

// robotsEntryFromResponse aplica a política de robots.txt da RFC 9309:
// 2xx usa as regras; 404/410 libera; 401/403/429 e 5xx bloqueiam
// temporariamente; respostas inválidas liberam com cache curto.
func robotsEntryFromResponse(status int, body []byte, ttl time.Duration) robotsEntry {
	switch {
	case status >= 200 && status < 300:
		data, err := robotstxt.FromBytes(body)
		if err != nil {
			return robotsEntry{allowAll: true, expiresAt: time.Now().Add(5 * time.Minute)}
		}
		return robotsEntry{rules: data, expiresAt: time.Now().Add(ttl)}
	case status == 404 || status == 410:
		return robotsEntry{allowAll: true, expiresAt: time.Now().Add(ttl)}
	default:
		// 401/403/429 e 5xx: bloquear temporariamente (RFC 9309).
		return robotsEntry{denyAll: true, expiresAt: time.Now().Add(5 * time.Minute)}
	}
}

func robotToken(ua string) string {
	ua = strings.TrimSpace(ua)
	if i := strings.IndexAny(ua, " /"); i > 0 {
		return ua[:i]
	}
	return ua
}
