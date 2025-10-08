package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
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

// RemoveThinkTags removes <think> tags and everything in between them from a string.
func RemoveThinkTags(input string) string {
	re := regexp.MustCompile(`(?s)<think>.*?</think>`)
	return re.ReplaceAllString(input, "")
}

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
	ok := (err == nil)
	return ok, err
}

// AppendOverlap appends s2 to s1, removing any overlapping part.
// It finds the longest suffix of s1 that is also a prefix of s2 and
// combines them to avoid duplicating the overlapping section.
func AppendOverlap(s1, s2 string) string {
	len1 := len(s1)
	len2 := len(s2)

	// Determine the maximum possible overlap length to check.
	// This can't be longer than the shorter of the two strings.
	maxOverlap := len1
	if len2 < len1 {
		maxOverlap = len2
	}

	// Iterate backwards from the longest possible overlap.
	// The first match found will be the longest one.
	for i := maxOverlap; i > 0; i-- {
		// Check if the suffix of s1 matches the prefix of s2.
		if s1[len1-i:] == s2[:i] {
			// If a match is found, append the non-overlapping part of s2 and return.
			return s1 + s2[i:]
		}
	}

	// If no overlap is found after checking all possibilities,
	// simply concatenate the two strings.
	return s1 + s2
}
func truncateToLastCloseBrace(s string) string {
	lastIndex := strings.LastIndex(s, "}")
	if lastIndex == -1 {
		return "" // No closing brace found
	}
	return s[:lastIndex+1]
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

	responses := make([]string, 0)

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Make LLM call
		if attempt > 0 {
			workingMessages[1].Content = messages[1].Content + "\nFinish your work:\n" + strings.TrimSpace(accumulatedResponse)
		}

		// fmt.Printf("workingMessages[1].Content: %v\n", workingMessages[1].Content)
		response, err := llmClient.Chat(ctx, workingMessages)
		responses = append(responses, response.Content)
		if err != nil {
			lastError = fmt.Errorf("LLM call failed on attempt %d: %w", attempt+1, err)
			continue
		}

		if response == nil || response.Content == "" {
			lastError = fmt.Errorf("empty response from LLM on attempt %d", attempt+1)
			// ask the LLM to fix the output
			continue
		}
		startLen := len(accumulatedResponse)
		accumulatedResponse = AppendOverlap(strings.TrimSpace(accumulatedResponse), strings.TrimSpace((response.Content)))
		afterLen := len(accumulatedResponse)
		gap := afterLen - startLen
		ok, _ := isValidJson(RemoveThinkTags(accumulatedResponse))

		if ok {
			return RemoveThinkTags(accumulatedResponse), nil
		}

		if attempt > 0 && gap == 0 {
			accumulatedResponse = truncateToLastCloseBrace(accumulatedResponse)
			resp, _ := jsonrepair.RepairJSON(RemoveThinkTags(accumulatedResponse))
			fmt.Printf("resp: %v\n", resp)
			return resp, nil
		}

	}

	if lastError != nil {
		return RemoveThinkTags(accumulatedResponse), fmt.Errorf("failed after %d attempts: %w", maxRetries+1, lastError)
	}

	return RemoveThinkTags(accumulatedResponse), fmt.Errorf("failed to generate valid JSON after %d attempts", maxRetries+1)
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
			accumulatedResponse = strings.TrimSpace(response.Content)
		} else {
			// For continuation, append the new content
			accumulatedResponse += strings.TrimSpace(response.Content)
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
