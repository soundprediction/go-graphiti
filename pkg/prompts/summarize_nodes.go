package prompts

import (
	"fmt"

	"github.com/soundprediction/go-graphiti/pkg/llm"
)

// SummarizeNodesPrompt defines the interface for summarize nodes prompts.
type SummarizeNodesPrompt interface {
	SummarizePair() PromptVersion
	SummarizeContext() PromptVersion
	SummaryDescription() PromptVersion
}

// SummarizeNodesVersions holds all versions of summarize nodes prompts.
type SummarizeNodesVersions struct {
	summarizePairPrompt      PromptVersion
	summarizeContextPrompt   PromptVersion
	summaryDescriptionPrompt PromptVersion
}

func (s *SummarizeNodesVersions) SummarizePair() PromptVersion    { return s.summarizePairPrompt }
func (s *SummarizeNodesVersions) SummarizeContext() PromptVersion { return s.summarizeContextPrompt }
func (s *SummarizeNodesVersions) SummaryDescription() PromptVersion {
	return s.summaryDescriptionPrompt
}

// summarizePairPrompt combines summaries.
// Uses TSV format for node summaries to reduce token usage.
func summarizePairPrompt(context map[string]interface{}) ([]llm.Message, error) {
	sysPrompt := `You are a helpful assistant that combines summaries.`

	nodeSummaries := context["node_summaries"]
	ensureASCII := true
	if val, ok := context["ensure_ascii"]; ok {
		if b, ok := val.(bool); ok {
			ensureASCII = b
		}
	}

	nodeSummariesTSV, err := ToPromptCSV(nodeSummaries, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal node summaries: %w", err)
	}

	userPrompt := fmt.Sprintf(`
Synthesize the information from the following two summaries into a single succinct summary.

Summaries must be under 250 words.

Summaries are provided in TSV (tab-separated values) format:
%s
`, nodeSummariesTSV)
	logPrompts(context, sysPrompt, userPrompt)
	return []llm.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// summarizeContextPrompt extracts entity properties from provided text.
// Uses TSV format for episodes and attributes to reduce token usage.
func summarizeContextPrompt(context map[string]interface{}) ([]llm.Message, error) {
	sysPrompt := `You are a helpful assistant that extracts entity properties from the provided text.`

	previousEpisodes := context["previous_episodes"]
	episodeContent := context["episode_content"]
	nodeName := context["node_name"]
	nodeSummary := context["node_summary"]
	attributes := context["attributes"]

	ensureASCII := true
	if val, ok := context["ensure_ascii"]; ok {
		if b, ok := val.(bool); ok {
			ensureASCII = b
		}
	}

	previousEpisodesTSV, err := ToPromptCSV(previousEpisodes, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal previous episodes: %w", err)
	}

	attributesTSV, err := ToPromptCSV(attributes, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal attributes: %w", err)
	}

	userPrompt := fmt.Sprintf(`

<PREVIOUS MESSAGES>
%s
</PREVIOUS MESSAGES>
<CURRENT MESSAGE>
%v
</CURRENT MESSAGE>

Note: PREVIOUS MESSAGES and ATTRIBUTES are provided in TSV (tab-separated values) format.

Given the above MESSAGES and the following ENTITY name, create a summary for the ENTITY. Your summary must only use
information from the provided MESSAGES. Your summary should also only contain information relevant to the
provided ENTITY. Summaries must be under 250 words.

In addition, extract any values for the provided entity properties based on their descriptions.
If the value of the entity property cannot be found in the current context, set the value of the property to the Python value None.

Guidelines:
1. Do not hallucinate entity property values if they cannot be found in the current context.
2. Only use the provided messages, entity, and entity context to set attribute values.

<ENTITY>
%v
</ENTITY>

<ENTITY CONTEXT>
%v
</ENTITY CONTEXT>

<ATTRIBUTES>
%s
</ATTRIBUTES>
`, previousEpisodesTSV, episodeContent, nodeName, nodeSummary, attributesTSV)
	logPrompts(context, sysPrompt, userPrompt)
	return []llm.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// summaryDescriptionPrompt describes provided contents in a single sentence.
// Uses TSV format for summary data to reduce token usage.
func summaryDescriptionPrompt(context map[string]interface{}) ([]llm.Message, error) {
	sysPrompt := `You are a helpful assistant that describes provided contents in a single sentence.`

	summary := context["summary"]
	ensureASCII := true
	if val, ok := context["ensure_ascii"]; ok {
		if b, ok := val.(bool); ok {
			ensureASCII = b
		}
	}

	summaryTSV, err := ToPromptCSV(summary, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal summary: %w", err)
	}

	userPrompt := fmt.Sprintf(`
Create a short one sentence description of the summary that explains what kind of information is summarized.
Summaries must be under 250 words.

Summary (in TSV format):
%s
`, summaryTSV)
	logPrompts(context, sysPrompt, userPrompt)
	return []llm.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// NewSummarizeNodesVersions creates a new SummarizeNodesVersions instance.
func NewSummarizeNodesVersions() *SummarizeNodesVersions {
	return &SummarizeNodesVersions{
		summarizePairPrompt:      NewPromptVersion(summarizePairPrompt),
		summarizeContextPrompt:   NewPromptVersion(summarizeContextPrompt),
		summaryDescriptionPrompt: NewPromptVersion(summaryDescriptionPrompt),
	}
}
