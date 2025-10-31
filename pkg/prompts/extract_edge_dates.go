package prompts

import (
	"fmt"
	"log/slog"

	"github.com/soundprediction/go-graphiti/pkg/llm"
	"github.com/soundprediction/go-graphiti/pkg/types"
)

// ExtractEdgeDatesPrompt defines the interface for extract edge dates prompts.
type ExtractEdgeDatesPrompt interface {
	ExtractDates() PromptVersion
}

// ExtractEdgeDatesVersions holds all versions of extract edge dates prompts.
type ExtractEdgeDatesVersions struct {
	ExtractDatesPrompt PromptVersion
}

func (e *ExtractEdgeDatesVersions) ExtractDates() PromptVersion { return e.ExtractDatesPrompt }

// extractDatesPrompt extracts dates from edges.
// Uses TSV format for episodes and edges to reduce token usage and improve LLM parsing.
func extractDatesPrompt(context map[string]interface{}) ([]types.Message, error) {
	sysPrompt := `You are an expert temporal information extractor that identifies valid_at and invalid_at dates for relationships from text.`

	previousEpisodes := context["previous_episodes"]
	episodeContent := context["episode_content"]
	edges := context["edges"]
	referenceTime := context["reference_time"]

	ensureASCII := false
	if val, ok := context["ensure_ascii"]; ok {
		if b, ok := val.(bool); ok {
			ensureASCII = b
		}
	}

	previousEpisodesTSV, err := ToPromptCSV(previousEpisodes, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal previous episodes: %w", err)
	}

	edgesTSV, err := ToPromptCSV(edges, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal edges: %w", err)
	}

	userPrompt := fmt.Sprintf(`
<PREVIOUS MESSAGES>
%s
</PREVIOUS MESSAGES>
<CURRENT MESSAGE>
%v
</CURRENT MESSAGE>
<EDGES>
%s
</EDGES>
<REFERENCE TIME>
%v
</REFERENCE TIME>

Note: PREVIOUS MESSAGES and EDGES are provided in TSV (tab-separated values) format.

Extract temporal information (valid_at and invalid_at dates) for the given edges based on the messages.

Guidelines:
1. Use ISO 8601 format with Z suffix (UTC)
2. If the relationship is ongoing, set valid_at to reference time
3. If the relationship has ended, set invalid_at to the end time
4. Leave empty string if no temporal information is available
5. Use reference time to resolve relative temporal expressions

Return the results in TSV (tab-separated values) format with the following structure:

valid_at	invalid_at
2024-01-01T00:00:00Z
	2024-12-31T23:59:59Z

Output ONLY the TSV data with a header row. Include exactly one data row with the dates. Use empty strings (not null) for missing dates.
`, previousEpisodesTSV, episodeContent, edgesTSV, referenceTime)
	logPrompts(context["logger"].(*slog.Logger), sysPrompt, userPrompt)
	return []types.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// NewExtractEdgeDatesVersions creates a new ExtractEdgeDatesVersions instance.
func NewExtractEdgeDatesVersions() *ExtractEdgeDatesVersions {
	return &ExtractEdgeDatesVersions{
		ExtractDatesPrompt: NewPromptVersion(extractDatesPrompt),
	}
}
