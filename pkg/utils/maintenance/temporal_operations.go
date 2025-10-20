package maintenance

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	jsonrepair "github.com/kaptinlin/jsonrepair"
	"github.com/soundprediction/go-graphiti/pkg/llm"
	"github.com/soundprediction/go-graphiti/pkg/prompts"
	"github.com/soundprediction/go-graphiti/pkg/types"
)

// TemporalOperations provides temporal analysis and edge dating operations
type TemporalOperations struct {
	llm     llm.Client
	prompts prompts.Library
}

// NewTemporalOperations creates a new TemporalOperations instance
func NewTemporalOperations(llm llm.Client, prompts prompts.Library) *TemporalOperations {
	return &TemporalOperations{
		llm:     llm,
		prompts: prompts,
	}
}

// ExtractEdgeDates extracts temporal information for an edge from episode context
func (to *TemporalOperations) ExtractEdgeDates(ctx context.Context, edge *types.Edge, currentEpisode *types.Node, previousEpisodes []*types.Node) (*time.Time, *time.Time, error) {
	start := time.Now()

	// Prepare previous episodes content
	previousEpisodeContents := make([]string, len(previousEpisodes))
	for i, ep := range previousEpisodes {
		previousEpisodeContents[i] = ep.Summary
	}

	// Prepare context for LLM
	promptContext := map[string]interface{}{
		"edge_fact":           edge.Summary,
		"current_episode":     currentEpisode.Summary,
		"previous_episodes":   previousEpisodeContents,
		"reference_timestamp": currentEpisode.ValidFrom.Format(time.RFC3339),
		"ensure_ascii":        true,
	}

	// Extract dates using LLM
	messages, err := to.prompts.ExtractEdgeDates().ExtractDates().Call(promptContext)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create edge dates prompt: %w", err)
	}

	response, err := to.llm.ChatWithStructuredOutput(ctx, messages, &prompts.EdgeDates{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract edge dates: %w", err)
	}

	// Repair JSON before unmarshaling
	repairedResponse, _ := jsonrepair.JSONRepair(string(response))

	// Try to unmarshal - if it's a quoted JSON string, unmarshal twice
	var rawJSON json.RawMessage
	if err := json.Unmarshal([]byte(repairedResponse), &rawJSON); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal repaired response: %w", err)
	}

	var edgeDates prompts.EdgeDates
	if err := json.Unmarshal(rawJSON, &edgeDates); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal edge dates response: %w", err)
	}

	var validAt *time.Time
	var invalidAt *time.Time

	// Parse valid_at date
	if edgeDates.ValidAt != nil && *edgeDates.ValidAt != "" {
		// Strip any surrounding quotes (can happen with double JSON encoding)
		cleanValidAt := strings.Trim(*edgeDates.ValidAt, "\"")
		parsed, err := time.Parse(time.RFC3339, strings.ReplaceAll(cleanValidAt, "Z", "+00:00"))
		if err != nil {
			log.Printf("Warning: failed to parse valid_at date '%s': %v", cleanValidAt, err)
		} else {
			utcTime := parsed.UTC()
			validAt = &utcTime
		}
	}

	// Parse invalid_at date
	if edgeDates.InvalidAt != nil && *edgeDates.InvalidAt != "" {
		// Strip any surrounding quotes (can happen with double JSON encoding)
		cleanInvalidAt := strings.Trim(*edgeDates.InvalidAt, "\"")
		parsed, err := time.Parse(time.RFC3339, strings.ReplaceAll(cleanInvalidAt, "Z", "+00:00"))
		if err != nil {
			log.Printf("Warning: failed to parse invalid_at date '%s': %v", cleanInvalidAt, err)
		} else {
			utcTime := parsed.UTC()
			invalidAt = &utcTime
		}
	}

	log.Printf("Extracted edge dates in %v", time.Since(start))
	return validAt, invalidAt, nil
}

// GetEdgeContradictions identifies edges that contradict a new edge
func (to *TemporalOperations) GetEdgeContradictions(ctx context.Context, newEdge *types.Edge, existingEdges []*types.Edge) ([]*types.Edge, error) {
	if len(existingEdges) == 0 {
		return []*types.Edge{}, nil
	}

	start := time.Now()

	// Prepare context for LLM
	newEdgeContext := map[string]interface{}{
		"fact": newEdge.Summary,
	}

	existingEdgeContext := make([]map[string]interface{}, len(existingEdges))
	for i, edge := range existingEdges {
		existingEdgeContext[i] = map[string]interface{}{
			"id":   i,
			"fact": edge.Summary,
		}
	}

	promptContext := map[string]interface{}{
		"new_edge":       newEdgeContext,
		"existing_edges": existingEdgeContext,
		"ensure_ascii":   true,
	}

	// Use LLM to identify contradictions
	messages, err := to.prompts.InvalidateEdges().Invalidate().Call(promptContext)
	if err != nil {
		return nil, fmt.Errorf("failed to create invalidation prompt: %w", err)
	}

	response, err := to.llm.ChatWithStructuredOutput(ctx, messages, &prompts.InvalidatedEdges{})
	if err != nil {
		return nil, fmt.Errorf("failed to identify contradictions: %w", err)
	}

	// Repair JSON before unmarshaling
	repairedResponse, _ := jsonrepair.JSONRepair(string(response))

	// Try to unmarshal - if it's a quoted JSON string, unmarshal twice
	var rawJSON json.RawMessage
	if err := json.Unmarshal([]byte(repairedResponse), &rawJSON); err != nil {
		return nil, fmt.Errorf("failed to unmarshal repaired response: %w", err)
	}

	var invalidatedEdges prompts.InvalidatedEdges
	if err := json.Unmarshal(rawJSON, &invalidatedEdges); err != nil {
		return nil, fmt.Errorf("failed to unmarshal invalidation response: %w", err)
	}

	// Extract contradicted edges
	var contradictedEdges []*types.Edge
	for _, factID := range invalidatedEdges.ContradictedFacts {
		if factID >= 0 && factID < len(existingEdges) {
			contradictedEdges = append(contradictedEdges, existingEdges[factID])
		}
	}

	log.Printf("Found %d contradicted edges in %v", len(contradictedEdges), time.Since(start))
	return contradictedEdges, nil
}

// ExtractAndSaveEdgeDates extracts temporal information for edges and updates them
func (to *TemporalOperations) ExtractAndSaveEdgeDates(ctx context.Context, edges []*types.Edge, currentEpisode *types.Node, previousEpisodes []*types.Node) ([]*types.Edge, error) {
	if len(edges) == 0 {
		return []*types.Edge{}, nil
	}

	log.Printf("Extracting dates for %d edges", len(edges))

	var updatedEdges []*types.Edge

	for _, edge := range edges {
		// Extract dates for this edge
		validAt, invalidAt, err := to.ExtractEdgeDates(ctx, edge, currentEpisode, previousEpisodes)
		if err != nil {
			log.Printf("Warning: failed to extract dates for edge %s: %v", edge.ID, err)
			updatedEdges = append(updatedEdges, edge) // Use original edge if extraction fails
			continue
		}

		// Create updated edge with new temporal information
		updatedEdge := *edge // Copy the edge
		if validAt != nil {
			updatedEdge.ValidFrom = *validAt
		}
		if invalidAt != nil {
			updatedEdge.ValidTo = invalidAt
		}
		updatedEdge.UpdatedAt = time.Now().UTC()

		updatedEdges = append(updatedEdges, &updatedEdge)
	}

	log.Printf("Updated temporal information for %d edges", len(updatedEdges))
	return updatedEdges, nil
}

// ValidateEdgeTemporalConsistency checks if edge temporal information is consistent
func (to *TemporalOperations) ValidateEdgeTemporalConsistency(edge *types.Edge) error {
	// Check if ValidTo is after ValidFrom
	if edge.ValidTo != nil && edge.ValidTo.Before(edge.ValidFrom) {
		return fmt.Errorf("edge %s has invalid temporal range: ValidTo (%v) is before ValidFrom (%v)",
			edge.ID, edge.ValidTo, edge.ValidFrom)
	}

	// Check if edge is already expired at creation time
	now := time.Now().UTC()
	if edge.ValidTo != nil && edge.ValidTo.Before(edge.CreatedAt) {
		log.Printf("Warning: edge %s was created already expired (ValidTo: %v, CreatedAt: %v)",
			edge.ID, edge.ValidTo, edge.CreatedAt)
	}

	// Check if ValidFrom is in the future relative to creation
	if edge.ValidFrom.After(now.Add(24 * time.Hour)) {
		log.Printf("Warning: edge %s has ValidFrom significantly in the future (%v)",
			edge.ID, edge.ValidFrom)
	}

	return nil
}

// ApplyTemporalInvalidation applies temporal invalidation logic to a set of edges
func (to *TemporalOperations) ApplyTemporalInvalidation(newEdge *types.Edge, candidateEdges []*types.Edge) []*types.Edge {
	if len(candidateEdges) == 0 {
		return []*types.Edge{}
	}

	now := time.Now().UTC()
	var invalidatedEdges []*types.Edge

	for _, candidateEdge := range candidateEdges {
		// Skip edges that are already invalid before the new edge becomes valid
		if candidateEdge.ValidTo != nil && candidateEdge.ValidTo.Before(newEdge.ValidFrom) {
			continue
		}

		// Skip if new edge is invalid before the candidate becomes valid
		if newEdge.ValidTo != nil && newEdge.ValidTo.Before(candidateEdge.ValidFrom) {
			continue
		}

		// Invalidate edge if the new edge becomes valid after this one
		if candidateEdge.ValidFrom.Before(newEdge.ValidFrom) {
			invalidatedEdge := *candidateEdge // Copy the edge
			validTo := newEdge.ValidFrom
			invalidatedEdge.ValidTo = &validTo
			invalidatedEdge.UpdatedAt = now

			invalidatedEdges = append(invalidatedEdges, &invalidatedEdge)
		}
	}

	return invalidatedEdges
}

// GetActiveEdgesAtTime returns edges that were active at a specific time
func (to *TemporalOperations) GetActiveEdgesAtTime(edges []*types.Edge, targetTime time.Time) []*types.Edge {
	var activeEdges []*types.Edge

	for _, edge := range edges {
		// Check if edge was valid at the target time
		if edge.ValidFrom.After(targetTime) {
			continue // Edge hadn't started yet
		}

		if edge.ValidTo != nil && edge.ValidTo.Before(targetTime) {
			continue // Edge had already ended
		}

		activeEdges = append(activeEdges, edge)
	}

	return activeEdges
}

// GetEdgeLifespan calculates the lifespan of an edge
func (to *TemporalOperations) GetEdgeLifespan(edge *types.Edge) *time.Duration {
	if edge.ValidTo == nil {
		return nil // Edge is still active
	}

	lifespan := edge.ValidTo.Sub(edge.ValidFrom)
	return &lifespan
}
