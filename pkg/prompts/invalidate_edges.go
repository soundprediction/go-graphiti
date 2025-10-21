package prompts

import (
	"fmt"

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

	previousEpisodesJSON, err := ToPromptJSON(previousEpisodes, ensureASCII, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal previous episodes: %w", err)
	}

	existingEdgesJSON, err := ToPromptJSON(existingEdges, ensureASCII, 2)
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

Based on the current message, determine which existing edges should be invalidated.
Edges should be invalidated if:
1. The current message contradicts the relationship
2. The relationship has ended according to the message
3. New information makes the edge no longer accurate

Return a list of edge IDs that should be invalidated.
`, previousEpisodesJSON, episodeContent, existingEdgesJSON, referenceTime)
	logPrompts(context, sysPrompt, userPrompt)
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
