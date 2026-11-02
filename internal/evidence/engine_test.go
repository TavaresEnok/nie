package evidence

import (
	"testing"

	"nie/internal/domain"
)

func TestOriginClusteringDoesNotOvercountCopies(t *testing.T) {
	tracker := NewClusterTracker()

	sampleText := "Para configurar BGP no MikroTik use /routing bgp connection add name=peer1 as=65000"
	fp := Fingerprint(sampleText)

	// Blog 1 copies the text
	c1 := tracker.Add(fp, "blog1.com")
	if c1 != 1 {
		t.Errorf("esperava 1 cluster independente, obteve %d", c1)
	}

	// Blog 2 copies the exact same text
	c2 := tracker.Add(fp, "blog2.com")
	if c2 != 2 {
		t.Errorf("esperava 2 domínios no cluster, obteve %d", c2)
	}

	// Blog 1 repeats same article on another path
	c3 := tracker.Add(fp, "blog1.com")
	if c3 != 2 {
		t.Errorf("mesmo domínio não deve inflar contagem independente, obteve %d", c3)
	}
}

func TestResolveClaimStatus(t *testing.T) {
	eng := NewEngine()

	// Case 1: Unconfirmed (only 1 non-official evidence)
	c1 := &domain.Claim{Title: "BGP Test"}
	eng.ResolveClaimStatus(c1, []domain.Evidence{
		{ID: "e1", Stance: "supports", IsOfficialDoc: false},
	})
	if c1.Status != "unconfirmed" {
		t.Errorf("esperava status unconfirmed, obteve %s", c1.Status)
	}

	// Case 2: Corroborated (>= 2 distinct clusters)
	c2 := &domain.Claim{Title: "BGP Test 2"}
	cluster1 := "cluster-a"
	cluster2 := "cluster-b"
	eng.ResolveClaimStatus(c2, []domain.Evidence{
		{ID: "e1", Stance: "supports", ClusterID: &cluster1, IsOfficialDoc: false},
		{ID: "e2", Stance: "supports", ClusterID: &cluster2, IsOfficialDoc: false},
	})
	if c2.Status != "corroborated" {
		t.Errorf("esperava status corroborated, obteve %s", c2.Status)
	}

	// Case 3: Sustained (has official vendor doc)
	c3 := &domain.Claim{Title: "BGP Test 3"}
	eng.ResolveClaimStatus(c3, []domain.Evidence{
		{ID: "e1", Stance: "supports", IsOfficialDoc: true},
	})
	if c3.Status != "sustained" {
		t.Errorf("esperava status sustained, obteve %s", c3.Status)
	}

	// Case 4: Contested (has refutation)
	c4 := &domain.Claim{Title: "BGP Test 4"}
	eng.ResolveClaimStatus(c4, []domain.Evidence{
		{ID: "e1", Stance: "supports", IsOfficialDoc: true},
		{ID: "e2", Stance: "refutes", IsOfficialDoc: false},
	})
	if c4.Status != "contested" {
		t.Errorf("esperava status contested, obteve %s", c4.Status)
	}
}
