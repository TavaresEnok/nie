package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"nie/internal/domain"
	"nie/internal/urlutil"
)

var (
	ErrNotFound = errors.New("não encontrado")
	ErrNoJob    = errors.New("nenhum job disponível")
)

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, dsn string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("config do banco: %w", err)
	}
	cfg.MaxConns = 12
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("conectar ao banco: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Pool() *pgxpool.Pool { return s.pool }
func (s *Store) Close()              { s.pool.Close() }

func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

const campaignSelect = `
SELECT c.id::text, c.query, c.status, c.seed_urls, c.max_depth, c.max_urls,
       c.max_domains, c.max_bytes, c.min_domain_interval_seconds,
       c.created_at, c.updated_at, c.finished_at,
       COALESCE(j.pending, 0)::int, COALESCE(j.running, 0)::int,
       COALESCE(j.done, 0)::int, COALESCE(j.failed, 0)::int, COALESCE(j.dead, 0)::int,
       COALESCE(dv.documents, 0)::int, COALESCE(p.passages, 0)::int,
       COALESCE(j.domains, 0)::int, COALESCE(dv.bytes, 0)::bigint
FROM campaigns c
LEFT JOIN LATERAL (
    SELECT count(*) FILTER (WHERE status = 'pending') AS pending,
           count(*) FILTER (WHERE status = 'running') AS running,
           count(*) FILTER (WHERE status = 'done')    AS done,
           count(*) FILTER (WHERE status = 'dead')    AS failed,
           count(*) FILTER (WHERE status = 'dead')    AS dead,
           count(DISTINCT domain) FILTER (WHERE domain <> '') AS domains
    FROM jobs WHERE campaign_id = c.id
) j ON true
LEFT JOIN LATERAL (
    SELECT count(DISTINCT p.id)::int AS passages
    FROM passages p
    JOIN document_versions v ON v.document_id = p.document_id
    WHERE v.campaign_id = c.id
) p ON true
LEFT JOIN LATERAL (
    SELECT count(*)::int AS documents, COALESCE(sum(x.size), 0)::bigint AS bytes
    FROM (SELECT DISTINCT d.id, d.size
          FROM document_versions v JOIN documents d ON d.id = v.document_id
          WHERE v.campaign_id = c.id) x
) dv ON true
`

func scanCampaign(row pgx.Row) (domain.Campaign, error) {
	var c domain.Campaign
	var seeds []byte
	var finished *time.Time
	err := row.Scan(
		&c.ID, &c.Query, &c.Status, &seeds, &c.Budget.MaxDepth, &c.Budget.MaxURLs,
		&c.Budget.MaxDomains, &c.Budget.MaxBytes, &c.Budget.MinDomainIntervalSeconds,
		&c.CreatedAt, &c.UpdatedAt, &finished,
		&c.Stats.JobsPending, &c.Stats.JobsRunning, &c.Stats.JobsDone,
		&c.Stats.JobsFailed, &c.Stats.JobsDead, &c.Stats.Documents, &c.Stats.Passages,
		&c.Stats.Domains, &c.Stats.Bytes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Campaign{}, ErrNotFound
		}
		return domain.Campaign{}, err
	}
	c.FinishedAt = finished
	_ = json.Unmarshal(seeds, &c.SeedURLs)
	if c.SeedURLs == nil {
		c.SeedURLs = []string{}
	}
	return c, nil
}

func (s *Store) CreateCampaign(ctx context.Context, query string, seeds []string, b domain.Budget) (domain.Campaign, error) {
	normalized := make([]string, 0, len(seeds))
	seen := make(map[string]struct{})
	for _, raw := range seeds {
		norm, err := urlutil.Normalize(raw)
		if err != nil {
			return domain.Campaign{}, fmt.Errorf("%w: %s", err, raw)
		}
		u, _, err := urlutil.ParseNormalized(norm)
		if err != nil {
			return domain.Campaign{}, fmt.Errorf("%w: %s", err, raw)
		}
		if err := urlutil.ValidateCrawlURL(u); err != nil {
			return domain.Campaign{}, fmt.Errorf("%w: %s", err, raw)
		}
		if _, ok := seen[norm]; ok {
			continue
		}
		seen[norm] = struct{}{}
		normalized = append(normalized, norm)
	}
	seedJSON, _ := json.Marshal(normalized)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.Campaign{}, err
	}
	defer tx.Rollback(ctx)

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO campaigns(query, status, seed_urls, max_depth, max_urls, max_domains, max_bytes, min_domain_interval_seconds)
		VALUES($1, 'queued', $2, $3, $4, $5, $6, $7)
		RETURNING id::text`,
		query, seedJSON, b.MaxDepth, b.MaxURLs, b.MaxDomains, b.MaxBytes, b.MinDomainIntervalSeconds,
	).Scan(&id)
	if err != nil {
		return domain.Campaign{}, fmt.Errorf("criar campanha: %w", err)
	}
	for _, norm := range normalized {
		_, err = tx.Exec(ctx, `
			INSERT INTO jobs(campaign_id, kind, url, url_hash, domain, depth, priority, reason)
			VALUES($1::uuid, 'fetch', $2, $3, $4, 0, 10, 'seed')
			ON CONFLICT DO NOTHING`,
			id, norm, urlutil.Hash(norm), urlutil.Domain(norm))
		if err != nil {
			return domain.Campaign{}, fmt.Errorf("enfileirar seed: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Campaign{}, err
	}
	return s.GetCampaign(ctx, id)
}

func (s *Store) ListCampaigns(ctx context.Context, limit int) ([]domain.Campaign, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, campaignSelect+` ORDER BY c.created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Campaign, 0)
	for rows.Next() {
		c, err := scanCampaign(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) GetCampaign(ctx context.Context, id string) (domain.Campaign, error) {
	if !validUUID(id) {
		return domain.Campaign{}, ErrNotFound
	}
	return scanCampaign(s.pool.QueryRow(ctx, campaignSelect+` WHERE c.id = $1::uuid`, id))
}

func (s *Store) SetCampaignStatus(ctx context.Context, id, status string) error {
	if !validUUID(id) {
		return ErrNotFound
	}
	var cmd string
	switch status {
	case "running", "queued":
		cmd = `UPDATE campaigns SET status='running', updated_at=now(), finished_at=NULL WHERE id=$1::uuid AND status IN ('paused','queued','completed','failed')`
	case "paused":
		cmd = `UPDATE campaigns SET status='paused', updated_at=now() WHERE id=$1::uuid AND status IN ('queued','running')`
	default:
		return fmt.Errorf("status inválido")
	}
	tag, err := s.pool.Exec(ctx, cmd, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type DocumentRecord struct {
	SHA256        string
	MIME          string
	Title         string
	Size          int64
	Text          string
	Metadata      map[string]any
	CampaignID    string
	URL           string
	NormalizedURL string
	Domain        string
	StatusCode    int
	ETag          string
	LastModified  string
	Passages      []domain.Passage
}

func (s *Store) RecordDocument(ctx context.Context, rec DocumentRecord) (string, bool, error) {
	meta, _ := json.Marshal(rec.Metadata)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", false, err
	}
	defer tx.Rollback(ctx)

	var docID string
	created := false
	err = tx.QueryRow(ctx, `
		INSERT INTO documents(sha256, mime, title, size, text, metadata)
		VALUES($1, $2, $3, $4, $5, $6)
		ON CONFLICT (sha256) DO NOTHING
		RETURNING id::text`,
		rec.SHA256, rec.MIME, rec.Title, rec.Size, rec.Text, meta,
	).Scan(&docID)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := tx.QueryRow(ctx, `SELECT id::text FROM documents WHERE sha256=$1`, rec.SHA256).Scan(&docID); err != nil {
			return "", false, err
		}
	} else if err != nil {
		return "", false, fmt.Errorf("inserir documento: %w", err)
	} else {
		created = true
	}

	if created {
		for _, p := range rec.Passages {
			_, err := tx.Exec(ctx, `
				INSERT INTO passages(document_id, ordinal, page, text)
				VALUES($1::uuid, $2, $3, $4)`,
				docID, p.Ordinal, p.Page, p.Text)
			if err != nil {
				return "", false, fmt.Errorf("inserir trecho: %w", err)
			}
		}
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO document_versions(document_id, campaign_id, url, normalized_url, domain, status_code, etag, last_modified)
		VALUES($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (campaign_id, document_id, normalized_url)
		DO UPDATE SET fetched_at=now(), status_code=EXCLUDED.status_code,
		              etag=EXCLUDED.etag, last_modified=EXCLUDED.last_modified`,
		docID, rec.CampaignID, rec.URL, rec.NormalizedURL, rec.Domain, rec.StatusCode, rec.ETag, rec.LastModified)
	if err != nil {
		return "", false, fmt.Errorf("inserir versão: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", false, err
	}
	return docID, created, nil
}

func (s *Store) EnqueueFetch(ctx context.Context, campaignID, rawURL string, depth int, parentURL, reason string, priority int) (bool, error) {
	norm, err := urlutil.Normalize(rawURL)
	if err != nil {
		return false, nil
	}
	u, _, err := urlutil.ParseNormalized(norm)
	if err != nil || urlutil.ValidateCrawlURL(u) != nil {
		return false, nil
	}
	domainName := urlutil.Domain(norm)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	var maxURLs, maxDomains int
	var status string
	err = tx.QueryRow(ctx, `
		SELECT max_urls, max_domains, status FROM campaigns WHERE id=$1::uuid FOR UPDATE`, campaignID).
		Scan(&maxURLs, &maxDomains, &status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, ErrNotFound
		}
		return false, err
	}

	var used, domains int
	if err := tx.QueryRow(ctx, `
		SELECT count(*)::int, count(DISTINCT domain) FILTER (WHERE domain <> '')::int
		FROM jobs WHERE campaign_id=$1::uuid`, campaignID).Scan(&used, &domains); err != nil {
		return false, err
	}
	if used >= maxURLs {
		return false, nil
	}
	var already bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM jobs WHERE campaign_id=$1::uuid AND domain=$2 AND status<>'dead')`,
		campaignID, domainName).Scan(&already); err != nil {
		return false, err
	}
	if !already && domains >= maxDomains {
		return false, nil
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO jobs(campaign_id, kind, url, url_hash, domain, depth, priority, parent_url, reason)
		VALUES($1::uuid, 'fetch', $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT DO NOTHING`,
		campaignID, norm, urlutil.Hash(norm), domainName, depth, priority, parentURL, reason)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() > 0 {
		_, _ = tx.Exec(ctx, `
			INSERT INTO discovery_edges(campaign_id, from_url, to_url, to_domain, kind, depth, reason)
			VALUES($1::uuid, $2, $3, $4, 'link', $5, $6)
			ON CONFLICT DO NOTHING`,
			campaignID, parentURL, norm, domainName, depth, reason)
	}
	if status == "completed" {
		_, _ = tx.Exec(ctx, `UPDATE campaigns SET status='running', updated_at=now(), finished_at=NULL WHERE id=$1::uuid`, campaignID)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *Store) RecordDiscoveryEdge(ctx context.Context, campaignID, fromURL, toURL, kind, reason string, depth int) error {
	toDomain := urlutil.Domain(toURL)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO discovery_edges(campaign_id, from_url, to_url, to_domain, kind, depth, reason)
		VALUES($1::uuid, NULLIF($2, ''), $3, $4, $5, $6, $7)
		ON CONFLICT (campaign_id, to_url) DO NOTHING
	`, campaignID, fromURL, toURL, toDomain, kind, depth, reason)
	return err
}

func (s *Store) EnqueueDiscovery(ctx context.Context, campaignID, kind, query string) (bool, error) {
	if kind != "discover_web" && kind != "discover_github" && kind != "discover_mwmbl" && kind != "discover_swh" {
		return false, fmt.Errorf("kind de descoberta inválido")
	}
	payload, _ := json.Marshal(map[string]string{"query": query})
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	var maxURLs int
	err = tx.QueryRow(ctx, `SELECT max_urls FROM campaigns WHERE id=$1::uuid FOR UPDATE`, campaignID).Scan(&maxURLs)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, ErrNotFound
		}
		return false, err
	}
	var used int
	if err := tx.QueryRow(ctx, `SELECT count(*)::int FROM jobs WHERE campaign_id=$1::uuid`, campaignID).Scan(&used); err != nil {
		return false, err
	}
	if used >= maxURLs {
		return false, nil
	}
	tag, err := tx.Exec(ctx, `
		INSERT INTO jobs(campaign_id, kind, payload, priority, reason)
		VALUES($1::uuid, $2, $3, 5, $2)`, campaignID, kind, payload)
	if err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *Store) ClaimJob(ctx context.Context, lease time.Duration) (*domain.Job, error) {
	var j domain.Job
	var payload []byte
	var lastErr *string
	var leaseUntil *time.Time
	err := s.pool.QueryRow(ctx, `
		WITH candidate AS (
			SELECT j.id
			FROM jobs j JOIN campaigns c ON c.id = j.campaign_id
			WHERE c.status IN ('queued','running')
			  AND j.attempts < j.max_attempts
			  AND (
			    (j.status='pending' AND j.scheduled_at <= now())
			    OR (j.status='running' AND j.lease_until IS NOT NULL AND j.lease_until < now())
			  )
			ORDER BY j.priority ASC, j.id ASC
			FOR UPDATE OF j SKIP LOCKED
			LIMIT 1
		)
		UPDATE jobs j
		SET status='running', attempts=j.attempts+1, lease_until=now()+$1::interval, updated_at=now()
		FROM candidate
		WHERE j.id=candidate.id
		RETURNING j.id, j.campaign_id::text, j.kind, COALESCE(j.url,''), COALESCE(j.url_hash,''),
		          j.domain, j.depth, j.priority, COALESCE(j.parent_url,''), COALESCE(j.reason,''),
		          j.payload, j.status, j.attempts, j.max_attempts, j.scheduled_at, j.lease_until,
		          j.last_error, j.created_at`,
		fmt.Sprintf("%d seconds", int(lease.Seconds())),
	).Scan(&j.ID, &j.CampaignID, &j.Kind, &j.URL, &j.NormalizedURL, &j.Domain, &j.Depth,
		&j.Priority, &j.ParentURL, &j.Reason, &payload, &j.Status, &j.Attempts,
		&j.MaxAttempts, &j.ScheduledAt, &leaseUntil, &lastErr, &j.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	j.Payload = payload
	j.LeaseUntil = leaseUntil
	if lastErr != nil {
		j.LastError = *lastErr
	}
	return &j, nil
}

func (s *Store) CompleteJob(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `UPDATE jobs SET status='done', lease_until=NULL, updated_at=now() WHERE id=$1`, id)
	return err
}

// ReapExpiredJobs encerra jobs cujo lease expirou e que já esgotaram as tentativas.
func (s *Store) ReapExpiredJobs(ctx context.Context) (int64, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE jobs
		SET status='dead', lease_until=NULL,
		    last_error=COALESCE(NULLIF(last_error, ''), 'lease expirado'),
		    updated_at=now()
		WHERE status='running' AND lease_until IS NOT NULL AND lease_until < now()
		  AND attempts >= max_attempts`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// FinishIdleCampaigns conclui campanhas sem jobs pendentes ou em execução.
func (s *Store) FinishIdleCampaigns(ctx context.Context) (int64, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE campaigns c
		SET status='completed', updated_at=now(), finished_at=now()
		WHERE c.status IN ('queued','running')
		  AND NOT EXISTS (
		    SELECT 1 FROM jobs j
		    WHERE j.campaign_id=c.id AND j.status IN ('pending','running')
		  )`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (s *Store) RetryJob(ctx context.Context, id int64, delay time.Duration, cause error) error {
	msg := ""
	if cause != nil {
		msg = cause.Error()
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE jobs SET status='pending', lease_until=NULL, scheduled_at=now()+$2::interval,
		                last_error=$3, updated_at=now()
		WHERE id=$1`, id, fmt.Sprintf("%d seconds", int(delay.Seconds())), msg)
	return err
}

func (s *Store) DeadJob(ctx context.Context, id int64, cause error) error {
	msg := ""
	if cause != nil {
		msg = cause.Error()
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE jobs SET status='dead', lease_until=NULL, last_error=$2, updated_at=now() WHERE id=$1`, id, msg)
	return err
}

func (s *Store) DeferJob(ctx context.Context, id int64, at time.Time) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE jobs SET status='pending', lease_until=NULL, scheduled_at=$2,
		                attempts=GREATEST(attempts-1, 0), updated_at=now()
		WHERE id=$1`, id, at)
	return err
}

func (s *Store) FinishIfIdle(ctx context.Context, campaignID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var status string
	if err := tx.QueryRow(ctx, `SELECT status FROM campaigns WHERE id=$1::uuid FOR UPDATE`, campaignID).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	var active int
	if err := tx.QueryRow(ctx, `SELECT count(*)::int FROM jobs WHERE campaign_id=$1::uuid AND status IN ('pending','running')`, campaignID).Scan(&active); err != nil {
		return err
	}
	if active == 0 && (status == "queued" || status == "running") {
		if _, err := tx.Exec(ctx, `UPDATE campaigns SET status='completed', updated_at=now(), finished_at=now() WHERE id=$1::uuid`, campaignID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) CampaignContext(ctx context.Context, campaignID string) (domain.Budget, int64, error) {
	var b domain.Budget
	var used int64
	err := s.pool.QueryRow(ctx, `
		SELECT c.max_depth, c.max_urls, c.max_domains, c.max_bytes, c.min_domain_interval_seconds,
		       COALESCE((
			SELECT sum(x.size) FROM (
				SELECT DISTINCT d.id, d.size
				FROM document_versions v JOIN documents d ON d.id=v.document_id
				WHERE v.campaign_id=c.id
			) x
		), 0)::bigint
		FROM campaigns c WHERE c.id=$1::uuid`, campaignID).
		Scan(&b.MaxDepth, &b.MaxURLs, &b.MaxDomains, &b.MaxBytes, &b.MinDomainIntervalSeconds, &used)
	if errors.Is(err, pgx.ErrNoRows) {
		return b, 0, ErrNotFound
	}
	return b, used, err
}

// Throttle retorna (próximaJanela, permitido). Quando permitido=false, o job deve ser adiado.
func (s *Store) Throttle(ctx context.Context, domainName string, interval time.Duration) (time.Time, bool, error) {
	secs := int(interval.Seconds())
	if secs < 1 {
		secs = 1
	}
	var next time.Time
	err := s.pool.QueryRow(ctx, `
		INSERT INTO domain_state(domain, next_allowed_at)
		VALUES($1, now() + make_interval(secs => $2::double precision))
		ON CONFLICT (domain) DO UPDATE
		SET next_allowed_at = now() + make_interval(secs => $2::double precision), updated_at=now()
		WHERE domain_state.next_allowed_at <= now()
		RETURNING next_allowed_at`, domainName, secs).Scan(&next)
	if err == nil {
		return next, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, false, err
	}
	if err := s.pool.QueryRow(ctx, `SELECT next_allowed_at FROM domain_state WHERE domain=$1`, domainName).Scan(&next); err != nil {
		return time.Time{}, false, err
	}
	return next, false, nil
}

func (s *Store) ListJobs(ctx context.Context, campaignID, status string, limit int) ([]domain.Job, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	args := []any{}
	where := []string{"1=1"}
	if campaignID != "" {
		if !validUUID(campaignID) {
			return nil, ErrNotFound
		}
		args = append(args, campaignID)
		where = append(where, fmt.Sprintf("campaign_id=$%d::uuid", len(args)))
	}
	if status != "" {
		args = append(args, status)
		where = append(where, fmt.Sprintf("status=$%d", len(args)))
	}
	args = append(args, limit)
	rows, err := s.pool.Query(ctx, `
		SELECT id, campaign_id::text, kind, COALESCE(url,''), COALESCE(url_hash,''), domain,
		       depth, priority, COALESCE(parent_url,''), COALESCE(reason,''), payload, status,
		       attempts, max_attempts, scheduled_at, lease_until, last_error, created_at
		FROM jobs WHERE `+strings.Join(where, " AND ")+`
		ORDER BY id DESC LIMIT $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Job, 0)
	for rows.Next() {
		var j domain.Job
		var payload []byte
		var lastErr *string
		var lease *time.Time
		if err := rows.Scan(&j.ID, &j.CampaignID, &j.Kind, &j.URL, &j.NormalizedURL, &j.Domain,
			&j.Depth, &j.Priority, &j.ParentURL, &j.Reason, &payload, &j.Status, &j.Attempts,
			&j.MaxAttempts, &j.ScheduledAt, &lease, &lastErr, &j.CreatedAt); err != nil {
			return nil, err
		}
		j.Payload = payload
		j.LeaseUntil = lease
		if lastErr != nil {
			j.LastError = *lastErr
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func (s *Store) ListEdges(ctx context.Context, campaignID string, limit int) ([]domain.DiscoveryEdge, error) {
	if !validUUID(campaignID) {
		return nil, ErrNotFound
	}
	if limit <= 0 || limit > 2000 {
		limit = 500
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, campaign_id::text, from_url, to_url, to_domain, kind, depth, reason, created_at
		FROM discovery_edges WHERE campaign_id=$1::uuid ORDER BY id LIMIT $2`, campaignID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.DiscoveryEdge, 0)
	for rows.Next() {
		var e domain.DiscoveryEdge
		if err := rows.Scan(&e.ID, &e.CampaignID, &e.FromURL, &e.ToURL, &e.ToDomain, &e.Kind, &e.Depth, &e.Reason, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func validUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, r := range s {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
				return false
			}
		}
	}
	return true
}
