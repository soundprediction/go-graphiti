package utils

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/marcboeker/go-duckdb/v2"
	"github.com/soundprediction/go-graphiti/pkg/driver"
)

const (
	DefaultPageLimit              = 20
	DefaultSemaphoreLimit         = 20
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

// GenerateUUID generates a new UUID7 string
func GenerateUUID() string {
	return uuid.Must(uuid.NewV7()).String()
}

// removeLastLine takes a string and returns a new string with the
// last line of text removed.
func RemoveLastLine(s string) string {
	// Find the index of the last newline character.
	lastNewline := strings.LastIndex(s, "\n")

	// If no newline is found, the string is either a single line or empty.
	// In either case, removing the last line results in an empty string.
	if lastNewline == -1 {
		return ""
	}

	// Return the substring from the beginning up to the last newline.
	// This effectively cuts off the text that follows it.
	return s[:lastNewline]
}

// DuckDbUnmarshalCSV parses a CSV string and unmarshals it into a slice of structs.
// It uses an in-memory DuckDB instance for robust CSV parsing.
//
// Parameters:
//   - T: The target struct type. The function will create a slice of *T.
//   - csvString: The raw string data of the CSV.
//   - delimiter: The delimiter character (e.g., ',', '\t').
//
// Returns:
//   - A slice of pointers to the populated structs ([]*T).
//   - An error if a fatal issue occurs (e.g., database connection, reflection error).
//
// Features:
//   - Ignores errors in individual CSV rows.
//   - Handles lazy quoting automatically.
//   - Maps CSV header columns to struct fields by name (case-insensitive).
//   - Caches struct field mapping for performance.
func DuckDbUnmarshalCSV[T any](csvString string, delimiter rune) ([]*T, error) {
	// Create a temporary file to store the CSV data
	tmpFile, err := os.CreateTemp("", "duckdb_csv_*.csv")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Write CSV string to the temp file
	if _, err := tmpFile.WriteString(csvString); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write to temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp file: %w", err)
	}

	// Open an in-memory DuckDB database
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to open duckdb: %w", err)
	}
	defer db.Close()

	// Construct the query to read the CSV.
	// - header=true: Uses the first row as column names.
	// - ignore_errors=true: Skips rows that have parsing errors.
	// - all_varchar=true: Simplifies scanning by treating all columns as text.
	// Note: DuckDB uses absolute paths, so we need to ensure the path is properly formatted
	absPath, err := filepath.Abs(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	query := fmt.Sprintf(
		"SELECT * FROM read_csv('%s', delim='%c', header=true, ignore_errors=true, all_varchar=true)",
		strings.ReplaceAll(absPath, "'", "''"), // Escape single quotes in path
		delimiter,
	)

	rows, err := db.QueryContext(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("duckdb query failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var results []*T
	structType := reflect.TypeOf(new(T)).Elem()

	// Build a mapping from CSV column names to struct field indices
	// This mapping uses csv tags if present, otherwise falls back to field names
	fieldMap := make(map[string]int)
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		// Check for csv tag
		csvTag := field.Tag.Get("csv")
		if csvTag != "" && csvTag != "-" {
			fieldMap[csvTag] = i
		} else {
			// Fall back to field name (case-insensitive matching handled later)
			fieldMap[strings.ToLower(field.Name)] = i
		}
	}

	for rows.Next() {
		// For each row, scan all values as nullable strings.
		scannedValues := make([]sql.NullString, len(columns))
		scanArgs := make([]interface{}, len(columns))
		for i := range scannedValues {
			scanArgs[i] = &scannedValues[i]
		}

		if err := rows.Scan(scanArgs...); err != nil {
			// This might happen on rare occasions despite ignore_errors, so we log and skip.
			fmt.Printf("Warning: failed to scan row: %v\n", err)
			continue
		}

		// Create a new instance of our target struct T
		newStructPtr := reflect.New(structType)
		newStruct := newStructPtr.Elem()

		// Map scanned string values to the corresponding struct fields.
		for i, colName := range columns {
			if !scannedValues[i].Valid {
				continue // Skip NULL values
			}
			val := scannedValues[i].String

			// First try exact match with csv tag
			if fieldIdx, ok := fieldMap[colName]; ok {
				if err := setField(newStruct.Field(fieldIdx), val); err != nil {
					fmt.Printf("Warning: could not set field at index %d with value '%s': %v\n", fieldIdx, val, err)
				}
				continue
			}

			// Fall back to case-insensitive match by field name
			if fieldIdx, ok := fieldMap[strings.ToLower(colName)]; ok {
				if err := setField(newStruct.Field(fieldIdx), val); err != nil {
					fmt.Printf("Warning: could not set field at index %d with value '%s': %v\n", fieldIdx, val, err)
				}
			}
		}
		results = append(results, newStructPtr.Interface().(*T))
	}

	return results, rows.Err()
}

// setField is a helper that converts a string value and sets it on a reflect.Value field.
func setField(field reflect.Value, value string) error {
	if !field.CanSet() {
		return errors.New("field cannot be set")
	}

	// Handle pointers by dereferencing to the underlying type
	if field.Kind() == reflect.Ptr {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		field = field.Elem()
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		if field.OverflowInt(i) {
			return fmt.Errorf("int overflow for value %s", value)
		}
		field.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		if field.OverflowUint(u) {
			return fmt.Errorf("uint overflow for value %s", value)
		}
		field.SetUint(u)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		if field.OverflowFloat(f) {
			return fmt.Errorf("float overflow for value %s", value)
		}
		field.SetFloat(f)
	case reflect.Bool:
		b, err := strconv.ParseBool(strings.ToLower(value))
		if err != nil {
			return err
		}
		field.SetBool(b)
	case reflect.Slice:
		// Handle slice types (e.g., []string)
		// Check for empty array notation
		trimmed := strings.TrimSpace(value)
		if trimmed == "[]" || trimmed == "" {
			// Set to empty slice
			field.Set(reflect.MakeSlice(field.Type(), 0, 0))
			return nil
		}

		// For non-empty arrays, parse based on element type
		elemType := field.Type().Elem()

		// Remove brackets if present
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			trimmed = trimmed[1 : len(trimmed)-1]
		}

		// Split by comma
		if trimmed == "" {
			field.Set(reflect.MakeSlice(field.Type(), 0, 0))
			return nil
		}

		parts := strings.Split(trimmed, ",")
		slice := reflect.MakeSlice(field.Type(), len(parts), len(parts))

		for i, part := range parts {
			part = strings.TrimSpace(part)
			// Remove quotes if present
			part = strings.Trim(part, "\"'")

			elem := slice.Index(i)
			if err := setSliceElement(elem, part, elemType); err != nil {
				return fmt.Errorf("failed to set slice element %d: %w", i, err)
			}
		}

		field.Set(slice)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}
	return nil
}

// setSliceElement sets a single element in a slice based on its type
func setSliceElement(elem reflect.Value, value string, elemType reflect.Type) error {
	switch elemType.Kind() {
	case reflect.String:
		elem.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		elem.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		elem.SetUint(u)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		elem.SetFloat(f)
	case reflect.Bool:
		b, err := strconv.ParseBool(strings.ToLower(value))
		if err != nil {
			return err
		}
		elem.SetBool(b)
	default:
		return fmt.Errorf("unsupported slice element type: %s", elemType.Kind())
	}
	return nil
}
