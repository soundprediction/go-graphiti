package main

import (
	"testing"
)

// TestHelperFunctions tests the utility helper functions
func TestHelperFunctions(t *testing.T) {
	t.Run("floatPtr", func(t *testing.T) {
		input := float32(0.7)
		result := floatPtr(input)
		if result == nil {
			t.Error("floatPtr returned nil")
		}
		if *result != input {
			t.Errorf("floatPtr(%f) = %f, want %f", input, *result, input)
		}
	})

	t.Run("intPtr", func(t *testing.T) {
		input := 100
		result := intPtr(input)
		if result == nil {
			t.Error("intPtr returned nil")
		}
		if *result != input {
			t.Errorf("intPtr(%d) = %d, want %d", input, *result, input)
		}
	})
}

// TestExampleCompilation ensures the example compiles without issues
func TestExampleCompilation(t *testing.T) {
	// This test just ensures the example compiles and imports work correctly
	// The actual main() function is not called to avoid requiring external dependencies
	t.Log("Example compiles successfully")
}