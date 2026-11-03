package store

import (
	"context"
	"fmt"
	"nie/internal/domain"
	"os"
	"sync"
	"testing"
	"time"
)

func TestAutomaticResearchIntegration(t *testing.T) {
	dsn := os.Getenv("NIE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("NIE_TEST_DATABASE_URL não definido")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	st, err := New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	st.ConfigureAutomaticResearch(3)
	camp, err := st.CreateCampaign(ctx, "research integration", nil, domain.Budget{MaxURLs: 10, MaxDepth: 2, MaxDomains: 5, MaxBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	defer st.pool.Exec(context.Background(), `DELETE FROM campaigns WHERE id=$1::uuid`, camp.ID)
	if _, err := st.EnqueueDiscovery(ctx, camp.ID, "discover_web", "initial"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.pool.Exec(ctx, `UPDATE jobs SET status='done' WHERE campaign_id=$1::uuid`, camp.ID); err != nil {
		t.Fatal(err)
	}
	// Vários workers e o reaper observam o mesmo fim de rodada.
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				_, err := st.FinishIdleCampaigns(ctx)
				errs <- err
			} else {
				errs <- st.FinishIfIdle(ctx, camp.ID)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := st.pool.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE campaign_id=$1::uuid AND kind='investigate'`, camp.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("uma única avaliação: count=%d err=%v", count, err)
	}
	job, err := st.ClaimJob(ctx, time.Minute)
	if err != nil || job == nil || job.Kind != "investigate" {
		t.Fatalf("avaliação não agendada: job=%+v err=%v", job, err)
	}
	round := ResearchRound{Number: 1, Decision: "continue", Reason: "missing_version", Leads: []ResearchQuery{{Kind: "discover_web", Query: "release compatibility", Reason: "missing_version"}}}
	if err := st.SetCampaignStatus(ctx, camp.ID, "paused"); err != nil {
		t.Fatal(err)
	}
	if err := st.CommitResearchRound(ctx, job, round); err == nil {
		t.Fatal("campanha pausada publicou buscas")
	}
	if err := st.SetCampaignStatus(ctx, camp.ID, "running"); err != nil {
		t.Fatal(err)
	}
	unauthorized := round
	unauthorized.Leads = []ResearchQuery{{Kind: "discover_github", Query: "must not run"}}
	if err := st.CommitResearchRound(ctx, job, unauthorized); err == nil {
		t.Fatal("provedor não selecionado foi habilitado")
	}
	if _, err := st.pool.Exec(ctx, `UPDATE jobs SET lease_until=now()-interval '1 second' WHERE id=$1`, job.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.CommitResearchRound(ctx, job, round); err == nil {
		t.Fatal("lease expirado publicou buscas")
	}
	recovered, err := st.ClaimJob(ctx, time.Minute)
	if err != nil || recovered == nil || recovered.ID != job.ID {
		t.Fatalf("recuperação: %+v %v", recovered, err)
	}
	if err := st.CommitResearchRound(ctx, job, round); err == nil {
		t.Fatal("worker antigo publicou após recuperação")
	}
	job = recovered
	for i := 0; i < 2; i++ {
		if err := st.CommitResearchRound(ctx, job, round); err != nil {
			t.Fatal(err)
		}
	}
	if err := st.CompleteJob(ctx, job.ID); err != nil {
		t.Fatal(err)
	}
	discovery, err := st.ClaimJob(ctx, time.Minute)
	if err != nil || discovery == nil || discovery.Kind != "discover_web" {
		t.Fatalf("busca derivada: %+v %v", discovery, err)
	}
	if err := st.CompleteJob(ctx, discovery.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.FinishIfIdle(ctx, camp.ID); err != nil {
		t.Fatal(err)
	}
	next, err := st.ClaimJob(ctx, time.Minute)
	if err != nil || next == nil || next.Kind != "investigate" {
		t.Fatalf("segunda avaliação: %+v %v", next, err)
	}
	round.Number = 2
	if err := st.CommitResearchRound(ctx, next, round); err != nil {
		t.Fatal(err)
	}
	if err := st.CompleteJob(ctx, next.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.FinishIfIdle(ctx, camp.ID); err != nil {
		t.Fatal(err)
	}
	rounds, err := st.ListResearchRounds(ctx, camp.ID)
	if err != nil || len(rounds) != 2 {
		t.Fatalf("histórico: %+v %v", rounds, err)
	}
	if rounds[0].EnqueuedQueries != 1 || rounds[1].Reason != "no_new_queries" {
		t.Fatalf("decisões incorretas: %+v", rounds)
	}
	final, err := st.GetCampaign(ctx, camp.ID)
	if err != nil || final.Status != "completed" {
		t.Fatalf("campanha final: %+v %v", final, err)
	}
}

func TestAutomaticResearchFailureIntegration(t *testing.T) {
	dsn := os.Getenv("NIE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("NIE_TEST_DATABASE_URL não definido")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	st, err := New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	st.ConfigureAutomaticResearch(3)
	camp, err := st.CreateCampaign(ctx, "failed evaluation", nil, domain.Budget{MaxURLs: 10, MaxDomains: 2, MaxBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	defer st.pool.Exec(context.Background(), `DELETE FROM campaigns WHERE id=$1::uuid`, camp.ID)
	if err := st.FinishIfIdle(ctx, camp.ID); err != nil {
		t.Fatal(err)
	}
	job, err := st.ClaimJob(ctx, time.Minute)
	if err != nil || job == nil {
		t.Fatalf("job=%+v err=%v", job, err)
	}
	if err := st.DeadJob(ctx, job.ID, fmt.Errorf("parser unavailable")); err != nil {
		t.Fatal(err)
	}
	if _, err := st.FinishIdleCampaigns(ctx); err != nil {
		t.Fatal(err)
	}
	rounds, err := st.ListResearchRounds(ctx, camp.ID)
	if err != nil || len(rounds) != 1 || rounds[0].Reason != "evaluation_failed" {
		t.Fatalf("falha sem motivo auditável: %+v %v", rounds, err)
	}
	final, err := st.GetCampaign(ctx, camp.ID)
	if err != nil || final.Status != "completed" {
		t.Fatalf("campanha final: %+v %v", final, err)
	}
}

func TestAutomaticResearchLimitsIntegration(t *testing.T) {
	dsn := os.Getenv("NIE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("NIE_TEST_DATABASE_URL não definido")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	st, err := New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	for _, reason := range []string{"round_limit", "budget_exhausted"} {
		t.Run(reason, func(t *testing.T) {
			st.ConfigureAutomaticResearch(1)
			maxURLs := 10
			var seeds []string
			if reason == "budget_exhausted" {
				maxURLs = 1
				seeds = []string{"https://example.org/limit"}
			}
			camp, err := st.CreateCampaign(ctx, fmt.Sprintf("limit %s", reason), seeds, domain.Budget{MaxURLs: maxURLs, MaxDomains: 2, MaxBytes: 1 << 20})
			if err != nil {
				t.Fatal(err)
			}
			defer st.pool.Exec(context.Background(), `DELETE FROM campaigns WHERE id=$1::uuid`, camp.ID)
			if _, err := st.EnqueueDiscovery(ctx, camp.ID, "discover_web", "initial"); err != nil {
				t.Fatal(err)
			}
			if _, err := st.pool.Exec(ctx, `UPDATE jobs SET status='done' WHERE campaign_id=$1::uuid`, camp.ID); err != nil {
				t.Fatal(err)
			}
			if err := st.FinishIfIdle(ctx, camp.ID); err != nil {
				t.Fatal(err)
			}
			job, err := st.ClaimJob(ctx, time.Minute)
			if err != nil || job == nil {
				t.Fatalf("job=%+v err=%v", job, err)
			}
			if err := st.CommitResearchRound(ctx, job, ResearchRound{Number: 1, Decision: "continue", Leads: []ResearchQuery{{Kind: "discover_web", Query: "new"}}}); err != nil {
				t.Fatal(err)
			}
			rounds, err := st.ListResearchRounds(ctx, camp.ID)
			if err != nil || len(rounds) != 1 || rounds[0].Reason != reason || rounds[0].EnqueuedQueries != 0 {
				t.Fatalf("limite ignorado: %+v %v", rounds, err)
			}
		})
	}
}
