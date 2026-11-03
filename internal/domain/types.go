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
	ID              int64           `json:"id"`
	CampaignID      string          `json:"campaign_id"`
	Kind            string          `json:"kind"`
	URL             string          `json:"url,omitempty"`
	NormalizedURL   string          `json:"normalized_url,omitempty"`
	Domain          string          `json:"domain,omitempty"`
	Depth           int             `json:"depth"`
	Priority        int             `json:"priority"`
	ParentURL       string          `json:"parent_url,omitempty"`
	Reason          string          `json:"reason,omitempty"`
	Payload         json.RawMessage `json:"payload,omitempty"`
	Status          string          `json:"status"`
	Attempts        int             `json:"attempts"`
	LeaseGeneration int64           `json:"lease_generation"`
	MaxAttempts     int             `json:"max_attempts"`
	ScheduledAt     time.Time       `json:"scheduled_at"`
	LeaseUntil      *time.Time      `json:"lease_until,omitempty"`
	LastError       string          `json:"last_error,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

type Passage struct {
	ID              int64  `json:"id"`
	DocumentID      string `json:"document_id"`
	Ordinal         int    `json:"ordinal"`
	Page            *int   `json:"page,omitempty"`
	Text            string `json:"text"`
	ExtractionRunID string `json:"extraction_run_id,omitempty"`
	StartOffset     *int   `json:"start_offset,omitempty"`
	EndOffset       *int   `json:"end_offset,omitempty"`
}

type ExtractionRun struct {
	ID             string         `json:"id"`
	DocumentID     string         `json:"document_id"`
	Parser         string         `json:"parser"`
	ParserVersion  string         `json:"parser_version"`
	Config         map[string]any `json:"config"`
	InputSHA256    *string        `json:"input_sha256,omitempty"`
	IdentitySHA256 *string        `json:"identity_sha256,omitempty"`
	Title          string         `json:"title"`
	Text           string         `json:"text,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	Passages       []Passage      `json:"passages,omitempty"`
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
	ID                 string            `json:"id"`
	SHA256             string            `json:"sha256"`
	MIME               string            `json:"mime"`
	Title              string            `json:"title"`
	Size               int64             `json:"size"`
	Text               string            `json:"text,omitempty"`
	Metadata           map[string]any    `json:"metadata,omitempty"`
	CreatedAt          time.Time         `json:"created_at"`
	Versions           []DocumentVersion `json:"versions,omitempty"`
	Passages           []Passage         `json:"passages,omitempty"`
	ActiveExtractionID *string           `json:"active_extraction_id,omitempty"`
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
	ExtractionRunID string    `json:"extraction_run_id,omitempty"`
	PassageID       int64     `json:"passage_id"`
	DocumentID      string    `json:"document_id"`
	Title           string    `json:"title"`
	MIME            string    `json:"mime"`
	SHA256          string    `json:"sha256"`
	URL             string    `json:"url"`
	Domain          string    `json:"domain"`
	Page            *int      `json:"page,omitempty"`
	Ordinal         int       `json:"ordinal"`
	Text            string    `json:"text,omitempty"`
	Snippet         string    `json:"snippet"`
	StartOffset     *int      `json:"start_offset,omitempty"`
	EndOffset       *int      `json:"end_offset,omitempty"`
	Match           string    `json:"match"`
	Score           float64   `json:"score"`
	FetchedAt       time.Time `json:"fetched_at"`
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
	Campaign       Campaign  `json:"campaign"`
	DocumentsCount int       `json:"documents_count"`
	PassagesCount  int       `json:"passages_count"`
	DomainsCount   int       `json:"domains_count"`
	BytesTotal     int64     `json:"bytes_total"`
	Markdown       string    `json:"markdown"`
	GeneratedAt    time.Time `json:"generated_at"`
}

// Fase 2: Fabricantes, Modelos e Firmware
type Vendor struct {
	ID        string    `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	CLIFamily string    `json:"cli_family"`
	Aliases   []string  `json:"aliases"`
	CreatedAt time.Time `json:"created_at"`
}

type DeviceModel struct {
	ID        string    `json:"id"`
	VendorID  string    `json:"vendor_id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	Family    string    `json:"family"`
	CreatedAt time.Time `json:"created_at"`
}

type FirmwareRelease struct {
	ID        string    `json:"id"`
	VendorID  string    `json:"vendor_id"`
	ModelID   *string   `json:"model_id,omitempty"`
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	OSName    string    `json:"os_name"`
	CreatedAt time.Time `json:"created_at"`
}

// Fase 2: Evidências, Origin Clusters e Claims
type OriginCluster struct {
	ID             string    `json:"id"`
	CampaignID     string    `json:"campaign_id"`
	Fingerprint    string    `json:"fingerprint"`
	RootDomain     string    `json:"root_domain"`
	CanonicalURL   string    `json:"canonical_url"`
	DuplicateCount int       `json:"duplicate_count"`
	CreatedAt      time.Time `json:"created_at"`
}

type Evidence struct {
	ID                 string    `json:"id"`
	ClaimID            string    `json:"claim_id"`
	DocumentID         string    `json:"document_id"`
	PassageID          *int64    `json:"passage_id,omitempty"`
	ClusterID          *string   `json:"cluster_id,omitempty"`
	Stance             string    `json:"stance"` // supports, refutes, mentions
	Excerpt            string    `json:"excerpt"`
	Page               *int      `json:"page,omitempty"`
	Confidence         float32   `json:"confidence"`
	IsOfficialDoc      bool      `json:"is_official_doc"`
	Notes              string    `json:"notes,omitempty"`
	VendorMatch        bool      `json:"vendor_match"`
	ModelMatch         bool      `json:"model_match"`
	FirmwareMatch      bool      `json:"firmware_match"`
	FeatureMatch       bool      `json:"feature_match"`
	ModeMatch          bool      `json:"mode_match"`
	PrerequisitesMatch bool      `json:"prerequisites_match"`
	SourceSpan         string    `json:"source_span,omitempty"`
	ValidationRunID    *string   `json:"validation_run_id,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

type Contradiction struct {
	ID          string    `json:"id"`
	ClaimID     string    `json:"claim_id"`
	EvidenceAID string    `json:"evidence_a_id"`
	EvidenceBID string    `json:"evidence_b_id"`
	Explanation string    `json:"explanation"`
	CreatedAt   time.Time `json:"created_at"`
}

type Claim struct {
	ID                    string          `json:"id"`
	CampaignID            string          `json:"campaign_id"`
	Title                 string          `json:"title"`
	Statement             string          `json:"statement"`
	VendorID              *string         `json:"vendor_id,omitempty"`
	VendorName            string          `json:"vendor_name,omitempty"`
	ModelID               *string         `json:"model_id,omitempty"`
	FirmwareID            *string         `json:"firmware_id,omitempty"`
	Status                string          `json:"status"`                 // unconfirmed, sustained, corroborated, contested, validated_lab
	Applicability         string          `json:"applicability"`          // confirmed, incompatible, indeterminate
	OperationalValidation string          `json:"operational_validation"` // untested, lab_validated, failed
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
	Evidences             []Evidence      `json:"evidences,omitempty"`
	Contradictions        []Contradiction `json:"contradictions,omitempty"`
}

type InvestigationLead struct {
	ResearchQuestion string `json:"research_question"`
	Hypothesis       string `json:"hypothesis"`
	SearchLead       string `json:"search_lead"`
	EvidenceGap      string `json:"evidence_gap"`
	NextAction       string `json:"next_action"`
}

type InvestigationDossier struct {
	Campaign            Campaign            `json:"campaign"`
	Summary             string              `json:"summary"`
	KeyFindings         []string            `json:"key_findings"`
	Claims              []Claim             `json:"claims"`
	CompatibilityMatrix map[string]string   `json:"compatibility_matrix"`
	CLICommands         []string            `json:"cli_commands"`
	Warnings            []string            `json:"warnings"`
	RollbackPlan        string              `json:"rollback_plan"`
	LabChecklist        []string            `json:"lab_checklist"`
	InvestigationLeads  []InvestigationLead `json:"investigation_leads,omitempty"`
	Markdown            string              `json:"markdown"`
	GeneratedAt         time.Time           `json:"generated_at"`
}

// Fase 3: Autonomia, Agendamentos e Alertas
type Schedule struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Query           string     `json:"query"`
	SeedURLs        []string   `json:"seed_urls"`
	IntervalSeconds int        `json:"interval_seconds"`
	MaxDepth        int        `json:"max_depth"`
	MaxURLs         int        `json:"max_urls"`
	MaxDomains      int        `json:"max_domains"`
	Enabled         bool       `json:"enabled"`
	LastRunAt       *time.Time `json:"last_run_at,omitempty"`
	NextRunAt       time.Time  `json:"next_run_at"`
	LastCampaignID  *string    `json:"last_campaign_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type Alert struct {
	ID           string    `json:"id"`
	CampaignID   *string   `json:"campaign_id,omitempty"`
	Title        string    `json:"title"`
	Message      string    `json:"message"`
	Severity     string    `json:"severity"` // info, warning, critical
	Acknowledged bool      `json:"acknowledged"`
	CreatedAt    time.Time `json:"created_at"`
}

type DocumentDiff struct {
	ID         int64     `json:"id"`
	DocumentID string    `json:"document_id"`
	URL        string    `json:"url"`
	OldSHA256  string    `json:"old_sha256"`
	NewSHA256  string    `json:"new_sha256"`
	Summary    string    `json:"summary"`
	CreatedAt  time.Time `json:"created_at"`
}

// Fase 3: Grafo Visual de Descoberta
type GraphNode struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Domain string `json:"domain"`
	Depth  int    `json:"depth"`
	Type   string `json:"type"` // seed, link, search
	Status string `json:"status"`
}

type GraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Kind   string `json:"kind"`
	Depth  int    `json:"depth"`
	Reason string `json:"reason,omitempty"`
}

type DiscoveryGraph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// V2.1: Contrato dos Executores e Descoberta Contínua (ADENDO MESTRE V2.1)
type DiscoveryCandidate struct {
	URL            string    `json:"url"`
	NormalizedURL  string    `json:"normalized_url"`
	FromDocumentID *string   `json:"from_document_id,omitempty"`
	DiscoveryType  string    `json:"discovery_type"` // link, sitemap, mention, citation, api
	CampaignID     string    `json:"campaign_id"`
	Depth          int       `json:"depth"`
	Score          float64   `json:"score"`
	Reason         string    `json:"reason"`
	CreatedAt      time.Time `json:"created_at"`
}

type FetchTask struct {
	ID              string        `json:"id"`
	CampaignID      string        `json:"campaign_id"`
	URL             string        `json:"url"`
	Mode            string        `json:"mode"`     // http, browser, git, file
	Executor        string        `json:"executor"` // native, scrapy, crawl4ai
	Deadline        time.Time     `json:"deadline"`
	Timeout         time.Duration `json:"timeout"`
	MaxBytes        int64         `json:"max_bytes"`
	IfNoneMatch     string        `json:"if_none_match,omitempty"`
	IfModifiedSince string        `json:"if_modified_since,omitempty"`
	DomainPolicy    string        `json:"domain_policy"`
	CreatedAt       time.Time     `json:"created_at"`
}

type FetchResult struct {
	TaskID       string `json:"task_id"`
	EffectiveURL string `json:"effective_url"`
	StatusCode   int    `json:"status_code"`
	ContentType  string `json:"content_type"`
	RawContent   []byte `json:"-"`
	// RenderedContent é o DOM pós-JavaScript (derivado). Quando presente, a
	// extração usa este conteúdo; o acervo CAS sempre guarda RawContent.
	RenderedContent   []byte         `json:"-"`
	Text              string         `json:"text"`
	Title             string         `json:"title,omitempty"`
	ETag              string         `json:"etag,omitempty"`
	LastModified      string         `json:"last_modified,omitempty"`
	ExtractedLinks    []string       `json:"extracted_links"`
	ExtractedMentions []string       `json:"extracted_mentions"`
	Duration          time.Duration  `json:"duration"`
	Error             string         `json:"error,omitempty"`
	Metadata          map[string]any `json:"metadata,omitempty"`
}
