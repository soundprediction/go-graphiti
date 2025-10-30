package prompts

import (
	"fmt"
	"log/slog"

	"github.com/soundprediction/go-graphiti/pkg/llm"
)

// InvalidateEdgesPrompt defines the interface for invalidate edges prompts.
type InvalidateEdgesPrompt interface {
	Invalidate() PromptVersion
}

// InvalidateEdgesVersions holds all versions of invalidate edges prompts.
type InvalidateEdgesVersions struct {
	InvalidatePrompt PromptVersion
}

func (i *InvalidateEdgesVersions) Invalidate() PromptVersion { return i.InvalidatePrompt }

// invalidatePrompt determines which edges should be invalidated.
// Uses TSV format for edge data to reduce token usage and improve LLM parsing.
func invalidatePrompt(context map[string]interface{}) ([]llm.Message, error) {
	sysPrompt := `You are a helpful assistant that determines which existing edges should be invalidated based on new information.`

	previousEpisodes := context["previous_episodes"]
	episodeContent := context["episode_content"]
	existingEdges := context["existing_edges"]
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

	existingEdgesTSV, err := ToPromptCSV(existingEdges, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal existing edges: %w", err)
	}

	userPrompt := fmt.Sprintf(`
<PREVIOUS MESSAGES>
%s
</PREVIOUS MESSAGES>
<CURRENT MESSAGE>
%v
</CURRENT MESSAGE>
<EXISTING EDGES>
%s
</EXISTING EDGES>
<REFERENCE TIME>
%v
</REFERENCE TIME>

EXISTING EDGES are provided in TSV (tab-separated values) format with columns including:
- id: unique identifier for the edge
- fact: the relationship or fact represented by the edge
- Additional columns may include source, target, dates, etc.

Based on the CURRENT MESSAGE, determine which EXISTING EDGES should be invalidated.

Edges should be invalidated if:
1. The current message contradicts the relationship
2. The relationship has ended according to the message
3. New information makes the edge no longer accurate

Return a list of edge IDs from EXISTING EDGES that should be invalidated.
`, previousEpisodesTSV, episodeContent, existingEdgesTSV, referenceTime)
	logPrompts(context["logger"].(*slog.Logger), sysPrompt, userPrompt)
	return []llm.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// NewInvalidateEdgesVersions creates a new InvalidateEdgesVersions instance.
func NewInvalidateEdgesVersions() *InvalidateEdgesVersions {
	return &InvalidateEdgesVersions{
		InvalidatePrompt: NewPromptVersion(invalidatePrompt),
	}
}
