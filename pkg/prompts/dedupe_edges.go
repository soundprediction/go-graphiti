package prompts

import (
	"fmt"

	"github.com/soundprediction/go-graphiti/pkg/llm"
)

// DedupeEdgesPrompt defines the interface for dedupe edges prompts.
type DedupeEdgesPrompt interface {
	Edge() PromptVersion
	EdgeList() PromptVersion
	ResolveEdge() PromptVersion
}

// DedupeEdgesVersions holds all versions of dedupe edges prompts.
type DedupeEdgesVersions struct {
	EdgePrompt        PromptVersion
	EdgeListPrompt    PromptVersion
	ResolveEdgePrompt PromptVersion
}

func (d *DedupeEdgesVersions) Edge() PromptVersion        { return d.EdgePrompt }
func (d *DedupeEdgesVersions) EdgeList() PromptVersion    { return d.EdgeListPrompt }
func (d *DedupeEdgesVersions) ResolveEdge() PromptVersion { return d.ResolveEdgePrompt }

// dedupeEdgePrompt determines if edges are duplicates or contradictory.
func dedupeEdgePrompt(context map[string]interface{}) ([]llm.Message, error) {
	sysPrompt := `You are a helpful assistant that determines whether or not edges extracted from a conversation are duplicates or contradictions of existing edges.`

	previousEpisodes := context["previous_episodes"]
	episodeContent := context["episode_content"]
	newFact := context["new_fact"]
	existingFacts := context["existing_facts"]

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

	newFactJSON, err := ToPromptJSON(newFact, ensureASCII, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal new fact: %w", err)
	}

	existingFactsJSON, err := ToPromptJSON(existingFacts, ensureASCII, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal existing facts: %w", err)
	}

	userPrompt := fmt.Sprintf(`
<PREVIOUS MESSAGES>
%s
</PREVIOUS MESSAGES>
<CURRENT MESSAGE>
%v
</CURRENT MESSAGE>
<NEW FACT>
%s
</NEW FACT>
<EXISTING FACTS>
%s
</EXISTING FACTS>

Task:
You have TWO separate lists of facts. Each list uses 'idx' as its index field, starting from 0.

1. DUPLICATE DETECTION:
	- If the NEW FACT represents identical factual information as any fact in EXISTING FACTS, return those idx values in duplicate_facts.
	- Facts with similar information that contain key differences should NOT be marked as duplicates.
	- Return idx values from EXISTING FACTS.
	- If no duplicates, return an empty list for duplicate_facts.

2. FACT TYPE CLASSIFICATION:
	- Given the predefined FACT TYPES, determine if the NEW FACT should be classified as one of these types.
	- Return the fact type as fact_type or DEFAULT if NEW FACT is not one of the FACT TYPES.

3. CONTRADICTION DETECTION:
	- Based on FACT INVALIDATION CANDIDATES and NEW FACT, determine which facts the new fact contradicts.
	- Return idx values from FACT INVALIDATION CANDIDATES.
	- If no contradictions, return an empty list for contradicted_facts.

IMPORTANT:
- duplicate_facts: Use ONLY 'idx' values from EXISTING FACTS
- contradicted_facts: Use ONLY 'idx' values from FACT INVALIDATION CANDIDATES
- These are two separate lists with independent idx ranges starting from 0

Guidelines:
1. Some facts may be very similar but will have key differences, particularly around numeric values in the facts.
	Do not mark these facts as duplicates.


<SCHEMA>
duplicated_facts: []int
contradicted_facts: []int
fact_type: string
</SCHEMA>
`, previousEpisodesJSON, episodeContent, newFactJSON, existingFactsJSON)

	return []llm.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// dedupeEdgeListPrompt handles batch edge deduplication.
func dedupeEdgeListPrompt(context map[string]interface{}) ([]llm.Message, error) {
	sysPrompt := `You are a helpful assistant that de-duplicates edges from edge lists.`

	edges := context["edges"]

	ensureASCII := true
	if val, ok := context["ensure_ascii"]; ok {
		if b, ok := val.(bool); ok {
			ensureASCII = b
		}
	}

	edgesJSON, err := ToPromptJSON(edges, ensureASCII, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal edges: %w", err)
	}

	userPrompt := fmt.Sprintf(`
Given the following edges, identify unique facts and remove duplicates:

Edges:
%s

Task:
Return a list of unique facts, removing any duplicates.
`, edgesJSON)
	logPrompts(context, sysPrompt, userPrompt)
	return []llm.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// resolveEdgePrompt resolves conflicts between edges using TSV output.
func resolveEdgePrompt(context map[string]interface{}) ([]llm.Message, error) {
	sysPrompt := `You are a helpful assistant that determines whether extracted edges are duplicates or contradictions of existing edges.`

	existingEdges := context["existing_edges"]
	newEdge := context["new_edge"]
	edgeInvalidationCandidates := context["edge_invalidation_candidates"]
	edgeTypes := context["edge_types"]

	ensureASCII := true
	if val, ok := context["ensure_ascii"]; ok {
		if b, ok := val.(bool); ok {
			ensureASCII = b
		}
	}

	existingEdgesJSON, err := ToPromptJSON(existingEdges, ensureASCII, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal existing edges: %w", err)
	}

	edgeInvalidationCandidatesJSON, err := ToPromptJSON(edgeInvalidationCandidates, ensureASCII, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal edge invalidation candidates: %w", err)
	}

	edgeTypesJSON, err := ToPromptJSON(edgeTypes, ensureASCII, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal edge types: %w", err)
	}

	userPrompt := fmt.Sprintf(`
<NEW FACT>
%v
</NEW FACT>

<EXISTING FACTS>
%s
</EXISTING FACTS>

<FACT INVALIDATION CANDIDATES>
%s
</FACT INVALIDATION CANDIDATES>

<FACT TYPES>
%s
</FACT TYPES>

Task:
You have THREE separate lists: NEW FACT (string), EXISTING FACTS (with 'id' field), and FACT INVALIDATION CANDIDATES (with 'id' field starting from 0).

1. DUPLICATE DETECTION:
   - If the NEW FACT represents identical factual information as any fact in EXISTING FACTS, identify which ones.
   - Facts with similar information that contain key differences should NOT be marked as duplicates.
   - Return a comma-separated list of id values from EXISTING FACTS that are duplicates.
   - If no duplicates, return an empty string.

2. FACT TYPE CLASSIFICATION:
   - Given the predefined FACT TYPES, determine if the NEW FACT should be classified as one of these types.
   - Return the fact type name or DEFAULT if NEW FACT is not one of the FACT TYPES.

3. CONTRADICTION DETECTION:
   - Based on FACT INVALIDATION CANDIDATES and NEW FACT, determine which facts the new fact contradicts.
   - Return a comma-separated list of id values from FACT INVALIDATION CANDIDATES.
   - If no contradictions, return an empty string.

IMPORTANT:
- duplicate_facts: Use ONLY 'id' values from EXISTING FACTS as a comma-separated string (e.g., "1,3,5" or "" for none)
- contradicted_facts: Use ONLY 'id' values from FACT INVALIDATION CANDIDATES as a comma-separated string (e.g., "0,2" or "" for none)
- These are two separate lists with independent id ranges

Guidelines:
1. Some facts may be very similar but will have key differences, particularly around numeric values.
   Do not mark these facts as duplicates.

Output Format:
Provide your answer as a single-row TSV (tab-separated values) with the following schema:

<SCHEMA>
duplicate_facts: string (comma-separated integers or empty)
contradicted_facts: string (comma-separated integers or empty)
fact_type: string
</SCHEMA>

<EXAMPLE>
duplicate_facts	contradicted_facts	fact_type
1,3	0,2	KNOWS

</EXAMPLE>

Provide only the TSV header and data row. Finish your response with a new line.
`, newEdge, existingEdgesJSON, edgeInvalidationCandidatesJSON, edgeTypesJSON)
	logPrompts(context, sysPrompt, userPrompt)
	return []llm.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// NewDedupeEdgesVersions creates a new DedupeEdgesVersions instance.
func NewDedupeEdgesVersions() *DedupeEdgesVersions {
	return &DedupeEdgesVersions{
		EdgePrompt:        NewPromptVersion(dedupeEdgePrompt),
		EdgeListPrompt:    NewPromptVersion(dedupeEdgeListPrompt),
		ResolveEdgePrompt: NewPromptVersion(resolveEdgePrompt),
	}
}
