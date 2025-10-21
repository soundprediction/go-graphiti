package prompts

import (
	"fmt"

	"github.com/soundprediction/go-graphiti/pkg/llm"
)

// EvalPrompt defines the interface for evaluation prompts.
type EvalPrompt interface {
	QAPrompt() PromptVersion
	EvalPrompt() PromptVersion
	QueryExpansion() PromptVersion
	EvalAddEpisodeResults() PromptVersion
}

// EvalVersions holds all versions of evaluation prompts.
type EvalVersions struct {
	qaPrompt             PromptVersion
	evalPrompt           PromptVersion
	queryExpansionPrompt PromptVersion
	evalAddEpisodePrompt PromptVersion
}

func (e *EvalVersions) QAPrompt() PromptVersion              { return e.qaPrompt }
func (e *EvalVersions) EvalPrompt() PromptVersion            { return e.evalPrompt }
func (e *EvalVersions) QueryExpansion() PromptVersion        { return e.queryExpansionPrompt }
func (e *EvalVersions) EvalAddEpisodeResults() PromptVersion { return e.evalAddEpisodePrompt }

// queryExpansionPrompt rephrases questions into queries used in a database retrieval system.
// Uses TSV format for query data to reduce token usage and improve LLM parsing.
func queryExpansionPrompt(context map[string]interface{}) ([]llm.Message, error) {
	sysPrompt := `You are an expert at rephrasing questions into queries used in a database retrieval system`

	query := context["query"]
	ensureASCII := false
	if val, ok := context["ensure_ascii"]; ok {
		if b, ok := val.(bool); ok {
			ensureASCII = b
		}
	}

	queryTSV, err := ToPromptCSV(query, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	userPrompt := fmt.Sprintf(`
Bob is asking Alice a question, are you able to rephrase the question into a simpler one about Alice in the third person
that maintains the relevant context?
<QUESTION>
%s
</QUESTION>

Note: Query data is provided in TSV (tab-separated values) format.
`, queryTSV)
	logPrompts(context, sysPrompt, userPrompt)
	return []llm.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// qaPrompt answers questions from Alice's first person perspective.
// Uses TSV format for entity summaries and facts to reduce token usage and improve LLM parsing.
func qaPrompt(context map[string]interface{}) ([]llm.Message, error) {
	sysPrompt := `You are Alice and should respond to all questions from the first person perspective of Alice`

	entitySummaries := context["entity_summaries"]
	facts := context["facts"]
	query := context["query"]

	ensureASCII := false
	if val, ok := context["ensure_ascii"]; ok {
		if b, ok := val.(bool); ok {
			ensureASCII = b
		}
	}

	entitySummariesTSV, err := ToPromptCSV(entitySummaries, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal entity summaries: %w", err)
	}

	factsTSV, err := ToPromptCSV(facts, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal facts: %w", err)
	}

	userPrompt := fmt.Sprintf(`
Your task is to briefly answer the question in the way that you think Alice would answer the question.
You are given the following entity summaries and facts to help you determine the answer to your question.

Note: ENTITY_SUMMARIES and FACTS are provided in TSV (tab-separated values) format.

<ENTITY_SUMMARIES>
%s
</ENTITY_SUMMARIES>
<FACTS>
%s
</FACTS>
<QUESTION>
%v
</QUESTION>
`, entitySummariesTSV, factsTSV, query)
	logPrompts(context, sysPrompt, userPrompt)
	return []llm.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// evalPrompt determines if answers to questions match a gold standard answer.
func evalPrompt(context map[string]interface{}) ([]llm.Message, error) {
	sysPrompt := `You are a judge that determines if answers to questions match a gold standard answer`

	query := context["query"]
	answer := context["answer"]
	response := context["response"]

	userPrompt := fmt.Sprintf(`
Given the QUESTION and the gold standard ANSWER determine if the RESPONSE to the question is correct or incorrect.
Although the RESPONSE may be more verbose, mark it as correct as long as it references the same topic
as the gold standard ANSWER. Also include your reasoning for the grade.
<QUESTION>
%v
</QUESTION>
<ANSWER>
%v
</ANSWER>
<RESPONSE>
%v
</RESPONSE>
`, query, answer, response)
	logPrompts(context, sysPrompt, userPrompt)
	return []llm.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// evalAddEpisodeResultsPrompt determines whether a baseline graph building result is better than a candidate.
// Uses TSV format for previous messages, baseline, and candidate data to reduce token usage and improve LLM parsing.
func evalAddEpisodeResultsPrompt(context map[string]interface{}) ([]llm.Message, error) {
	sysPrompt := `You are a judge that determines whether a baseline graph building result from a list of messages is better
than a candidate graph building result based on the same messages.`

	previousMessages := context["previous_messages"]
	message := context["message"]
	baseline := context["baseline"]
	candidate := context["candidate"]

	ensureASCII := false
	if val, ok := context["ensure_ascii"]; ok {
		if b, ok := val.(bool); ok {
			ensureASCII = b
		}
	}

	previousMessagesTSV, err := ToPromptCSV(previousMessages, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal previous messages: %w", err)
	}

	baselineTSV, err := ToPromptCSV(baseline, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal baseline: %w", err)
	}

	candidateTSV, err := ToPromptCSV(candidate, ensureASCII)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal candidate: %w", err)
	}

	userPrompt := fmt.Sprintf(`
Given the following PREVIOUS MESSAGES and MESSAGE, determine if the BASELINE graph data extracted from the
conversation is higher quality than the CANDIDATE graph data extracted from the conversation.

Return False if the BASELINE extraction is better, and True otherwise. If the CANDIDATE extraction and
BASELINE extraction are nearly identical in quality, return True. Add your reasoning for your decision to the reasoning field

<PREVIOUS MESSAGES>
%s
</PREVIOUS MESSAGES>
<MESSAGE>
%v
</MESSAGE>

<BASELINE>
%s
</BASELINE>

<CANDIDATE>
%s
</CANDIDATE>

Note: PREVIOUS MESSAGES, BASELINE, and CANDIDATE are provided in TSV (tab-separated values) format.
`, previousMessagesTSV, message, baselineTSV, candidateTSV)
	logPrompts(context, sysPrompt, userPrompt)
	return []llm.Message{
		llm.NewSystemMessage(sysPrompt),
		llm.NewUserMessage(userPrompt),
	}, nil
}

// NewEvalVersions creates a new EvalVersions instance.
func NewEvalVersions() *EvalVersions {
	return &EvalVersions{
		qaPrompt:             NewPromptVersion(qaPrompt),
		evalPrompt:           NewPromptVersion(evalPrompt),
		queryExpansionPrompt: NewPromptVersion(queryExpansionPrompt),
		evalAddEpisodePrompt: NewPromptVersion(evalAddEpisodeResultsPrompt),
	}
}
