package store

import (
	"context"
	"fmt"
	"strings"

	"nie/internal/domain"
	"nie/internal/urlutil"
)

func (s *Store) GetDiscoveryGraph(ctx context.Context, campaignID string) (domain.DiscoveryGraph, error) {
	edgesRows, err := s.pool.Query(ctx, `
		SELECT from_url, to_url, to_domain, kind, depth, reason
		FROM discovery_edges
		WHERE campaign_id = $1
		ORDER BY depth ASC, created_at ASC`, campaignID)
	if err != nil {
		return domain.DiscoveryGraph{}, fmt.Errorf("obter edges do grafo: %w", err)
	}
	defer edgesRows.Close()

	nodesMap := make(map[string]domain.GraphNode)
	var edges []domain.GraphEdge

	for edgesRows.Next() {
		var fromURL, toURL, toDomain, kind, reason string
		var depth int
		if err := edgesRows.Scan(&fromURL, &toURL, &toDomain, &kind, &depth, &reason); err != nil {
			return domain.DiscoveryGraph{}, err
		}

		if fromURL != "" {
			if _, exists := nodesMap[fromURL]; !exists {
				nodesMap[fromURL] = domain.GraphNode{
					ID:     fromURL,
					Label:  urlutil.Domain(fromURL),
					Domain: urlutil.Domain(fromURL),
					Depth:  depth - 1,
					Type:   "link",
					Status: "done",
				}
			}
		}

		nodeType := "link"
		if depth == 0 {
			nodeType = "seed"
		} else if strings.HasPrefix(toURL, "searxng:") || strings.HasPrefix(toURL, "github:") {
			nodeType = "search"
		}

		if _, exists := nodesMap[toURL]; !exists {
			label := urlutil.Domain(toURL)
			if label == "" {
				label = toURL
			}
			nodesMap[toURL] = domain.GraphNode{
				ID:     toURL,
				Label:  label,
				Domain: toDomain,
				Depth:  depth,
				Type:   nodeType,
				Status: "done",
			}
		}

		edges = append(edges, domain.GraphEdge{
			Source: fromURL,
			Target: toURL,
			Kind:   kind,
			Depth:  depth,
			Reason: reason,
		})
	}

	var nodes []domain.GraphNode
	for _, n := range nodesMap {
		nodes = append(nodes, n)
	}

	return domain.DiscoveryGraph{
		Nodes: nodes,
		Edges: edges,
	}, nil
}
