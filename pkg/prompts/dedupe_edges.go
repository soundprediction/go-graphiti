package prompts

import (
	"fmt"
	"log/slog"

	"github.com/soundprediction/go-graphiti/pkg/llm"
	"github.com/soundprediction/go-graphiti/pkg/types"
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
// Uses TSV format for episodes and facts to reduce token usage and improve LLM parsing.
func dedupeEdgePrompt(context map[string]interface{}) ([]types.Message, error) {
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

	previousEpisodesTSV, err := ToPromptCSV(previousEpisodes, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal previous episodes: %w", err)
	}

	newFactTSV, err := ToPromptCSV(newFact, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal new fact: %w", err)
	}

	existingFactsTSV, err := ToPromptCSV(existingFacts, ensureASCII)
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

Note: PREVIOUS MESSAGES, NEW FACT, and EXISTING FACTS are provided in TSV (tab-separated values) format.

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
`, previousEpisodesTSV, episodeContent, newFactTSV, existingFactsTSV)
	logPrompts(context["logger"].(*slog.Logger), sysPrompt, userPrompt)
	return []types.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// dedupeEdgeListPrompt handles batch edge deduplication.
// Uses TSV format for edges to reduce token usage and improve LLM parsing.
func dedupeEdgeListPrompt(context map[string]interface{}) ([]types.Message, error) {
	sysPrompt := `You are a helpful assistant that de-duplicates edges from edge lists.`

	edges := context["edges"]

	ensureASCII := true
	if val, ok := context["ensure_ascii"]; ok {
		if b, ok := val.(bool); ok {
			ensureASCII = b
		}
	}

	edgesTSV, err := ToPromptCSV(edges, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal edges: %w", err)
	}

	userPrompt := fmt.Sprintf(`
Given the following edges, identify unique facts and remove duplicates.

Edges are provided in TSV (tab-separated values) format:
%s

Task:
Return a list of unique facts, removing any duplicates.
`, edgesTSV)
	logPrompts(context["logger"].(*slog.Logger), sysPrompt, userPrompt)
	return []types.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// resolveEdgePrompt resolves conflicts between edges using TSV output.
func resolveEdgePrompt(context map[string]interface{}) ([]types.Message, error) {
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

	existingEdgesTSV, err := ToPromptCSV(existingEdges, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal existing edges: %w", err)
	}

	edgeInvalidationCandidatesTSV, err := ToPromptCSV(edgeInvalidationCandidates, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal edge invalidation candidates: %w", err)
	}

	// Filter out fact_type_description to reduce redundancy
	filteredEdgeTypes := filterEdgeTypes(edgeTypes)
	edgeTypesTSV, err := ToPromptCSV(filteredEdgeTypes, ensureASCII)
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

Note: EXISTING FACTS, FACT INVALIDATION CANDIDATES, and FACT TYPES are provided in TSV (tab-separated values) format.

Task:
You have THREE separate lists: NEW FACT (string), EXISTING FACTS (TSV format with 'id' and 'fact' columns), and FACT INVALIDATION CANDIDATES (TSV format with 'id' field).

1. DUPLICATE DETECTION:
   - If the NEW FACT represents identical factual information as any fact in EXISTING FACTS, identify which ones.
   - Facts with similar information that contain key differences should NOT be marked as duplicates.
   - Return a list of id values from EXISTING FACTS that are duplicates.
   - If no duplicates, return an empty list.

2. FACT TYPE CLASSIFICATION:
   - Given the predefined FACT TYPES, determine if the NEW FACT should be classified as one of these types.
   - Return the fact type name or DEFAULT if NEW FACT is not one of the FACT TYPES.

3. CONTRADICTION DETECTION:
   - Based on FACT INVALIDATION CANDIDATES and NEW FACT, determine which facts the new fact contradicts.
   - Return a list of id values from FACT INVALIDATION CANDIDATES.
   - If no contradictions, return an empty list.

IMPORTANT:
- duplicate_facts: Use ONLY 'id' values from EXISTING FACTS as a list of strings
- contradicted_facts: Use ONLY 'id' values from FACT INVALIDATION CANDIDATES as a list of strings
- These are two separate lists with independent id ranges

Guidelines:
1. Some facts may be very similar but will have key differences, particularly around numeric values.
   Do not mark these facts as duplicates.

Output Format:
Provide your answer as a single-row TSV (tab-separated values) with the following schema:

<SCHEMA>
duplicate_facts: list[string] 
contradicted_facts: list[string]
fact_type: string
</SCHEMA>

<EXAMPLE>
duplicate_facts\tcontradicted_facts\tfact_type
["019a0cd8-20db-7334-b493-f7242f062cce","019a0cd8-20db-7353-b922-cfc410ea9396"]\t["000b7f15-aa7b-4270-b517-e433a98e4931"]\tKNOWS

</EXAMPLE>

Provide only the TSV header and data row. Finish your response with a new line.
`, newEdge, existingEdgesTSV, edgeInvalidationCandidatesTSV, edgeTypesTSV)
	logPrompts(context["logger"].(*slog.Logger), sysPrompt, userPrompt)
	return []types.Message{
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
