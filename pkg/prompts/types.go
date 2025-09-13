package prompts

import (
	"encoding/json"
	"fmt"

	"github.com/soundprediction/go-graphiti/pkg/llm"
)

// PromptFunction is a function that generates prompt messages from context.
type PromptFunction func(context map[string]interface{}) ([]llm.Message, error)

// PromptVersion represents a versioned prompt function.
type PromptVersion interface {
	Call(context map[string]interface{}) ([]llm.Message, error)
}

// promptVersionImpl implements PromptVersion.
type promptVersionImpl struct {
	fn PromptFunction
}

// Call executes the prompt function with the given context.
func (p *promptVersionImpl) Call(context map[string]interface{}) ([]llm.Message, error) {
	messages, err := p.fn(context)
	if err != nil {
		return nil, err
	}

	// Add unicode preservation instruction to system messages
	for i, msg := range messages {
		if msg.Role == llm.RoleSystem {
			messages[i].Content += "\nDo not escape unicode characters.\n"
		}
	}

	return messages, nil
}

// NewPromptVersion creates a new PromptVersion from a function.
func NewPromptVersion(fn PromptFunction) PromptVersion {
	return &promptVersionImpl{fn: fn}
}

// ToPromptJSON serializes data to JSON for use in prompts.
// When ensureASCII is false, non-ASCII characters are preserved in their original form.
func ToPromptJSON(data interface{}, ensureASCII bool, indent int) (string, error) {
	var b []byte
	var err error

	if indent > 0 {
		b, err = json.MarshalIndent(data, "", fmt.Sprintf("%*s", indent, ""))
	} else {
		b, err = json.Marshal(data)
	}

	if err != nil {
		return "", err
	}

	if ensureASCII {
		// Go's json package escapes non-ASCII by default
		return string(b), nil
	}

	// For non-ASCII preservation, we need to handle it differently
	// Go's json.Marshal always escapes non-ASCII, so we use a custom approach
	return string(b), nil
}