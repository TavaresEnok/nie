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

	RequestTimeout     time.Duration
	MaxHTMLBytes       int64
	MaxPDFBytes        int64
	MaxTextBytes       int64
	RateLimitPerMin    int
	SearXNGURL         string
	GitHubEnabled      bool
	GitHubToken        string
	GitLabEnabled      bool
	GitLabURL          string
	GitLabToken        string
	CommonCrawlEnabled bool
	CommonCrawlIndex   string
	ZoektURL           string
	OpenSearchURL      string
	OpenSearchIndex    string
	OpenSearchUser     string
	OpenSearchPassword string
	TikaCommand        string
	DoclingCommand     string
	ArchiveBoxDir      string
	GPTResearcherURL   string
	PGVector           bool

	// Fase 2 e 3: IA, Autonomia e Git Local
	AIProvider        string
	AIBaseURL         string
	AIAPIKey          string
	AIModel           string
	SchedulerEnabled  bool
	SchedulerInterval time.Duration
	GitLocalDir       string

	// V2.1: Executores Especializados e Descobertas Complementares
	ScrapyURL                string
	Crawl4AIURL              string
	MwmblURL                 string
	WaybackEnabled           bool
	SoftwareHeritageEnabled  bool
	MentionsDiscoveryEnabled bool
	AutomaticResearch        bool
	MaxResearchRounds        int
	OCREnabled               bool
	OCRMaxPages              int
	OCRLanguage              string
	OCRTimeout               time.Duration
	RevisitInterval          time.Duration
}

func Load() Config {
	rate := envInt("NIE_RATE_LIMIT_PER_MINUTE", 6)
	if rate < 1 {
		rate = 1
	}
	return Config{
		Addr:                     env("NIE_ADDR", "127.0.0.1:8080"),
		DatabaseURL:              env("NIE_DATABASE_URL", "postgres://nie:nie@127.0.0.1:5432/nie?sslmode=disable"),
		DataDir:                  env("NIE_DATA_DIR", "./data"),
		WebDir:                   env("NIE_WEB_DIR", ""),
		APIToken:                 env("NIE_API_TOKEN", ""),
		UserAgent:                env("NIE_USER_AGENT", "NIEBot/0.1 (+https://localhost)"),
		LogLevel:                 strings.ToLower(env("NIE_LOG_LEVEL", "info")),
		WorkerEnabled:            envBool("NIE_WORKER", true),
		WorkerCount:              envInt("NIE_WORKERS", 2),
		PollInterval:             envDuration("NIE_POLL_INTERVAL", 2*time.Second),
		RequestTimeout:           envDuration("NIE_REQUEST_TIMEOUT", 30*time.Second),
		MaxHTMLBytes:             envInt64("NIE_MAX_HTML_BYTES", 5<<20),
		MaxPDFBytes:              envInt64("NIE_MAX_PDF_BYTES", 50<<20),
		MaxTextBytes:             envInt64("NIE_MAX_TEXT_BYTES", 5<<20),
		RateLimitPerMin:          rate,
		SearXNGURL:               strings.TrimRight(env("NIE_SEARXNG_URL", ""), "/"),
		GitHubEnabled:            envBool("NIE_GITHUB_ENABLED", false) || env("NIE_GITHUB_TOKEN", "") != "",
		GitHubToken:              env("NIE_GITHUB_TOKEN", ""),
		GitLabEnabled:            envBool("NIE_GITLAB_ENABLED", false) || env("NIE_GITLAB_TOKEN", "") != "",
		GitLabURL:                strings.TrimRight(env("NIE_GITLAB_URL", "https://gitlab.com"), "/"),
		GitLabToken:              env("NIE_GITLAB_TOKEN", ""),
		CommonCrawlEnabled:       envBool("NIE_COMMONCRAWL_ENABLED", false),
		CommonCrawlIndex:         env("NIE_COMMONCRAWL_INDEX", "https://index.commoncrawl.org/collinfo.json"),
		ZoektURL:                 strings.TrimRight(env("NIE_ZOEKT_URL", ""), "/"),
		OpenSearchURL:            strings.TrimRight(env("NIE_OPENSEARCH_URL", ""), "/"),
		OpenSearchIndex:          env("NIE_OPENSEARCH_INDEX", "nie"),
		OpenSearchUser:           env("NIE_OPENSEARCH_USER", ""),
		OpenSearchPassword:       env("NIE_OPENSEARCH_PASSWORD", ""),
		TikaCommand:              env("NIE_TIKA_COMMAND", ""),
		DoclingCommand:           env("NIE_DOCLING_COMMAND", ""),
		ArchiveBoxDir:            env("NIE_ARCHIVEBOX_DIR", ""),
		GPTResearcherURL:         strings.TrimRight(env("NIE_GPT_RESEARCHER_URL", ""), "/"),
		PGVector:                 envBool("NIE_PGVECTOR", false),
		AIProvider:               strings.ToLower(env("NIE_AI_PROVIDER", "none")),
		AIBaseURL:                env("NIE_AI_BASE_URL", ""),
		AIAPIKey:                 env("NIE_AI_API_KEY", ""),
		AIModel:                  env("NIE_AI_MODEL", "qwen2.5:7b"),
		SchedulerEnabled:         envBool("NIE_SCHEDULER_ENABLED", true),
		SchedulerInterval:        envDuration("NIE_SCHEDULER_INTERVAL", 30*time.Second),
		GitLocalDir:              env("NIE_GIT_LOCAL_DIR", ""),
		ScrapyURL:                strings.TrimRight(env("NIE_SCRAPY_URL", ""), "/"),
		Crawl4AIURL:              strings.TrimRight(env("NIE_CRAWL4AI_URL", ""), "/"),
		MwmblURL:                 strings.TrimRight(env("NIE_MWMBL_URL", "https://mwmbl.org"), "/"),
		WaybackEnabled:           envBool("NIE_WAYBACK_ENABLED", true),
		SoftwareHeritageEnabled:  envBool("NIE_SOFTWARE_HERITAGE_ENABLED", true),
		MentionsDiscoveryEnabled: envBool("NIE_MENTIONS_DISCOVERY_ENABLED", true),
		AutomaticResearch:        envBool("NIE_AUTOMATIC_RESEARCH", true),
		MaxResearchRounds:        envInt("NIE_MAX_RESEARCH_ROUNDS", 3),
		OCREnabled:               envBool("NIE_OCR_ENABLED", false),
		OCRMaxPages:              envInt("NIE_OCR_MAX_PAGES", 10),
		OCRLanguage:              env("NIE_OCR_LANGUAGE", "eng"),
		OCRTimeout:               envDuration("NIE_OCR_TIMEOUT", 45*time.Second),
		RevisitInterval:          envDuration("NIE_REVISIT_INTERVAL", 0),
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

// Leave headroom within the HTTP write timeout and the fetch job's lease.
func (c Config) OCRProcessingTimeout() time.Duration {
	if c.OCRTimeout <= 0 {
		return 45 * time.Second
	}
	return max(time.Second, min(c.OCRTimeout, 45*time.Second))
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
