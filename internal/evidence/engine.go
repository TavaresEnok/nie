package evidence

import (
	"fmt"
	"strings"

	"nie/internal/classifier"
	"nie/internal/domain"
)

type Engine struct {
	tracker *ClusterTracker
}

func NewEngine() *Engine {
	return &Engine{
		tracker: NewClusterTracker(),
	}
}

// EvaluateEvidence evaluates a claim given a set of passages and official status.
func (e *Engine) ResolveClaimStatus(claim *domain.Claim, evidences []domain.Evidence) {
	if len(evidences) == 0 {
		claim.Status = "unconfirmed"
		claim.Applicability = "indeterminate"
		return
	}

	hasOfficial := false
	uniqueClusters := make(map[string]bool)
	hasContradiction := false

	for _, ev := range evidences {
		if ev.Stance == "refutes" {
			hasContradiction = true
		}
		if ev.IsOfficialDoc {
			hasOfficial = true
		}
		if ev.ClusterID != nil && *ev.ClusterID != "" {
			uniqueClusters[*ev.ClusterID] = true
		} else {
			// Fallback unique pseudo-cluster per evidence ID
			uniqueClusters[ev.ID] = true
		}
	}

	if hasContradiction {
		claim.Status = "contested"
		claim.Applicability = "incompatible"
		return
	}

	// Epistemological status logic:
	// 1. Official vendor doc -> "sustained"
	// 2. >= 2 independent origin clusters -> "corroborated"
	// 3. Otherwise -> "unconfirmed"
	if hasOfficial {
		claim.Status = "sustained"
		claim.Applicability = "confirmed"
	} else if len(uniqueClusters) >= 2 {
		claim.Status = "corroborated"
		claim.Applicability = "confirmed"
	} else {
		claim.Status = "unconfirmed"
		claim.Applicability = "indeterminate"
	}
}

// BuildClaimsFromPassages analyzes a campaign's passages and synthesizes claims with extracted CLI commands and vendor contexts.
func (e *Engine) BuildClaimsFromPassages(campaignID string, query string, passages []domain.SearchResult) []domain.Claim {
	var claims []domain.Claim
	claimsByTopic := make(map[string]*domain.Claim)

	for _, p := range passages {
		classification := classifier.Classify(p.Snippet, p.Domain)
		topic := query
		vendorName := ""
		if len(classification.Vendors) > 0 {
			vendorName = classification.Vendors[0].Name
			topic = fmt.Sprintf("%s - %s", classification.Vendors[0].Name, query)
		}

		claim, exists := claimsByTopic[topic]
		if !exists {
			claim = &domain.Claim{
				ID:                    fmt.Sprintf("claim-%d", len(claimsByTopic)+1),
				CampaignID:            campaignID,
				Title:                 topic,
				Statement:             fmt.Sprintf("Configuração e comandos verificados para %s", topic),
				VendorName:            vendorName,
				Status:                "unconfirmed",
				Applicability:         "indeterminate",
				OperationalValidation: "untested",
				Evidences:             []domain.Evidence{},
			}
			claimsByTopic[topic] = claim
		}

		fp := Fingerprint(p.Snippet)
		e.tracker.Add(fp, p.Domain)
		clusterID := fp

		stance := "supports"
		if strings.Contains(strings.ToLower(p.Snippet), "deprecated") ||
			strings.Contains(strings.ToLower(p.Snippet), "obsolete") ||
			strings.Contains(strings.ToLower(p.Snippet), "not supported") ||
			strings.Contains(strings.ToLower(p.Snippet), "syntax error") {
			stance = "refutes"
		}

		ev := domain.Evidence{
			ID:            fmt.Sprintf("ev-%d", len(claim.Evidences)+1),
			ClaimID:       claim.ID,
			DocumentID:    p.DocumentID,
			PassageID:     &p.PassageID,
			ClusterID:     &clusterID,
			Stance:        stance,
			Excerpt:       p.Snippet,
			Page:          p.Page,
			Confidence:    0.85,
			IsOfficialDoc: classification.IsOfficialDoc,
			Notes:         fmt.Sprintf("Fonte: %s (SHA-256: %s)", p.Domain, p.SHA256),
		}
		claim.Evidences = append(claim.Evidences, ev)
	}

	for _, claim := range claimsByTopic {
		e.ResolveClaimStatus(claim, claim.Evidences)
		claims = append(claims, *claim)
	}
	return claims
}
