package store

import (
	"context"
	"encoding/json"
	"fmt"
	"nie/internal/domain"
	"os"
	"testing"
	"time"
)

func TestClaimRevisionsAndContradictionsIntegration(t *testing.T) {
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
	camp, err := st.CreateCampaign(ctx, "historical conflicts", nil, domain.Budget{MaxURLs: 10, MaxBytes: 1 << 20, MaxDomains: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer st.pool.Exec(context.Background(), `DELETE FROM campaigns WHERE id=$1::uuid`, camp.ID)
	doc, _, err := st.RecordDocument(ctx, DocumentRecord{SHA256: hashFor(fmt.Sprint(time.Now().UnixNano())), MIME: "text/plain", Size: 20, Text: "historical support", CampaignID: camp.ID, URL: "https://example.org/conflict", NormalizedURL: "https://example.org/conflict", Domain: "example.org", Passages: []domain.Passage{{Ordinal: 0, Text: "historical support"}}})
	if err != nil {
		t.Fatal(err)
	}
	defer st.pool.Exec(context.Background(), `DELETE FROM documents WHERE id=$1::uuid`, doc)
	claim := domain.Claim{CampaignID: camp.ID, Title: "same claim", Statement: "conflicting", Status: "contested", Applicability: "indeterminate", OperationalValidation: "untested", Evidences: []domain.Evidence{{ID: "a", DocumentID: doc, Stance: "supports", Excerpt: "historical support"}, {ID: "b", DocumentID: doc, Stance: "refutes", Excerpt: "historical refutation"}}, Contradictions: []domain.Contradiction{{EvidenceAID: "a", EvidenceBID: "b", Explanation: "two sources disagree"}}}
	if err := st.SaveClaims(ctx, []domain.Claim{claim}); err != nil {
		t.Fatal(err)
	}
	reloaded, err := st.ListClaimsByCampaign(ctx, camp.ID)
	if err != nil || len(reloaded) != 1 || len(reloaded[0].Contradictions) != 1 {
		t.Fatalf("contradição não persistiu: %+v %v", reloaded, err)
	}
	if reloaded[0].Contradictions[0].EvidenceAID == "a" || reloaded[0].Contradictions[0].EvidenceBID == "b" {
		t.Fatal("IDs temporários não foram convertidos")
	}
	claim.Statement = "new conclusion"
	claim.Status = "documented"
	claim.Evidences = claim.Evidences[:1]
	claim.Contradictions = nil
	if err := st.SaveClaims(ctx, []domain.Claim{claim}); err != nil {
		t.Fatal(err)
	}
	revisions, err := st.ListClaimRevisions(ctx, camp.ID, 100)
	if err != nil || len(revisions) < 2 {
		t.Fatalf("histórico incompleto: %+v %v", revisions, err)
	}
	found := false
	for _, rev := range revisions {
		var snapshot struct {
			Statement      string                 `json:"statement"`
			Evidences      []domain.Evidence      `json:"evidences"`
			Contradictions []domain.Contradiction `json:"contradictions"`
		}
		if err := json.Unmarshal(rev.Snapshot, &snapshot); err != nil {
			t.Fatal(err)
		}
		if snapshot.Statement == "conflicting" && len(snapshot.Evidences) == 2 && len(snapshot.Contradictions) == 1 {
			found = true
		}
	}
	if !found {
		t.Fatal("a reavaliação apagou as evidências e o conflito anteriores")
	}
}
