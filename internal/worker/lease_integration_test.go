package worker

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"nie/internal/discovery"
	"nie/internal/domain"
	"nie/internal/store"
	"os"
	"testing"
	"time"
)

func TestWorkerCancelsAfterLeaseReplacementIntegration(t *testing.T) {
	dsn := os.Getenv("NIE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("NIE_TEST_DATABASE_URL não definido")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	st, err := store.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	cancelled := make(chan struct{})
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(started); <-r.Context().Done(); close(cancelled) }))
	defer source.Close()
	camp, err := st.CreateCampaign(ctx, "lease cancel", nil, domain.Budget{MaxURLs: 5, MaxBytes: 1 << 20, MaxDomains: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Pool().Exec(context.Background(), `DELETE FROM campaigns WHERE id=$1::uuid`, camp.ID)
	if _, err := st.EnqueueDiscovery(ctx, camp.ID, "discover_web", "controlled request"); err != nil {
		t.Fatal(err)
	}
	job, err := st.ClaimJob(ctx, time.Minute)
	if err != nil || job == nil {
		t.Fatalf("job=%+v %v", job, err)
	}
	worker := &Worker{store: st, searx: discovery.NewSearXNG(source.URL), logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	completed := make(chan error, 1)
	go func() { completed <- worker.processWithLeaseTiming(ctx, job, 100*time.Millisecond, time.Minute) }()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal("request did not start")
	}
	if err := st.DeferClaimedJob(ctx, job, time.Now().Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	replacement, err := st.ClaimJob(ctx, time.Minute)
	if err != nil || replacement == nil {
		t.Fatalf("replacement=%+v %v", replacement, err)
	}
	select {
	case err := <-completed:
		if !errors.Is(err, store.ErrLeaseLost) {
			t.Fatalf("lost lease not reported: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("worker failed to cancel")
	}
	select {
	case <-cancelled:
	case <-ctx.Done():
		t.Fatal("HTTP request not cancelled")
	}
	if err := st.CompleteClaimedJob(ctx, replacement); err != nil {
		t.Fatal(err)
	}
}
