package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config reúne parâmetros de implantação. Ajustes de campanha ficam na API/UI.
type Config struct {
	Addr        string
	DatabaseURL string
	DataDir     string
	WebDir      string
	APIToken    string
	UserAgent   string
	LogLevel    string

	WorkerEnabled bool
	WorkerCount   int
	PollInterval  time.Duration

	RequestTimeout  time.Duration
	MaxHTMLBytes    int64
	MaxPDFBytes     int64
	MaxTextBytes    int64
	RateLimitPerMin int
	SearXNGURL      string
	GitHubEnabled   bool
	GitHubToken     string
}

func Load() Config {
	rate := envInt("NIE_RATE_LIMIT_PER_MINUTE", 6)
	if rate < 1 {
		rate = 1
	}
	return Config{
		Addr:            env("NIE_ADDR", "127.0.0.1:8080"),
		DatabaseURL:     env("NIE_DATABASE_URL", "postgres://nie:nie@127.0.0.1:5432/nie?sslmode=disable"),
		DataDir:         env("NIE_DATA_DIR", "./data"),
		WebDir:          env("NIE_WEB_DIR", ""),
		APIToken:        env("NIE_API_TOKEN", ""),
		UserAgent:       env("NIE_USER_AGENT", "NIEBot/0.1 (+https://localhost)"),
		LogLevel:        strings.ToLower(env("NIE_LOG_LEVEL", "info")),
		WorkerEnabled:   envBool("NIE_WORKER", true),
		WorkerCount:     envInt("NIE_WORKERS", 2),
		PollInterval:    envDuration("NIE_POLL_INTERVAL", 2*time.Second),
		RequestTimeout:  envDuration("NIE_REQUEST_TIMEOUT", 30*time.Second),
		MaxHTMLBytes:    envInt64("NIE_MAX_HTML_BYTES", 5<<20),
		MaxPDFBytes:     envInt64("NIE_MAX_PDF_BYTES", 50<<20),
		MaxTextBytes:    envInt64("NIE_MAX_TEXT_BYTES", 5<<20),
		RateLimitPerMin: rate,
		SearXNGURL:      strings.TrimRight(env("NIE_SEARXNG_URL", ""), "/"),
		GitHubEnabled:   envBool("NIE_GITHUB_ENABLED", false) || env("NIE_GITHUB_TOKEN", "") != "",
		GitHubToken:     env("NIE_GITHUB_TOKEN", ""),
	}
}

func (c Config) DomainInterval() time.Duration {
	// 6 req/min => 10s. Mantém pelo menos 1s.
	secs := 60 / c.RateLimitPerMin
	if secs < 1 {
		secs = 1
	}
	return time.Duration(secs) * time.Second
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	v, err := strconv.Atoi(strings.TrimSpace(env(key, "")))
	if err != nil {
		return def
	}
	return v
}

func envInt64(key string, def int64) int64 {
	v, err := strconv.ParseInt(strings.TrimSpace(env(key, "")), 10, 64)
	if err != nil {
		return def
	}
	return v
}

func envBool(key string, def bool) bool {
	v, err := strconv.ParseBool(strings.TrimSpace(env(key, "")))
	if err != nil {
		return def
	}
	return v
}

func envDuration(key string, def time.Duration) time.Duration {
	v, err := time.ParseDuration(strings.TrimSpace(env(key, "")))
	if err != nil {
		return def
	}
	return v
}
