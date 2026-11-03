package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"nie/internal/domain"
	"nie/internal/urlutil"
)

// ConfigureAutomaticResearch deve ser chamado antes de iniciar os workers.
// O limite inclui a avaliação final e é limitado para evitar campanhas infinitas.
func (s *Store) ConfigureAutomaticResearch(rounds int) {
	s.researchRounds = max(0, min(rounds, 10))
}

type ResearchQuery struct {
	Kind   string `json:"kind"`
	Query  string `json:"query"`
	Reason string `json:"reason"`
}

type ResearchQuestion struct {
	CampaignID   string `json:"campaign_id"`
	Ordinal      int    `json:"ordinal"`
	Text         string `json:"text"`
	Origin       string `json:"origin"`
	Status       string `json:"status"`
	EvidenceNote string `json:"evidence_note"`
}

type ResearchRound struct {
	JobID           int64              `json:"job_id"`
	CampaignID      string             `json:"campaign_id"`
	Number          int                `json:"round"`
	Decision        string             `json:"decision"`
	Reason          string             `json:"reason"`
	ClaimsCount     int                `json:"claims_count"`
	PassagesCount   int                `json:"passages_count"`
	EnqueuedQueries int                `json:"enqueued_queries"`
	Leads           []ResearchQuery    `json:"leads"`
	Questions       []ResearchQuestion `json:"questions,omitempty"`
}

// CampaignDiscoveryKinds restringe novas rodadas aos provedores que já foram
// selecionados na campanha, em vez de habilitar todas as fontes do servidor.
func (s *Store) CampaignDiscoveryKinds(ctx context.Context, campaignID string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT kind FROM jobs WHERE campaign_id=$1::uuid AND kind LIKE 'discover_%' ORDER BY kind`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var kinds []string
	for rows.Next() {
		var kind string
		if err := rows.Scan(&kind); err != nil {
			return nil, err
		}
		kinds = append(kinds, kind)
	}
	return kinds, rows.Err()
}

// CommitResearchRound publica decisão e novas tarefas na mesma transação.
// Reexecuções não duplicam pesquisas; pausa e lease perdido impedem a publicação.
func (s *Store) CommitResearchRound(ctx context.Context, job *domain.Job, round ResearchRound) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var status string
	if err := tx.QueryRow(ctx, `SELECT status FROM campaigns WHERE id=$1::uuid FOR UPDATE`, job.CampaignID).Scan(&status); err != nil {
		return err
	}
	if status != "running" && status != "queued" {
		return fmt.Errorf("campanha não está ativa: %s", status)
	}
	var owns bool
	if err := tx.QueryRow(ctx, `SELECT status='running' AND attempts=$2 AND lease_generation=$3 AND lease_until>now() FROM jobs WHERE id=$1 FOR UPDATE`, job.ID, job.Attempts, job.LeaseGeneration).Scan(&owns); err != nil {
		return err
	}
	if !owns {
		return errors.New("lease da investigação perdido")
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM research_rounds WHERE job_id=$1)`, job.ID).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return tx.Commit(ctx)
	}

	var used, fetches int
	var exhausted bool
	if err := tx.QueryRow(ctx, `SELECT count(*)::int FROM jobs WHERE campaign_id=$1::uuid AND kind LIKE 'discover_%'`, job.CampaignID).Scan(&used); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `SELECT count(*)::int FROM jobs WHERE campaign_id=$1::uuid AND kind='fetch'`, job.CampaignID).Scan(&fetches); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `SELECT $2::int >= max_urls OR COALESCE((SELECT sum(d.size) FROM documents d WHERE EXISTS(SELECT 1 FROM document_versions v WHERE v.document_id=d.id AND v.campaign_id=c.id)),0)>=max_bytes FROM campaigns c WHERE id=$1::uuid`, job.CampaignID, fetches).Scan(&exhausted); err != nil {
		return err
	}
	switch {
	case exhausted || used >= MaxDiscoveryJobsPerCampaign:
		round.Decision, round.Reason = "stop", "budget_exhausted"
	case round.Number >= s.researchRounds:
		round.Decision, round.Reason = "stop", "round_limit"
	}
	if round.Decision == "continue" {
		for _, lead := range round.Leads {
			if used >= MaxDiscoveryJobsPerCampaign {
				break
			}
			switch lead.Kind {
			case "discover_web", "discover_mwmbl", "discover_github", "discover_gitlab", "discover_swh", "discover_commoncrawl":
			default:
				return fmt.Errorf("provedor inválido: %s", lead.Kind)
			}
			norm := NormalizeQuery(lead.Query)
			if norm == "" {
				continue
			}
			var allowed bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM jobs WHERE campaign_id=$1::uuid AND kind=$2)`, job.CampaignID, lead.Kind).Scan(&allowed); err != nil {
				return err
			}
			if !allowed {
				return fmt.Errorf("provedor não selecionado nesta campanha: %s", lead.Kind)
			}
			payload, err := json.Marshal(map[string]any{"query": lead.Query, "research_round": round.Number, "research_job_id": job.ID})
			if err != nil {
				return err
			}
			tag, err := tx.Exec(ctx, `INSERT INTO jobs(campaign_id,kind,url_hash,payload,priority,reason) VALUES($1::uuid,$2,$3,$4,5,$5) ON CONFLICT DO NOTHING`, job.CampaignID, lead.Kind, urlutil.Hash(lead.Kind+"|"+norm), payload, lead.Reason)
			if err != nil {
				return err
			}
			if tag.RowsAffected() > 0 {
				round.EnqueuedQueries++
				used++
			}
		}
		if round.EnqueuedQueries == 0 {
			round.Decision, round.Reason = "stop", "no_new_queries"
		}
	}
	leads, err := json.Marshal(round.Leads)
	if err != nil {
		return err
	}
	for i, question := range round.Questions {
		switch question.Status {
		case "open", "queued", "addressed":
		default:
			return fmt.Errorf("estado de subpergunta inválido: %s", question.Status)
		}
		if question.Ordinal < 1 || strings.TrimSpace(question.Text) == "" || question.Origin == "" {
			return fmt.Errorf("subpergunta inválida")
		}
		if question.Status == "open" {
			var sent bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM jobs WHERE campaign_id=$1::uuid AND kind LIKE 'discover_%' AND lower(regexp_replace(btrim(COALESCE(payload->>'query','')), '\s+', ' ', 'g'))=$2)`, job.CampaignID, NormalizeQuery(question.Text)).Scan(&sent); err != nil {
				return err
			}
			if sent {
				question.Status = "queued"
				question.EvidenceNote = "consulta encaminhada à descoberta"
				round.Questions[i] = question
			}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO research_questions(campaign_id,ordinal,text,origin,status,evidence_note)
			VALUES($1::uuid,$2,$3,$4,$5,$6)
			ON CONFLICT (campaign_id, ordinal) DO UPDATE SET text=EXCLUDED.text, origin=EXCLUDED.origin, status=EXCLUDED.status, evidence_note=EXCLUDED.evidence_note, updated_at=now()`,
			job.CampaignID, question.Ordinal, question.Text, question.Origin, question.Status, question.EvidenceNote); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO research_rounds(job_id,campaign_id,round_number,decision,reason,claims_count,passages_count,enqueued_queries,leads) VALUES($1,$2::uuid,$3,$4,$5,$6,$7,$8,$9)`, job.ID, job.CampaignID, round.Number, round.Decision, round.Reason, round.ClaimsCount, round.PassagesCount, round.EnqueuedQueries, leads); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ListDiscoveryQueries devolve as consultas de descoberta já gravadas, normalizadas.
func (s *Store) ListDiscoveryQueries(ctx context.Context, campaignID string) (map[string]bool, error) {
	rows, err := s.pool.Query(ctx, `SELECT COALESCE(payload->>'query','') FROM jobs WHERE campaign_id=$1::uuid AND kind LIKE 'discover_%'`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var query string
		if err := rows.Scan(&query); err != nil {
			return nil, err
		}
		if norm := NormalizeQuery(query); norm != "" {
			out[norm] = true
		}
	}
	return out, rows.Err()
}

func (s *Store) ListResearchQuestions(ctx context.Context, campaignID string) ([]ResearchQuestion, error) {
	if !validUUID(campaignID) {
		return nil, ErrNotFound
	}
	rows, err := s.pool.Query(ctx, `SELECT campaign_id::text,ordinal,text,origin,status,evidence_note FROM research_questions WHERE campaign_id=$1::uuid ORDER BY ordinal`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ResearchQuestion{}
	for rows.Next() {
		var q ResearchQuestion
		if err := rows.Scan(&q.CampaignID, &q.Ordinal, &q.Text, &q.Origin, &q.Status, &q.EvidenceNote); err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

func (s *Store) ListResearchRounds(ctx context.Context, campaignID string) ([]ResearchRound, error) {
	if !validUUID(campaignID) {
		return nil, ErrNotFound
	}
	rows, err := s.pool.Query(ctx, `SELECT job_id,campaign_id::text,round_number,decision,reason,claims_count,passages_count,enqueued_queries,leads FROM research_rounds WHERE campaign_id=$1::uuid ORDER BY round_number,job_id`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ResearchRound{}
	for rows.Next() {
		var r ResearchRound
		var leads []byte
		if err := rows.Scan(&r.JobID, &r.CampaignID, &r.Number, &r.Decision, &r.Reason, &r.ClaimsCount, &r.PassagesCount, &r.EnqueuedQueries, &leads); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(leads, &r.Leads); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
