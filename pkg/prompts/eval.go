package prompts

import (
	"fmt"

	"github.com/soundprediction/go-graphiti/pkg/llm"
)

// EvalPrompt defines the interface for evaluation prompts.
type EvalPrompt interface {
	Evaluate() PromptVersion
}

// EvalVersions holds all versions of evaluation prompts.
type EvalVersions struct {
	EvaluatePrompt PromptVersion
}

func (e *EvalVersions) Evaluate() PromptVersion { return e.EvaluatePrompt }

// evaluatePrompt evaluates the quality of extracted information.
func evaluatePrompt(context map[string]interface{}) ([]llm.Message, error) {
	sysPrompt := `You are an expert evaluator that assesses the quality and accuracy of knowledge graph extraction.`

	originalText := context["original_text"]
	extractedNodes := context["extracted_nodes"]
	extractedEdges := context["extracted_edges"]
	criteria := context["criteria"]

	ensureASCII := true
	if val, ok := context["ensure_ascii"]; ok {
		if b, ok := val.(bool); ok {
			ensureASCII = b
		}
	}

	extractedNodesJSON, err := ToPromptJSON(extractedNodes, ensureASCII, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal extracted nodes: %w", err)
	}

	extractedEdgesJSON, err := ToPromptJSON(extractedEdges, ensureASCII, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal extracted edges: %w", err)
	}

	userPrompt := fmt.Sprintf(`
<ORIGINAL TEXT>
%v
</ORIGINAL TEXT>
<EXTRACTED NODES>
%s
</EXTRACTED NODES>
<EXTRACTED EDGES>
%s
</EXTRACTED EDGES>
<EVALUATION CRITERIA>
%v
</EVALUATION CRITERIA>

Evaluate the quality of the knowledge graph extraction based on the provided criteria.

Provide:
1. A numerical score (0-100)
2. An explanation of the evaluation
3. Specific areas for improvement
4. Assessment of completeness and accuracy
`, originalText, extractedNodesJSON, extractedEdgesJSON, criteria)

	return []llm.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// NewEvalVersions creates a new EvalVersions instance.
func NewEvalVersions() *EvalVersions {
	return &EvalVersions{
		EvaluatePrompt: NewPromptVersion(evaluatePrompt),
	}
}