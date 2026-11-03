package store

import (
	"context"
	"errors"
	"nie/internal/domain"
	"os"
	"testing"
	"time"
)

func TestJobLeaseFencingIntegration(t *testing.T) {
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
	camp, err := st.CreateCampaign(ctx, "lease fencing", []string{"https://example.org/lease"}, domain.Budget{MaxURLs: 5, MaxBytes: 1 << 20, MaxDomains: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer st.pool.Exec(context.Background(), `DELETE FROM campaigns WHERE id=$1::uuid`, camp.ID)
	first, err := st.ClaimJob(ctx, time.Minute)
	if err != nil || first == nil {
		t.Fatalf("first=%+v %v", first, err)
	}
	if err := st.RenewClaimedJob(ctx, first, 2*time.Minute); err != nil {
		t.Fatal(err)
	}
	// Throttling reuses the attempt count; the generation must still change.
	if err := st.DeferClaimedJob(ctx, first, time.Now().Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	second, err := st.ClaimJob(ctx, time.Minute)
	if err != nil || second == nil || second.ID != first.ID || second.Attempts != first.Attempts || second.LeaseGeneration == first.LeaseGeneration {
		t.Fatalf("generation not renewed: %+v %+v %v", first, second, err)
	}
	staleActions := []func() error{
		func() error { return st.RenewClaimedJob(ctx, first, time.Minute) },
		func() error { return st.CompleteClaimedJob(ctx, first) },
		func() error { return st.DeadClaimedJob(ctx, first, errors.New("stale")) },
		func() error { return st.RetryClaimedJob(ctx, first, time.Second, errors.New("stale")) },
		func() error { return st.DeferClaimedJob(ctx, first, time.Now()) },
	}
	for _, action := range staleActions {
		if err := action(); !errors.Is(err, ErrLeaseLost) {
			t.Fatalf("stale owner changed job: %v", err)
		}
	}
	if _, err := st.pool.Exec(ctx, `UPDATE jobs SET lease_until=now()-interval '1 second' WHERE id=$1`, second.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.CompleteClaimedJob(ctx, second); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("expired owner completed job: %v", err)
	}
	third, err := st.ClaimJob(ctx, time.Minute)
	if err != nil || third == nil || third.LeaseGeneration <= second.LeaseGeneration {
		t.Fatalf("recovery=%+v %v", third, err)
	}
	if err := st.CompleteClaimedJob(ctx, third); err != nil {
		t.Fatal(err)
	}
}
