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
func (d *DedupeEdgesVersions) EdgeList() PromptVersion   { return d.EdgeListPrompt }
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

Determine if the NEW FACT is a duplicate of or contradicts any of the EXISTING FACTS.
Mark facts as duplicates if they represent the same relationship.
Mark facts as contradicted if the new fact invalidates existing facts.
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

	return []llm.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// resolveEdgePrompt resolves conflicts between edges.
func resolveEdgePrompt(context map[string]interface{}) ([]llm.Message, error) {
	sysPrompt := `You are a helpful assistant that resolves conflicts between edges.`

	conflictingEdges := context["conflicting_edges"]

	ensureASCII := true
	if val, ok := context["ensure_ascii"]; ok {
		if b, ok := val.(bool); ok {
			ensureASCII = b
		}
	}

	conflictingEdgesJSON, err := ToPromptJSON(conflictingEdges, ensureASCII, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal conflicting edges: %w", err)
	}

	userPrompt := fmt.Sprintf(`
Resolve conflicts between the following edges:

Conflicting Edges:
%s

Task:
Determine which edges should be kept and which should be invalidated.
`, conflictingEdgesJSON)

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