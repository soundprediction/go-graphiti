package prompts

import (
	"fmt"

	"github.com/soundprediction/go-graphiti/pkg/llm"
)

// DedupeNodesPrompt defines the interface for dedupe nodes prompts.
type DedupeNodesPrompt interface {
	Node() PromptVersion
	NodeList() PromptVersion
	Nodes() PromptVersion
}

// DedupeNodesVersions holds all versions of dedupe nodes prompts.
type DedupeNodesVersions struct {
	NodePrompt     PromptVersion
	NodeListPrompt PromptVersion
	NodesPrompt    PromptVersion
}

func (d *DedupeNodesVersions) Node() PromptVersion     { return d.NodePrompt }
func (d *DedupeNodesVersions) NodeList() PromptVersion { return d.NodeListPrompt }
func (d *DedupeNodesVersions) Nodes() PromptVersion    { return d.NodesPrompt }

// nodePrompt determines if a new entity is a duplicate of existing entities.
func nodePrompt(context map[string]interface{}) ([]llm.Message, error) {
	sysPrompt := `You are a helpful assistant that determines whether or not a NEW ENTITY is a duplicate of any EXISTING ENTITIES.`

	previousEpisodes := context["previous_episodes"]
	episodeContent := context["episode_content"]
	extractedNode := context["extracted_node"]
	entityTypeDescription := context["entity_type_description"]
	existingNodes := context["existing_nodes"]

	ensureASCII := false
	if val, ok := context["ensure_ascii"]; ok {
		if b, ok := val.(bool); ok {
			ensureASCII = b
		}
	}

	previousEpisodesJSON, err := ToPromptCSV(previousEpisodes, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal previous episodes: %w", err)
	}

	extractedNodeJSON, err := ToPromptCSV(extractedNode, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal extracted node: %w", err)
	}

	entityTypeDescriptionJSON, err := ToPromptCSV(entityTypeDescription, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal entity type description: %w", err)
	}

	existingNodesJSON, err := ToPromptCSV(existingNodes, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal existing nodes: %w", err)
	}

	userPrompt := fmt.Sprintf(`
<PREVIOUS MESSAGES>
%s
</PREVIOUS MESSAGES>
<CURRENT MESSAGE>
%v
</CURRENT MESSAGE>
<NEW ENTITY>
%s
</NEW ENTITY>
<ENTITY TYPE DESCRIPTION>
%s
</ENTITY TYPE DESCRIPTION>

<EXISTING ENTITIES>
%s
</EXISTING ENTITIES>

Given the above EXISTING ENTITIES and their attributes, MESSAGE, and PREVIOUS MESSAGES; Determine if the NEW ENTITY extracted from the conversation
is a duplicate entity of one of the EXISTING ENTITIES.

Entities should only be considered duplicates if they refer to the *same real-world object or concept*.
Semantic Equivalence: if a descriptive label in existing_entities clearly refers to a named entity in context, treat them as duplicates.

Do NOT mark entities as duplicates if:
- They are related but distinct.
- They have similar names or purposes but refer to separate instances or concepts.

 TASK:
 1. Compare 'new_entity' against each item in 'existing_entities'.
 2. If it refers to the same real‐world object or concept, collect its index.
 3. Let 'duplicate_idx' = the *first* collected index, or –1 if none.
 4. Let 'duplicates' = the list of *all* collected indices (empty list if none).

Also return the full name of the NEW ENTITY (whether it is the name of the NEW ENTITY, a node it
is a duplicate of, or a combination of the two).
`, previousEpisodesJSON, episodeContent, extractedNodeJSON, entityTypeDescriptionJSON, existingNodesJSON)

	return []llm.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// nodesPrompt determines whether entities extracted from a conversation are duplicates.
func nodesPrompt(context map[string]interface{}) ([]llm.Message, error) {
	sysPrompt := `You are a helpful assistant that determines whether or not ENTITIES extracted from a conversation are duplicates of existing entities.`

	previousEpisodes := context["previous_episodes"]
	episodeContent := context["episode_content"]
	extractedNodes := context["extracted_nodes"]
	existingNodes := context["existing_nodes"]

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

	extractedNodesJSON, err := ToPromptCSV(extractedNodes, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal extracted nodes: %w", err)
	}

	existingNodesJSON, err := ToPromptCSV(existingNodes, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal existing nodes: %w", err)
	}

	userPrompt := fmt.Sprintf(`
<PREVIOUS MESSAGES>
%s
</PREVIOUS MESSAGES>
<CURRENT MESSAGE>
%v
</CURRENT MESSAGE>


Each of the following ENTITIES were extracted from the CURRENT MESSAGE.
Each entity in ENTITIES is represented as a JSON object with the following structure:
{
    id: integer id of the entity,
    name: "name of the entity",
    entity_type: "ontological classification of the entity",
    entity_type_description: "Description of what the entity type represents",
    duplication_candidates: [
        {
            idx: integer index of the candidate entity,
            name: "name of the candidate entity",
            entity_type: "ontological classification of the candidate entity",
            ...<additional attributes>
        }
    ]
}

<ENTITIES>
%s
</ENTITIES>

<EXISTING ENTITIES>
%s
</EXISTING ENTITIES>

For each of the above ENTITIES, determine if the entity is a duplicate of any of the EXISTING ENTITIES.

Entities should only be considered duplicates if they refer to the *same real-world object or concept*.

Do NOT mark entities as duplicates if:
- They are related but distinct.
- They have similar names or purposes but refer to separate instances or concepts.

Task:
Your response will be json called entity_resolutions with a list that contains one entry for each entity.

For every entity, return an object with the following quantities:

	- "id": integer id from ENTITIES,
	- "name": the best full name for the entity (preserve the original name unless a duplicate has a more complete name),
	- "duplicate_idx": the idx of the EXISTING ENTITY that is the best duplicate match, or -1 if there is no duplicate,
	- "duplicates": a sorted list of all idx values from EXISTING ENTITIES that refer to duplicates (deduplicate the list, use [] when none or unsure)

- Only use idx values that appear in EXISTING ENTITIES.
- Never fabricate entities or indices.
- Output TSV; use the SCHEMA
<SCHEMA>
id: string
name: string
duplicate_idx: int
duplicates: list[int]
</SCHEMA>

- Refer to the EXAMPLE
<EXAMPLE>
id\tname\tduplicate_idx\tduplicates
0\t"anterior compartment of the lower leg"\t-1\t[]
1\t"tibialis anterior"\t-1\t[],
2\t"extensor hallucis longus"\t-1\t[],
3\t"anterior tibialis"\t1\t[1]

</EXAMPLE>

Finish your response with a new line
`, previousEpisodesJSON, episodeContent, extractedNodesJSON, existingNodesJSON)

	return []llm.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// nodeListPrompt de-duplicates nodes from node lists.
func nodeListPrompt(context map[string]interface{}) ([]llm.Message, error) {
	sysPrompt := `You are a helpful assistant that de-duplicates nodes from node lists.`

	nodes := context["nodes"]

	ensureASCII := true
	if val, ok := context["ensure_ascii"]; ok {
		if b, ok := val.(bool); ok {
			ensureASCII = b
		}
	}

	nodesJSON, err := ToPromptCSV(nodes, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal nodes: %w", err)
	}

	userPrompt := fmt.Sprintf(`
Given the following context, deduplicate a list of nodes:

Nodes:
%s

Task:
1. Group nodes together such that all duplicate nodes are in the same list of uuids
2. All duplicate uuids should be grouped together in the same list
3. Also return a new summary that synthesizes the summary into a new short summary

Guidelines:
1. Each uuid from the list of nodes should appear EXACTLY once in your response
2. If a node has no duplicates, it should appear in the response in a list of only one uuid

Respond with a JSON object in the following format:
{
    "nodes": [
        {
            "uuids": ["5d643020624c42fa9de13f97b1b3fa39", "node that is a duplicate of 5d643020624c42fa9de13f97b1b3fa39"],
            "summary": "Brief summary of the node summaries that appear in the list of names."
        }
    ]
}
`, nodesJSON)

	return []llm.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// NewDedupeNodesVersions creates a new DedupeNodesVersions instance.
func NewDedupeNodesVersions() *DedupeNodesVersions {
	return &DedupeNodesVersions{
		NodePrompt:     NewPromptVersion(nodePrompt),
		NodeListPrompt: NewPromptVersion(nodeListPrompt),
		NodesPrompt:    NewPromptVersion(nodesPrompt),
	}
}
