package prompts

import (
	"fmt"

	"github.com/soundprediction/go-graphiti/pkg/llm"
)

// SummarizeNodesPrompt defines the interface for summarize nodes prompts.
type SummarizeNodesPrompt interface {
	Summarize() PromptVersion
}

// SummarizeNodesVersions holds all versions of summarize nodes prompts.
type SummarizeNodesVersions struct {
	SummarizePrompt PromptVersion
}

func (s *SummarizeNodesVersions) Summarize() PromptVersion { return s.SummarizePrompt }

// summarizePrompt creates summaries for nodes.
func summarizePrompt(context map[string]interface{}) ([]llm.Message, error) {
	sysPrompt := `You are a helpful assistant that creates concise summaries of entities based on available information.`

	previousEpisodes := context["previous_episodes"]
	episodeContent := context["episode_content"]
	node := context["node"]

	ensureASCII := true
	if val, ok := context["ensure_ascii"]; ok {
		if b, ok := val.(bool); ok {
			ensureASCII = b
		}
	}

	previousEpisodesJSON, err := ToPromptJSON(previousEpisodes, ensureASCII, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal previous episodes: %w", err)
	}

	episodeContentJSON, err := ToPromptJSON(episodeContent, ensureASCII, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal episode content: %w", err)
	}

	userPrompt := fmt.Sprintf(`
<PREVIOUS MESSAGES>
%s
</PREVIOUS MESSAGES>
<CURRENT MESSAGE>
%s
</CURRENT MESSAGE>
<NODE>
%v
</NODE>

Create a concise summary of the entity based on all available information from the messages.

Guidelines:
1. Keep the summary under 250 words
2. Include the most important and relevant information
3. Do not hallucinate information not present in the messages
4. Focus on facts and attributes about the entity
5. Update any existing summary with new information
`, previousEpisodesJSON, episodeContentJSON, node)

	return []llm.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// NewSummarizeNodesVersions creates a new SummarizeNodesVersions instance.
func NewSummarizeNodesVersions() *SummarizeNodesVersions {
	return &SummarizeNodesVersions{
		SummarizePrompt: NewPromptVersion(summarizePrompt),
	}
}