package main

import (
	"testing"
)

// TestTruncateString tests the utility function
func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{
			input:    "Hello world",
			maxLen:   20,
			expected: "Hello world",
		},
		{
			input:    "This is a very long string that should be truncated",
			maxLen:   10,
			expected: "This is...",
		},
		{
			input:    "Short",
			maxLen:   5,
			expected: "Short",
		},
		{
			input:    "Exactly ten",
			maxLen:   11,
			expected: "Exactly ten",
		},
		{
			input:    "Too long for limit",
			maxLen:   8,
			expected: "Too l...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

// TestExampleCompilation ensures the example compiles without issues
func TestExampleCompilation(t *testing.T) {
	// This test just ensures the example compiles and imports work correctly
	// The actual main() function is not called to avoid requiring external dependencies
	t.Log("Example compiles successfully")
}