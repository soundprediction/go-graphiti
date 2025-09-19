package utils

import (
	"errors"
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/soundprediction/go-graphiti/pkg/driver"
)

const (
	DefaultPageLimit          = 20
	DefaultSemaphoreLimit     = 20
	DefaultMaxReflexionIterations = 0
)

var (
	// ErrInvalidGroupID is returned when a group ID contains invalid characters
	ErrInvalidGroupID = errors.New("group ID contains invalid characters")
	// ErrInvalidEntityType is returned when an entity type is invalid
	ErrInvalidEntityType = errors.New("invalid entity type")
)

// GetUseParallelRuntime returns whether to use parallel runtime based on environment variable
func GetUseParallelRuntime() bool {
	val := os.Getenv("USE_PARALLEL_RUNTIME")
	if val == "" {
		return false
	}
	useParallel, _ := strconv.ParseBool(val)
	return useParallel
}

// GetSemaphoreLimit returns the semaphore limit from environment variable or default
func GetSemaphoreLimit() int {
	val := os.Getenv("SEMAPHORE_LIMIT")
	if val == "" {
		return DefaultSemaphoreLimit
	}
	limit, err := strconv.Atoi(val)
	if err != nil {
		return DefaultSemaphoreLimit
	}
	return limit
}

// GetMaxReflexionIterations returns the max reflexion iterations from environment variable or default
func GetMaxReflexionIterations() int {
	val := os.Getenv("MAX_REFLEXION_ITERATIONS")
	if val == "" {
		return DefaultMaxReflexionIterations
	}
	iterations, err := strconv.Atoi(val)
	if err != nil {
		return DefaultMaxReflexionIterations
	}
	return iterations
}

// ParseDBDate parses various date formats from database responses
func ParseDBDate(inputDate interface{}) (*time.Time, error) {
	switch v := inputDate.(type) {
	case time.Time:
		return &v, nil
	case string:
		if v == "" {
			return nil, nil
		}
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			// Try parsing ISO format without timezone
			parsed, err = time.Parse("2006-01-02T15:04:05", v)
			if err != nil {
				return nil, fmt.Errorf("failed to parse date string %q: %w", v, err)
			}
		}
		return &parsed, nil
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported date type: %T", v)
	}
}

// GetDefaultGroupID differentiates the default group id based on the database type
func GetDefaultGroupID(provider driver.GraphProvider) string {
	if provider == driver.GraphProviderFalkorDB {
		return "_"
	}
	return ""
}

// LuceneSanitize escapes special characters from a query before passing into Lucene
func LuceneSanitize(query string) string {
	// Escape special characters: + - && || ! ( ) { } [ ] ^ " ~ * ? : \ /
	replacer := strings.NewReplacer(
		"+", `\+`,
		"-", `\-`,
		"&", `\&`,
		"|", `\|`,
		"!", `\!`,
		"(", `\(`,
		")", `\)`,
		"{", `\{`,
		"}", `\}`,
		"[", `\[`,
		"]", `\]`,
		"^", `\^`,
		"\"", `\"`,
		"~", `\~`,
		"*", `\*`,
		"?", `\?`,
		":", `\:`,
		"\\", `\\`,
		"/", `\/`,
		"O", `\O`,
		"R", `\R`,
		"N", `\N`,
		"T", `\T`,
		"A", `\A`,
		"D", `\D`,
	)
	return replacer.Replace(query)
}

// NormalizeL2 normalizes a vector using L2 normalization
func NormalizeL2(embedding []float64) []float64 {
	if len(embedding) == 0 {
		return embedding
	}

	// Calculate the L2 norm
	var norm float64
	for _, val := range embedding {
		norm += val * val
	}
	norm = math.Sqrt(norm)

	// Avoid division by zero
	if norm == 0 {
		return embedding
	}

	// Normalize the vector
	normalized := make([]float64, len(embedding))
	for i, val := range embedding {
		normalized[i] = val / norm
	}

	return normalized
}

// NormalizeL2Float32 normalizes a float32 vector using L2 normalization
func NormalizeL2Float32(embedding []float32) []float32 {
	if len(embedding) == 0 {
		return embedding
	}

	// Calculate the L2 norm
	var norm float32
	for _, val := range embedding {
		norm += val * val
	}
	norm = float32(math.Sqrt(float64(norm)))

	// Avoid division by zero
	if norm == 0 {
		return embedding
	}

	// Normalize the vector
	normalized := make([]float32, len(embedding))
	for i, val := range embedding {
		normalized[i] = val / norm
	}

	return normalized
}

// ValidateGroupID validates that a group_id contains only ASCII alphanumeric characters, dashes, and underscores
func ValidateGroupID(groupID string) error {
	// Allow empty string (default case)
	if groupID == "" {
		return nil
	}

	// Check if string contains only ASCII alphanumeric characters, dashes, or underscores
	// Pattern matches: letters (a-z, A-Z), digits (0-9), hyphens (-), and underscores (_)
	matched, err := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, groupID)
	if err != nil {
		return fmt.Errorf("failed to validate group ID: %w", err)
	}

	if !matched {
		return fmt.Errorf("%w: group ID %q contains invalid characters", ErrInvalidGroupID, groupID)
	}

	return nil
}

// ValidateExcludedEntityTypes validates that excluded entity types are valid type names
func ValidateExcludedEntityTypes(excludedEntityTypes []string, availableTypes []string) error {
	if len(excludedEntityTypes) == 0 {
		return nil
	}

	// Build set of available type names
	availableSet := make(map[string]bool)
	availableSet["Entity"] = true // Default type is always available
	for _, t := range availableTypes {
		availableSet[t] = true
	}

	// Check for invalid type names
	var invalidTypes []string
	for _, excludedType := range excludedEntityTypes {
		if !availableSet[excludedType] {
			invalidTypes = append(invalidTypes, excludedType)
		}
	}

	if len(invalidTypes) > 0 {
		availableList := make([]string, 0, len(availableSet))
		for t := range availableSet {
			availableList = append(availableList, t)
		}
		return fmt.Errorf("%w: invalid excluded entity types: %v, available types: %v",
			ErrInvalidEntityType, invalidTypes, availableList)
	}

	return nil
}