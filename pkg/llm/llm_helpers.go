package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	jsonrepair "github.com/RealAlexandreAI/json-repair"
)

// GenerateJSONResponseWithContinuation makes repeated LLM calls with continuation prompts
// until valid JSON is received or max retries is reached.
//
// Parameters:
//   - ctx: Context for the LLM call
//   - llmClient: The LLM client to use
//   - systemPrompt: The initial system/instruction prompt
//   - userPrompt: The user's request prompt
//   - targetStruct: A pointer to the struct to unmarshal JSON into (for validation)
//   - maxRetries: Maximum number of continuation attempts (default 3 if <= 0)
//
// Returns:
//   - The final JSON string (may be partial if all retries exhausted)
//   - Error if all retries fail or if there's a critical error
//
// Example:
//
//	type MyStruct struct {
//	    Name string `json:"name"`
//	    Items []string `json:"items"`
//	}
//	var result MyStruct
//	jsonStr, err := GenerateJSONResponseWithContinuation(
//	    ctx, llmClient,
//	    "You are a JSON generator. Return only valid JSON.",
//	    "Generate a list of 10 pregnancy tips",
//	    &result,
//	    5,
//	)
func GenerateJSONResponseWithContinuation(
	ctx context.Context,
	llmClient Client,
	systemPrompt string,
	userPrompt string,
	targetStruct interface{},
	maxRetries int,
) (string, error) {
	// Build initial messages
	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	return GenerateJSONResponseWithContinuationMessages(ctx, llmClient, messages, targetStruct, maxRetries)
}

func isValidJson(s string) (bool, error) {
	var js json.RawMessage
	err := json.Unmarshal([]byte(s), &js)
	return err != nil, err
}

// GenerateJSONResponseWithContinuationMessages makes repeated LLM calls with continuation prompts
// until valid JSON is received or max retries is reached. This version accepts pre-built messages.
//
// Parameters:
//   - ctx: Context for the LLM call
//   - llmClient: The LLM client to use
//   - messages: The initial message history
//   - targetStruct: A pointer to the struct to unmarshal JSON into (for validation)
//   - maxRetries: Maximum number of continuation attempts (default 3 if <= 0)
//
// Returns:
//   - The final JSON string (may be partial if all retries exhausted)
//   - Error if all retries fail or if there's a critical error
func GenerateJSONResponseWithContinuationMessages(
	ctx context.Context,
	llmClient Client,
	messages []Message,
	targetStruct interface{},
	maxRetries int,
) (string, error) {
	if maxRetries <= 0 {
		maxRetries = 20
	}

	// Make a copy of messages to avoid modifying the original slice
	workingMessages := make([]Message, len(messages))
	copy(workingMessages, messages)
	var accumulatedResponse string
	var lastError error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Make LLM call
		if attempt > 0 {
			workingMessages[1].Content += messages[1].Content + "\n Resume, do not replicate:\n" + accumulatedResponse
		}

		// fmt.Printf("workingMessages[1].Content: %v\n", workingMessages[1].Content)
		response, err := llmClient.Chat(ctx, workingMessages)
		if err != nil {
			lastError = fmt.Errorf("LLM call failed on attempt %d: %w", attempt+1, err)
			continue
		}

		if response == nil || response.Content == "" {
			lastError = fmt.Errorf("empty response from LLM on attempt %d", attempt+1)
			// ask the LLM to fix the output
			continue
		}

		// Accumulate the response
		if attempt == 0 {
			accumulatedResponse = response.Content
		} else {
			// For continuation, append the new content
			accumulatedResponse += response.Content
		}
		// fmt.Printf("accumulatedResponse: %v\n", accumulatedResponse)
		// Try to validate JSON without repair (don't repair during continuation)
		// First unmarshal to handle potential quoted JSON

		// check if the response is a full json
		ok, err := isValidJson(response.Content)
		if err != nil {
			if ok {
				repairedJSON, _ := jsonrepair.RepairJSON(accumulatedResponse)
				return repairedJSON, nil
			}
		}

		// Success! Valid JSON that matches the schema
		// Now repair the JSON before returning
		repairedJSON, _ := jsonrepair.RepairJSON(string(accumulatedResponse))
		return repairedJSON, nil
	}

	// All retries exhausted - try to repair what we have
	repairedJSON, _ := jsonrepair.RepairJSON(accumulatedResponse)
	if lastError != nil {
		return repairedJSON, fmt.Errorf("failed after %d attempts: %w", maxRetries+1, lastError)
	}

	return repairedJSON, fmt.Errorf("failed to generate valid JSON after %d attempts", maxRetries+1)
}

// GenerateJSONWithContinuation is a simpler version that doesn't validate against a struct
// and just ensures valid JSON is returned.
func GenerateJSONWithContinuation(
	ctx context.Context,
	llmClient Client,
	systemPrompt string,
	userPrompt string,
	maxRetries int,
) (string, error) {
	if maxRetries <= 0 {
		maxRetries = 3
	}

	// Build initial messages
	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	var accumulatedResponse string
	var lastError error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Make LLM call
		response, err := llmClient.Chat(ctx, messages)
		if err != nil {
			lastError = fmt.Errorf("LLM call failed on attempt %d: %w", attempt+1, err)
			continue
		}

		if response == nil || response.Content == "" {
			lastError = fmt.Errorf("empty response from LLM on attempt %d", attempt+1)
			continue
		}

		// Accumulate the response
		if attempt == 0 {
			accumulatedResponse = response.Content
		} else {
			// For continuation, append the new content
			accumulatedResponse += response.Content
		}

		// Try to repair JSON
		repairedJSON, _ := jsonrepair.RepairJSON(accumulatedResponse)

		// Validate it's proper JSON
		var testJSON interface{}
		err = json.Unmarshal([]byte(repairedJSON), &testJSON)
		if err != nil {
			// JSON is invalid or incomplete, try continuation
			lastError = fmt.Errorf("invalid JSON on attempt %d: %w", attempt+1, err)

			if attempt < maxRetries {
				// Add continuation prompt
				messages = append(messages, Message{
					Role:    "assistant",
					Content: accumulatedResponse,
				})
				messages = append(messages, Message{
					Role:    "user",
					Content: "The JSON response was incomplete or invalid. Please continue from where you left off and complete the JSON:",
				})
			}
			continue
		}

		// Success! Valid JSON
		return repairedJSON, nil
	}

	// All retries exhausted
	if lastError != nil {
		return accumulatedResponse, fmt.Errorf("failed after %d attempts: %w", maxRetries+1, lastError)
	}

	return accumulatedResponse, fmt.Errorf("failed to generate valid JSON after %d attempts", maxRetries+1)
}

// ExtractJSONFromResponse attempts to extract JSON from LLM responses that may contain
// markdown code blocks or other surrounding text.
func ExtractJSONFromResponse(response string) string {
	// Remove markdown code blocks if present
	response = strings.TrimSpace(response)

	// Check for ```json ... ``` pattern
	if strings.Contains(response, "```json") {
		start := strings.Index(response, "```json")
		end := strings.Index(response[start+7:], "```")
		if end != -1 {
			return strings.TrimSpace(response[start+7 : start+7+end])
		}
	}

	// Check for ``` ... ``` pattern
	if strings.HasPrefix(response, "```") {
		lines := strings.Split(response, "\n")
		if len(lines) > 2 {
			// Remove first and last line (the ``` markers)
			return strings.TrimSpace(strings.Join(lines[1:len(lines)-1], "\n"))
		}
	}

	// Try to find JSON object boundaries
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")
	if jsonStart != -1 && jsonEnd != -1 && jsonEnd > jsonStart {
		return response[jsonStart : jsonEnd+1]
	}

	// Try to find JSON array boundaries
	jsonStart = strings.Index(response, "[")
	jsonEnd = strings.LastIndex(response, "]")
	if jsonStart != -1 && jsonEnd != -1 && jsonEnd > jsonStart {
		return response[jsonStart : jsonEnd+1]
	}

	// Return as-is if no extraction possible
	return response
}
