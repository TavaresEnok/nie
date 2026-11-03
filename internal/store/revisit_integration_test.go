package store

import (
	"context"
	"os"
	"testing"
	"time"

	"nie/internal/domain"
)

func TestDocumentChangeReopensEvaluation(t *testing.T) {
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
	t.Cleanup(st.Close)
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	st.ConfigureAutomaticResearch(3)
	camp, err := st.CreateCampaign(ctx, "revisit", nil, domain.Budget{MaxURLs: 5, MaxDomains: 2, MaxBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		st.Pool().Exec(context.Background(), `DELETE FROM campaigns WHERE id=$1::uuid`, camp.ID)
	})
	url := "https://example.org/manual"
	first, err := st.SaveDocument(ctx, DocumentRecord{SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", MIME: "text/plain", Size: 4, Text: "old", CampaignID: camp.ID, URL: url, NormalizedURL: url, Domain: "example.org", Passages: []domain.Passage{{Ordinal: 0, Text: "old"}}})
	if err != nil || first.Changed {
		t.Fatalf("primeira=%+v err=%v", first, err)
	}
	second, err := st.SaveDocument(ctx, DocumentRecord{SHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", MIME: "text/plain", Size: 4, Text: "new", CampaignID: camp.ID, URL: url, NormalizedURL: url, Domain: "example.org", ETag: `"v2"`, Passages: []domain.Passage{{Ordinal: 0, Text: "new"}}})
	if err != nil || !second.Changed || second.PreviousSHA != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("segunda=%+v err=%v", second, err)
	}
	changes, err := st.ListDocumentChanges(ctx, camp.ID)
	if err != nil || len(changes) != 1 || changes[0].OldSHA256 == changes[0].NewSHA256 {
		t.Fatalf("changes=%+v err=%v", changes, err)
	}
	if _, err := st.Pool().Exec(ctx, `UPDATE campaigns SET status='completed', finished_at=now() WHERE id=$1::uuid`, camp.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.ReevaluateAfterChange(ctx, camp.ID); err != nil {
		t.Fatal(err)
	}
	job, err := st.ClaimJob(ctx, time.Minute)
	if err != nil || job == nil || job.Kind != "investigate" || job.Reason != "document_changed" {
		t.Fatalf("job=%+v err=%v", job, err)
	}
}
