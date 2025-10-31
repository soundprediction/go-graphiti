package prompts

import (
	"fmt"
	"log/slog"

	"github.com/soundprediction/go-graphiti/pkg/llm"
	"github.com/soundprediction/go-graphiti/pkg/types"
)

// ExtractEdgesPrompt defines the interface for extract edges prompts.
type ExtractEdgesPrompt interface {
	Edge() PromptVersion
	Reflexion() PromptVersion
	ExtractAttributes() PromptVersion
}

// ExtractEdgesVersions holds all versions of extract edges prompts.
type ExtractEdgesVersions struct {
	EdgePrompt              PromptVersion
	ReflexionPrompt         PromptVersion
	ExtractAttributesPrompt PromptVersion
}

func (e *ExtractEdgesVersions) Edge() PromptVersion              { return e.EdgePrompt }
func (e *ExtractEdgesVersions) Reflexion() PromptVersion         { return e.ReflexionPrompt }
func (e *ExtractEdgesVersions) ExtractAttributes() PromptVersion { return e.ExtractAttributesPrompt }

// filterEdgeTypes removes the fact_type_description field from edge types
// to reduce redundancy in prompts.
func filterEdgeTypes(edgeTypes interface{}) interface{} {
	// Handle slice of maps
	if slice, ok := edgeTypes.([]map[string]interface{}); ok {
		filtered := make([]map[string]interface{}, len(slice))
		for i, m := range slice {
			filtered[i] = make(map[string]interface{})
			for k, v := range m {
				if k != "fact_type_description" {
					filtered[i][k] = v
				}
			}
		}
		return filtered
	}
	// If not the expected type, return as-is
	return edgeTypes
}

// edgePrompt extracts fact triples from text.
// Uses TSV format for episodes and edge types to reduce token usage and improve LLM parsing.
func edgePrompt(context map[string]interface{}) ([]types.Message, error) {
	sysPrompt := `You are an expert fact extractor that extracts fact triples from text.
1. Extracted fact triples should also be extracted with relevant date information.
2. Treat the CURRENT TIME as the time the CURRENT MESSAGE was sent. All temporal information should be extracted relative to this time.`

	edgeTypes := context["edge_types"]
	previousEpisodes := context["previous_episodes"]
	episodeContent := context["episode_content"]
	nodes := context["nodes"]
	referenceTime := context["reference_time"]
	customPrompt := context["custom_prompt"]

	ensureASCII := false
	if val, ok := context["ensure_ascii"]; ok {
		if b, ok := val.(bool); ok {
			ensureASCII = b
		}
	}

	// Filter out fact_type_description to reduce redundancy
	filteredEdgeTypes := filterEdgeTypes(edgeTypes)
	edgeTypesTSV, err := ToPromptCSV(filteredEdgeTypes, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal edge types: %w", err)
	}

	previousEpisodesTSV, err := ToPromptCSV(previousEpisodes, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal previous episodes: %w", err)
	}

	nodesTSV, err := ToPromptCSV(nodes, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal nodes: %w", err)
	}

	userPrompt := fmt.Sprintf(`
<FACT TYPES>
%s
</FACT TYPES>

<PREVIOUS_MESSAGES>
%s
</PREVIOUS_MESSAGES>

<CURRENT_MESSAGE>
%v
</CURRENT_MESSAGE>

<ENTITIES>
%s
</ENTITIES>

<REFERENCE_TIME>
%v  # ISO 8601 (UTC); used to resolve relative time mentions
</REFERENCE_TIME>

Note: FACT TYPES, PREVIOUS_MESSAGES, and ENTITIES are provided in TSV (tab-separated values) format.

# TASK
Extract all factual relationships between the given ENTITIES based on the CURRENT MESSAGE.
Only extract facts that:
- involve two DISTINCT ENTITIES from the ENTITIES list,
- are clearly stated or unambiguously implied in the CURRENT MESSAGE,
    and can be represented as edges in a knowledge graph.
- Facts should include entity names rather than pronouns whenever possible.
- The FACT TYPES provide a list of the most important types of facts, make sure to extract facts of these types
- The FACT TYPES are not an exhaustive list, extract all facts from the message even if they do not fit into one
    of the FACT TYPES
- The FACT TYPES each contain their fact_type_signature which represents the source and target entity types.

You may use information from the PREVIOUS MESSAGES only to disambiguate references or support continuity.

%v

# DATETIME RULES

- Use ISO 8601 with "Z" suffix (UTC) (e.g., 2025-04-30T00:00:00Z).
- If the fact is ongoing (present tense), set 'valid_at' to REFERENCE_TIME.
- If a change/termination is expressed, set 'invalid_at' to the relevant timestamp.
- Leave both fields 'null' if no explicit or resolvable time is stated.
- If only a date is mentioned (no time), assume 00:00:00.
- If only a year is mentioned, use January 1st at 00:00:00.

# EXTRACTION RULES

1. Only emit facts where both the subject and object match IDs in ENTITIES.
2. Each fact must involve two **distinct** entities.
3. Use a SCREAMING_SNAKE_CASE string as the 'relation_type' (e.g., FOUNDED, WORKS_AT).
4. Do not emit duplicate or semantically redundant facts.
5. The 'fact_text' should quote or closely paraphrase the original source sentence(s).
6. Use 'REFERENCE_TIME' to resolve vague or relative temporal expressions (e.g., "last week").
7. Do **not** hallucinate or infer temporal bounds from unrelated events.
8. Format your response in a TSV table, with the schema:

<SCHEMA>
source_id: int 
relation_type: string 
target_id: int 
fact: string 
summary: string 
valid_at: string 
invalid_at: string 
</SCHEMA>

9. Refer to the EXAMPLE; end with a new line

<EXAMPLE>
source_id\trelation_type\ttarget_id\tfact\tsummary\tvalid_at\tinvalid_at
0\t"CAUSES"\t2\t"If that pressure is not relieved\tpermanent facial nerve palsy can ensue"\t"Acute Facial Palsy (AFP) causes facial nerve palsy"\t"2025-09-27T00:00:00Z"\tnull

</EXAMPLE>
`, edgeTypesTSV, previousEpisodesTSV, episodeContent, nodesTSV, referenceTime, customPrompt)
	logPrompts(context["logger"].(*slog.Logger), sysPrompt, userPrompt)
	return []types.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// extractEdgesReflexionPrompt determines which facts have not been extracted.
// Uses TSV format for episodes to reduce token usage and improve LLM parsing.
func extractEdgesReflexionPrompt(context map[string]interface{}) ([]types.Message, error) {
	sysPrompt := `You are an AI assistant that determines which facts have not been extracted from the given context`

	previousEpisodes := context["previous_episodes"]
	episodeContent := context["episode_content"]
	nodes := context["nodes"]
	extractedFacts := context["extracted_facts"]

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

	nodesTSV, err := ToPromptCSV(nodes, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal nodes: %w", err)
	}

	userPrompt := fmt.Sprintf(`
<PREVIOUS MESSAGES>
%s
</PREVIOUS MESSAGES>
<CURRENT MESSAGE>
%v
</CURRENT MESSAGE>

<EXTRACTED ENTITIES>
%s
</EXTRACTED ENTITIES>

<EXTRACTED FACTS>
%v
</EXTRACTED FACTS>

Note: PREVIOUS MESSAGES and EXTRACTED ENTITIES are provided in TSV (tab-separated values) format.

Given the above MESSAGES, list of EXTRACTED ENTITIES entities, and list of EXTRACTED FACTS;
determine if any facts haven't been extracted.
`, previousEpisodesTSV, episodeContent, nodesTSV, extractedFacts)
	logPrompts(context["logger"].(*slog.Logger), sysPrompt, userPrompt)
	return []types.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// extractEdgesAttributesPrompt extracts fact properties from text.
func extractEdgesAttributesPrompt(context map[string]interface{}) ([]types.Message, error) {
	sysPrompt := `You are a helpful assistant that extracts fact properties from the provided text.`

	episodeContent := context["episode_content"]
	referenceTime := context["reference_time"]
	fact := context["fact"]

	userPrompt := fmt.Sprintf(`
<MESSAGE>
%v
</MESSAGE>
<REFERENCE TIME>
%v
</REFERENCE TIME>

Given the above MESSAGE, its REFERENCE TIME, and the following FACT, update any of its attributes based on the information provided
in MESSAGE. Use the provided attribute descriptions to better understand how each attribute should be determined.

Guidelines:
1. Do not hallucinate entity property values if they cannot be found in the current context.
2. Only use the provided MESSAGES and FACT to set attribute values.

<FACT>
%v
</FACT>
`, episodeContent, referenceTime, fact)
	logPrompts(context["logger"].(*slog.Logger), sysPrompt, userPrompt)
	return []types.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// NewExtractEdgesVersions creates a new ExtractEdgesVersions instance.
func NewExtractEdgesVersions() *ExtractEdgesVersions {
	return &ExtractEdgesVersions{
		EdgePrompt:              NewPromptVersion(edgePrompt),
		ReflexionPrompt:         NewPromptVersion(extractEdgesReflexionPrompt),
		ExtractAttributesPrompt: NewPromptVersion(extractEdgesAttributesPrompt),
	}
}
