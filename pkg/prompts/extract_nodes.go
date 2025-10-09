package prompts

import (
	"fmt"

	"github.com/soundprediction/go-graphiti/pkg/llm"
)

// ExtractNodesPrompt defines the interface for extract nodes prompts.
type ExtractNodesPrompt interface {
	ExtractMessage() PromptVersion
	ExtractJSON() PromptVersion
	ExtractText() PromptVersion
	Reflexion() PromptVersion
	ClassifyNodes() PromptVersion
	ExtractAttributes() PromptVersion
	ExtractSummary() PromptVersion
}

// ExtractNodesVersions holds all versions of extract nodes prompts.
type ExtractNodesVersions struct {
	extractMessagePrompt    PromptVersion
	extractJSONPrompt       PromptVersion
	extractTextPrompt       PromptVersion
	reflexionPrompt         PromptVersion
	classifyNodesPrompt     PromptVersion
	extractAttributesPrompt PromptVersion
	extractSummaryPrompt    PromptVersion
}

func (e *ExtractNodesVersions) ExtractMessage() PromptVersion    { return e.extractMessagePrompt }
func (e *ExtractNodesVersions) ExtractJSON() PromptVersion       { return e.extractJSONPrompt }
func (e *ExtractNodesVersions) ExtractText() PromptVersion       { return e.extractTextPrompt }
func (e *ExtractNodesVersions) Reflexion() PromptVersion         { return e.reflexionPrompt }
func (e *ExtractNodesVersions) ClassifyNodes() PromptVersion     { return e.classifyNodesPrompt }
func (e *ExtractNodesVersions) ExtractAttributes() PromptVersion { return e.extractAttributesPrompt }
func (e *ExtractNodesVersions) ExtractSummary() PromptVersion    { return e.extractSummaryPrompt }

// extractMessagePrompt extracts entity nodes from conversational messages.
func extractMessagePrompt(context map[string]interface{}) ([]llm.Message, error) {
	sysPrompt := `You are an AI assistant that extracts entity nodes from conversational messages. 
Your primary task is to extract and classify the speaker and other significant entities mentioned in the conversation.`

	// Get values from context
	entityTypes := context["entity_types"]
	previousEpisodes := context["previous_episodes"]
	episodeContent := context["episode_content"]
	customPrompt := context["custom_prompt"]

	ensureASCII := true
	if val, ok := context["ensure_ascii"]; ok {
		if b, ok := val.(bool); ok {
			ensureASCII = b
		}
	}

	previousEpisodesJSON, err := ToPromptCSV(previousEpisodes, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal previous episodes: %w", err)
	}

	userPrompt := fmt.Sprintf(`<ENTITY TYPES>
%v
</ENTITY TYPES>

<PREVIOUS MESSAGES>
%s
</PREVIOUS MESSAGES>

<CURRENT MESSAGE>
%v
</CURRENT MESSAGE>

Instructions:

You are given a conversation context and a CURRENT MESSAGE. Your task is to extract **entity nodes** mentioned **explicitly or implicitly** in the CURRENT MESSAGE.
Pronoun references such as he/she/they or this/that/those should be disambiguated to the names of the 
reference entities. Only extract distinct entities from the CURRENT MESSAGE. Don't extract pronouns like you, me, he/she/they, we/us as entities.

1. **Speaker Extraction**: Always extract the speaker (the part before the colon in each dialogue line) as the first entity node.
   - If the speaker is mentioned again in the message, treat both mentions as a **single entity**.

2. **Entity Identification**:
   - Extract all significant entities, concepts, or actors that are **explicitly or implicitly** mentioned in the CURRENT MESSAGE.
   - **Exclude** entities mentioned only in the PREVIOUS MESSAGES (they are for context only).

3. **Entity Classification**:
   - Use the descriptions in ENTITY TYPES to classify each extracted entity.
   - Assign the appropriate entity_type_id for each one.

4. **Exclusions**:
   - Do NOT extract entities representing relationships or actions.
   - Do NOT extract dates, times, or other temporal information—these will be handled separately.

5. **Formatting**:
   - Be **explicit and unambiguous** in naming entities (e.g., use full names when available).

%v`, entityTypes, previousEpisodesJSON, episodeContent, customPrompt)

	return []llm.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// extractJSONPrompt extracts entity nodes from JSON.
func extractJSONPrompt(context map[string]interface{}) ([]llm.Message, error) {
	sysPrompt := `You are an AI assistant that extracts entity nodes from JSON. 
Your primary task is to extract and classify relevant entities from JSON files`

	entityTypes := context["entity_types"]
	sourceDescription := context["source_description"]
	episodeContent := context["episode_content"]
	customPrompt := context["custom_prompt"]

	userPrompt := fmt.Sprintf(`
<ENTITY TYPES>
%v
</ENTITY TYPES>

<SOURCE DESCRIPTION>:
%v
</SOURCE DESCRIPTION>
<JSON>
%v
</JSON>

%v

Given the above source description and JSON, extract relevant entities from the provided JSON.
For each entity extracted, also determine its entity type based on the provided ENTITY TYPES and their descriptions.
Indicate the classified entity type by providing its entity_type_id.

Guidelines:
1. Always try to extract an entities that the JSON represents. This will often be something like a "name" or "user field
2. Do NOT extract any properties that contain dates
`, entityTypes, sourceDescription, episodeContent, customPrompt)

	return []llm.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// extractTextPrompt extracts entity nodes from text.
func extractTextPrompt(context map[string]interface{}) ([]llm.Message, error) {
	sysPrompt := `You are an AI assistant that extracts entity nodes from text. 
Your primary task is to extract and classify the speaker and other significant entities mentioned in the provided text.`

	entityTypes := context["entity_types"]
	episodeContent := context["episode_content"]
	customPrompt := context["custom_prompt"]

	userPrompt := fmt.Sprintf(`
<ENTITY TYPES>
%v
</ENTITY TYPES>

<TEXT>
%v
</TEXT>

Given the above text, extract entities from the TEXT that are explicitly or implicitly mentioned.
For each entity extracted, also determine its entity type based on the provided ENTITY TYPES and their descriptions.
Indicate the classified entity type by providing its entity_type_id.

%v


Guidelines:
1. Extract significant entities, concepts, or actors mentioned in the conversation.
2. Avoid creating nodes for relationships or actions.
3. Avoid creating nodes for temporal information like dates, times or years (these will be added to edges later).
4. Be as explicit as possible in your node names, using full names and avoiding abbreviations.
5. Format your response as a TSV, with SCHEMA

<SCHEMA>
entity: string
entity_type_id: int
</SCHEMA>

<EXAMPLE>
entity\tentity_type_id
phlebotomist\t34
cognitive behavioral therapy\t30

</EXAMPLE>

Use the EXAMPLE as a guide
Finish your response with a new line
`, entityTypes, episodeContent, customPrompt)
	return []llm.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// extractNodesReflexionPrompt determines which entities have not been extracted.
func extractNodesReflexionPrompt(context map[string]interface{}) ([]llm.Message, error) {
	sysPrompt := `You are an AI assistant that determines which entities have not been extracted from the given context`

	previousEpisodes := context["previous_episodes"]
	episodeContent := context["episode_content"]
	extractedEntities := context["extracted_entities"]

	ensureASCII := true
	if val, ok := context["ensure_ascii"]; ok {
		if b, ok := val.(bool); ok {
			ensureASCII = b
		}
	}

	previousEpisodesJSON, err := ToPromptCSV(previousEpisodes, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal previous episodes: %w", err)
	}

	userPrompt := fmt.Sprintf(`
<PREVIOUS MESSAGES>
%s
</PREVIOUS MESSAGES>
<CURRENT MESSAGE>
%v
</CURRENT MESSAGE>

<EXTRACTED ENTITIES>
%v
</EXTRACTED ENTITIES>

Given the above previous messages, current message, and list of extracted entities; determine if any entities haven't been
extracted.
`, previousEpisodesJSON, episodeContent, extractedEntities)

	return []llm.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// classifyNodesPrompt classifies entity nodes.
func classifyNodesPrompt(context map[string]interface{}) ([]llm.Message, error) {
	sysPrompt := `You are an AI assistant that classifies entity nodes given the context from which they were extracted`

	previousEpisodes := context["previous_episodes"]
	episodeContent := context["episode_content"]
	extractedEntities := context["extracted_entities"]
	entityTypes := context["entity_types"]

	ensureASCII := true
	if val, ok := context["ensure_ascii"]; ok {
		if b, ok := val.(bool); ok {
			ensureASCII = b
		}
	}

	previousEpisodesJSON, err := ToPromptCSV(previousEpisodes, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal previous episodes: %w", err)
	}

	userPrompt := fmt.Sprintf(`
<PREVIOUS MESSAGES>
%s
</PREVIOUS MESSAGES>
<CURRENT MESSAGE>
%v
</CURRENT MESSAGE>

<EXTRACTED ENTITIES>
%v
</EXTRACTED ENTITIES>

<ENTITY TYPES>
%v
</ENTITY TYPES>

Given the above conversation, extracted entities, and provided entity types and their descriptions, classify the extracted entities.

Guidelines:
1. Each entity must have exactly one type
2. Only use the provided ENTITY TYPES as types, do not use additional types to classify entities.
3. If none of the provided entity types accurately classify an extracted node, the type should be set to None
`, previousEpisodesJSON, episodeContent, extractedEntities, entityTypes)

	return []llm.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// extractNodesAttributesPrompt extracts entity properties from text.
func extractNodesAttributesPrompt(context map[string]interface{}) ([]llm.Message, error) {
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

	sysPrompt := `You are a helpful assistant that extracts entity properties from the provided text.`

	userPrompt := fmt.Sprintf(`
<MESSAGES>
%s
%s
</MESSAGES>

Given the above MESSAGES and the following ENTITY, update any of its attributes based on the information provided
in MESSAGES. Use the provided attribute descriptions to better understand how each attribute should be determined.

Guidelines:
1. Do not hallucinate entity property values if they cannot be found in the current context.
2. Only use the provided MESSAGES and ENTITY to set attribute values.

<ENTITY>
%v
</ENTITY>
`, previousEpisodesJSON, episodeContentJSON, node)

	return []llm.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// extractSummaryPrompt extracts entity summaries from text.
func extractSummaryPrompt(context map[string]interface{}) ([]llm.Message, error) {
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

	sysPrompt := `You are a helpful assistant that extracts entity summaries from the provided text.`

	userPrompt := fmt.Sprintf(`
<MESSAGES>
%s
%s
</MESSAGES>

Given the above MESSAGES and the following ENTITY, update the summary that combines relevant information about the entity
from the messages and relevant information from the existing summary.

Guidelines:
1. Do not hallucinate entity summary information if they cannot be found in the current context.
2. Only use the provided MESSAGES and ENTITY to set attribute values.
3. The summary attribute represents a summary of the ENTITY, and should be updated with new information about the Entity from the MESSAGES. 
    Summaries must be no longer than 250 words.

<ENTITY>
%v
</ENTITY>
`, previousEpisodesJSON, episodeContentJSON, node)

	return []llm.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// NewExtractNodesVersions creates a new ExtractNodesVersions instance.
func NewExtractNodesVersions() *ExtractNodesVersions {
	return &ExtractNodesVersions{
		extractMessagePrompt:    NewPromptVersion(extractMessagePrompt),
		extractJSONPrompt:       NewPromptVersion(extractJSONPrompt),
		extractTextPrompt:       NewPromptVersion(extractTextPrompt),
		reflexionPrompt:         NewPromptVersion(extractNodesReflexionPrompt),
		classifyNodesPrompt:     NewPromptVersion(classifyNodesPrompt),
		extractAttributesPrompt: NewPromptVersion(extractNodesAttributesPrompt),
		extractSummaryPrompt:    NewPromptVersion(extractSummaryPrompt),
	}
}
