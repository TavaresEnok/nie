package domain

import (
	"encoding/json"
	"time"
)

type Budget struct {
	MaxDepth                 int   `json:"max_depth"`
	MaxURLs                  int   `json:"max_urls"`
	MaxDomains               int   `json:"max_domains"`
	MaxBytes                 int64 `json:"max_bytes"`
	MinDomainIntervalSeconds int   `json:"min_domain_interval_seconds"`
}

type CampaignStats struct {
	JobsPending int   `json:"jobs_pending"`
	JobsRunning int   `json:"jobs_running"`
	JobsDone    int   `json:"jobs_done"`
	JobsFailed  int   `json:"jobs_failed"`
	JobsDead    int   `json:"jobs_dead"`
	Documents   int   `json:"documents"`
	Passages    int   `json:"passages"`
	Domains     int   `json:"domains"`
	Bytes       int64 `json:"bytes"`
}

type Campaign struct {
	ID         string        `json:"id"`
	Query      string        `json:"query"`
	Status     string        `json:"status"`
	SeedURLs   []string      `json:"seed_urls"`
	Budget     Budget        `json:"budget"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
	FinishedAt *time.Time    `json:"finished_at,omitempty"`
	Stats      CampaignStats `json:"stats"`
}

type Job struct {
	ID            int64           `json:"id"`
	CampaignID    string          `json:"campaign_id"`
	Kind          string          `json:"kind"`
	URL           string          `json:"url,omitempty"`
	NormalizedURL string          `json:"normalized_url,omitempty"`
	Domain        string          `json:"domain,omitempty"`
	Depth         int             `json:"depth"`
	Priority      int             `json:"priority"`
	ParentURL     string          `json:"parent_url,omitempty"`
	Reason        string          `json:"reason,omitempty"`
	Payload       json.RawMessage `json:"payload,omitempty"`
	Status        string          `json:"status"`
	Attempts      int             `json:"attempts"`
	MaxAttempts   int             `json:"max_attempts"`
	ScheduledAt   time.Time       `json:"scheduled_at"`
	LeaseUntil    *time.Time      `json:"lease_until,omitempty"`
	LastError     string          `json:"last_error,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

type Passage struct {
	ID         int64  `json:"id"`
	DocumentID string `json:"document_id"`
	Ordinal    int    `json:"ordinal"`
	Page       *int   `json:"page,omitempty"`
	Text       string `json:"text"`
}

type DocumentVersion struct {
	ID            string    `json:"id"`
	DocumentID    string    `json:"document_id"`
	CampaignID    string    `json:"campaign_id"`
	URL           string    `json:"url"`
	NormalizedURL string    `json:"normalized_url"`
	Domain        string    `json:"domain"`
	StatusCode    int       `json:"status_code"`
	ETag          string    `json:"etag,omitempty"`
	LastModified  string    `json:"last_modified,omitempty"`
	FetchedAt     time.Time `json:"fetched_at"`
}

type Document struct {
	ID        string            `json:"id"`
	SHA256    string            `json:"sha256"`
	MIME      string            `json:"mime"`
	Title     string            `json:"title"`
	Size      int64             `json:"size"`
	Text      string            `json:"text,omitempty"`
	Metadata  map[string]any    `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	Versions  []DocumentVersion `json:"versions,omitempty"`
	Passages  []Passage         `json:"passages,omitempty"`
}

type DocumentSummary struct {
	ID         string    `json:"id"`
	SHA256     string    `json:"sha256"`
	MIME       string    `json:"mime"`
	Title      string    `json:"title"`
	Size       int64     `json:"size"`
	URL        string    `json:"url"`
	Domain     string    `json:"domain"`
	CampaignID string    `json:"campaign_id"`
	FetchedAt  time.Time `json:"fetched_at"`
	CreatedAt  time.Time `json:"created_at"`
}

type SearchResult struct {
	PassageID  int64     `json:"passage_id"`
	DocumentID string    `json:"document_id"`
	Title      string    `json:"title"`
	MIME       string    `json:"mime"`
	SHA256     string    `json:"sha256"`
	URL        string    `json:"url"`
	Domain     string    `json:"domain"`
	Page       *int      `json:"page,omitempty"`
	Ordinal    int       `json:"ordinal"`
	Snippet    string    `json:"snippet"`
	Match      string    `json:"match"`
	Score      float64   `json:"score"`
	FetchedAt  time.Time `json:"fetched_at"`
}

type DiscoveryEdge struct {
	ID         int64     `json:"id"`
	CampaignID string    `json:"campaign_id"`
	FromURL    string    `json:"from_url,omitempty"`
	ToURL      string    `json:"to_url"`
	ToDomain   string    `json:"to_domain"`
	Kind       string    `json:"kind"`
	Depth      int       `json:"depth"`
	Reason     string    `json:"reason,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type Stats struct {
	Campaigns  int            `json:"campaigns"`
	Documents  int            `json:"documents"`
	Passages   int            `json:"passages"`
	Bytes      int64          `json:"bytes"`
	Domains    int            `json:"domains"`
	Jobs       map[string]int `json:"jobs"`
	Version    string         `json:"version"`
	StartedAt  time.Time      `json:"started_at"`
	UptimeSecs int64          `json:"uptime_seconds"`
}

type Dossier struct {
	Campaign       Campaign        `json:"campaign"`
	DocumentsCount int             `json:"documents_count"`
	PassagesCount  int             `json:"passages_count"`
	DomainsCount   int             `json:"domains_count"`
	BytesTotal     int64           `json:"bytes_total"`
	Markdown       string          `json:"markdown"`
	GeneratedAt    time.Time       `json:"generated_at"`
}
