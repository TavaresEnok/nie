package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"nie/internal/domain"
)

func (s *Store) CreateSchedule(ctx context.Context, name, query string, seedURLs []string, intervalSeconds int, maxDepth, maxURLs, maxDomains int) (domain.Schedule, error) {
	seedsJSON, _ := json.Marshal(seedURLs)
	var sch domain.Schedule
	var seedsData []byte
	err := s.pool.QueryRow(ctx, `
		INSERT INTO schedules (name, query, seed_urls, interval_seconds, max_depth, max_urls, max_domains)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id::text, name, query, seed_urls, interval_seconds, max_depth, max_urls, max_domains,
		          enabled, last_run_at, next_run_at, last_campaign_id::text, created_at, updated_at`,
		name, query, seedsJSON, intervalSeconds, maxDepth, maxURLs, maxDomains).
		Scan(&sch.ID, &sch.Name, &sch.Query, &seedsData, &sch.IntervalSeconds, &sch.MaxDepth, &sch.MaxURLs, &sch.MaxDomains,
			&sch.Enabled, &sch.LastRunAt, &sch.NextRunAt, &sch.LastCampaignID, &sch.CreatedAt, &sch.UpdatedAt)
	if err != nil {
		return domain.Schedule{}, fmt.Errorf("criar agendamento: %w", err)
	}
	_ = json.Unmarshal(seedsData, &sch.SeedURLs)
	return sch, nil
}

func (s *Store) ListSchedules(ctx context.Context) ([]domain.Schedule, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, name, query, seed_urls, interval_seconds, max_depth, max_urls, max_domains,
		       enabled, last_run_at, next_run_at, last_campaign_id::text, created_at, updated_at
		FROM schedules
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("listar agendamentos: %w", err)
	}
	defer rows.Close()

	var out []domain.Schedule
	for rows.Next() {
		var sch domain.Schedule
		var seedsData []byte
		if err := rows.Scan(&sch.ID, &sch.Name, &sch.Query, &seedsData, &sch.IntervalSeconds,
			&sch.MaxDepth, &sch.MaxURLs, &sch.MaxDomains, &sch.Enabled, &sch.LastRunAt,
			&sch.NextRunAt, &sch.LastCampaignID, &sch.CreatedAt, &sch.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(seedsData, &sch.SeedURLs)
		out = append(out, sch)
	}
	return out, rows.Err()
}

func (s *Store) ListDueSchedules(ctx context.Context) ([]domain.Schedule, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, name, query, seed_urls, interval_seconds, max_depth, max_urls, max_domains,
		       enabled, last_run_at, next_run_at, last_campaign_id::text, created_at, updated_at
		FROM schedules
		WHERE enabled = true AND next_run_at <= now()
		ORDER BY next_run_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("listar agendamentos vencidos: %w", err)
	}
	defer rows.Close()

	var out []domain.Schedule
	for rows.Next() {
		var sch domain.Schedule
		var seedsData []byte
		if err := rows.Scan(&sch.ID, &sch.Name, &sch.Query, &seedsData, &sch.IntervalSeconds,
			&sch.MaxDepth, &sch.MaxURLs, &sch.MaxDomains, &sch.Enabled, &sch.LastRunAt,
			&sch.NextRunAt, &sch.LastCampaignID, &sch.CreatedAt, &sch.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(seedsData, &sch.SeedURLs)
		out = append(out, sch)
	}
	return out, rows.Err()
}

func (s *Store) MarkScheduleRun(ctx context.Context, scheduleID, campaignID string, nextRun time.Time) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE schedules
		SET last_run_at = now(),
		    next_run_at = $2,
		    last_campaign_id = $3,
		    updated_at = now()
		WHERE id = $1`, scheduleID, nextRun, campaignID)
	return err
}

func (s *Store) CreateAlert(ctx context.Context, campaignID *string, title, message, severity string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO alerts (campaign_id, title, message, severity)
		VALUES ($1, $2, $3, $4)`, campaignID, title, message, severity)
	return err
}

func (s *Store) ListAlerts(ctx context.Context, limit int) ([]domain.Alert, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, campaign_id::text, title, message, severity, acknowledged, created_at
		FROM alerts
		ORDER BY created_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("listar alertas: %w", err)
	}
	defer rows.Close()

	var out []domain.Alert
	for rows.Next() {
		var a domain.Alert
		var cID *string
		if err := rows.Scan(&a.ID, &cID, &a.Title, &a.Message, &a.Severity, &a.Acknowledged, &a.CreatedAt); err != nil {
			return nil, err
		}
		a.CampaignID = cID
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) AcknowledgeAlert(ctx context.Context, alertID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE alerts SET acknowledged = true WHERE id = $1`, alertID)
	return err
}

func (s *Store) RecordDocumentDiff(ctx context.Context, documentID, url, oldSHA, newSHA, summary string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO document_diffs (document_id, url, old_sha256, new_sha256, summary)
		VALUES ($1, $2, $3, $4, $5)`, documentID, url, oldSHA, newSHA, summary)
	return err
}
