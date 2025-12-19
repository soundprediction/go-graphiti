package search

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/soundprediction/go-predicato/pkg/types"
)

// FormatEdgeDateRange formats the valid and invalid dates of an edge for display
func FormatEdgeDateRange(edge *types.Edge) string {
	validAtStr := "date unknown"
	if !edge.ValidFrom.IsZero() {
		validAtStr = edge.ValidFrom.Format(time.RFC3339)
	}

	invalidAtStr := "present"
	if edge.ValidTo != nil {
		invalidAtStr = edge.ValidTo.Format(time.RFC3339)
	}

	return fmt.Sprintf("%s - %s", validAtStr, invalidAtStr)
}

// SearchResultsToContextString reformats a set of search results into a single string
// to pass directly to an LLM as context
func SearchResultsToContextString(results *HybridSearchResult, ensureASCII bool) (string, error) {
	// Convert edges to fact JSON
	var factJSON []map[string]interface{}
	for _, edge := range results.Edges {
		validAtStr := ""
		if !edge.ValidFrom.IsZero() {
			validAtStr = edge.ValidFrom.Format(time.RFC3339)
		}

		invalidAtStr := "Present"
		if edge.ValidTo != nil {
			invalidAtStr = edge.ValidTo.Format(time.RFC3339)
		}

		factJSON = append(factJSON, map[string]interface{}{
			"fact":       edge.Summary, // Use Summary as the fact description
			"valid_at":   validAtStr,
			"invalid_at": invalidAtStr,
		})
	}

	// Convert nodes to entity JSON
	var entityJSON []map[string]interface{}
	for _, node := range results.Nodes {
		if node.Type == types.EntityNodeType {
			entityJSON = append(entityJSON, map[string]interface{}{
				"entity_name": node.Name,
				"summary":     node.Summary,
			})
		}
	}

	// Convert episodic nodes to episode JSON
	var episodeJSON []map[string]interface{}
	for _, node := range results.Nodes {
		if node.Type == types.EpisodicNodeType {
			// Get source description from metadata if available
			sourceDesc := ""
			if node.Metadata != nil {
				if desc, ok := node.Metadata["source_description"].(string); ok {
					sourceDesc = desc
				}
			}
			episodeJSON = append(episodeJSON, map[string]interface{}{
				"source_description": sourceDesc,
				"content":            node.Content,
			})
		}
	}

	// Convert community nodes to community JSON
	var communityJSON []map[string]interface{}
	for _, node := range results.Nodes {
		if node.Type == types.CommunityNodeType {
			communityJSON = append(communityJSON, map[string]interface{}{
				"community_name": node.Name,
				"summary":        node.Summary,
			})
		}
	}

	// Convert to JSON strings with proper indentation
	factJSONStr, err := toPromptJSON(factJSON, ensureASCII, 12)
	if err != nil {
		return "", fmt.Errorf("failed to marshal fact JSON: %w", err)
	}

	entityJSONStr, err := toPromptJSON(entityJSON, ensureASCII, 12)
	if err != nil {
		return "", fmt.Errorf("failed to marshal entity JSON: %w", err)
	}

	episodeJSONStr, err := toPromptJSON(episodeJSON, ensureASCII, 12)
	if err != nil {
		return "", fmt.Errorf("failed to marshal episode JSON: %w", err)
	}

	communityJSONStr, err := toPromptJSON(communityJSON, ensureASCII, 12)
	if err != nil {
		return "", fmt.Errorf("failed to marshal community JSON: %w", err)
	}

	contextString := fmt.Sprintf(`
    FACTS and ENTITIES represent relevant context to the current conversation.
    COMMUNITIES represent a cluster of closely related entities.

    These are the most relevant facts and their valid and invalid dates. Facts are considered valid
    between their valid_at and invalid_at dates. Facts with an invalid_at date of "Present" are considered valid.
    <FACTS>
%s
    </FACTS>
    <ENTITIES>
%s
    </ENTITIES>
    <EPISODES>
%s
    </EPISODES>
    <COMMUNITIES>
%s
    </COMMUNITIES>`, factJSONStr, entityJSONStr, episodeJSONStr, communityJSONStr)

	return contextString, nil
}

// toPromptJSON converts data to JSON with proper indentation for LLM prompts
func toPromptJSON(data interface{}, ensureASCII bool, indent int) (string, error) {
	var jsonBytes []byte
	var err error

	if ensureASCII {
		jsonBytes, err = json.MarshalIndent(data, "", "    ")
		if err != nil {
			return "", err
		}
		// Convert to ASCII by escaping non-ASCII characters
		jsonStr := string(jsonBytes)
		var result strings.Builder
		for _, r := range jsonStr {
			if r > 127 {
				result.WriteString(fmt.Sprintf("\\u%04x", r))
			} else {
				result.WriteRune(r)
			}
		}
		jsonBytes = []byte(result.String())
	} else {
		jsonBytes, err = json.MarshalIndent(data, "", "    ")
		if err != nil {
			return "", err
		}
	}

	// Add proper indentation
	lines := strings.Split(string(jsonBytes), "\n")
	indentStr := strings.Repeat(" ", indent)

	var indentedLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			indentedLines = append(indentedLines, indentStr+line)
		} else {
			indentedLines = append(indentedLines, line)
		}
	}

	return strings.Join(indentedLines, "\n"), nil
}

// GetDefaultSearchConfig returns a sensible default search configuration
func GetDefaultSearchConfig() *SearchConfig {
	return CombinedHybridSearchRRF
}

// GetSearchConfigByName returns a predefined search configuration by name
func GetSearchConfigByName(name string) *SearchConfig {
	configs := map[string]*SearchConfig{
		"combined_hybrid_rrf":            CombinedHybridSearchRRF,
		"combined_hybrid_mmr":            CombinedHybridSearchMMR,
		"combined_hybrid_cross_encoder":  CombinedHybridSearchCrossEncoder,
		"edge_hybrid_rrf":                EdgeHybridSearchRRF,
		"edge_hybrid_mmr":                EdgeHybridSearchMMR,
		"edge_hybrid_node_distance":      EdgeHybridSearchNodeDistance,
		"edge_hybrid_episode_mentions":   EdgeHybridSearchEpisodeMentions,
		"edge_hybrid_cross_encoder":      EdgeHybridSearchCrossEncoder,
		"node_hybrid_rrf":                NodeHybridSearchRRF,
		"node_hybrid_mmr":                NodeHybridSearchMMR,
		"node_hybrid_node_distance":      NodeHybridSearchNodeDistance,
		"node_hybrid_episode_mentions":   NodeHybridSearchEpisodeMentions,
		"node_hybrid_cross_encoder":      NodeHybridSearchCrossEncoder,
		"community_hybrid_rrf":           CommunityHybridSearchRRF,
		"community_hybrid_mmr":           CommunityHybridSearchMMR,
		"community_hybrid_cross_encoder": CommunityHybridSearchCrossEncoder,
	}

	return configs[name]
}

// ListAvailableSearchConfigs returns a list of all available predefined search configuration names
func ListAvailableSearchConfigs() []string {
	return []string{
		"combined_hybrid_rrf",
		"combined_hybrid_mmr",
		"combined_hybrid_cross_encoder",
		"edge_hybrid_rrf",
		"edge_hybrid_mmr",
		"edge_hybrid_node_distance",
		"edge_hybrid_episode_mentions",
		"edge_hybrid_cross_encoder",
		"node_hybrid_rrf",
		"node_hybrid_mmr",
		"node_hybrid_node_distance",
		"node_hybrid_episode_mentions",
		"node_hybrid_cross_encoder",
		"community_hybrid_rrf",
		"community_hybrid_mmr",
		"community_hybrid_cross_encoder",
	}
}
