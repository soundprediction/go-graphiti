package prompts

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"

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

// ToPromptCSV serializes data to CSV format for use in prompts.
// Data should be a slice of structs, maps, or a slice of slices.
// When ensureASCII is true, non-ASCII characters are escaped.
func ToPromptCSV(data interface{}, ensureASCII bool) (string, error) {
	v := reflect.ValueOf(data)

	// Handle non-slice types
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return "", fmt.Errorf("ToPromptCSV requires a slice or array, got %T", data)
	}

	if v.Len() == 0 {
		return "", nil
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	// Determine the type of elements
	firstElem := v.Index(0)

	switch firstElem.Kind() {
	case reflect.Map:
		// Handle slice of maps
		if err := writeMapSliceCSV(w, v, ensureASCII); err != nil {
			return "", err
		}
	case reflect.Struct:
		// Handle slice of structs
		if err := writeStructSliceCSV(w, v, ensureASCII); err != nil {
			return "", err
		}
	case reflect.Slice, reflect.Array:
		// Handle slice of slices
		if err := writeSliceSliceCSV(w, v, ensureASCII); err != nil {
			return "", err
		}
	default:
		// Handle slice of primitives as a single column
		if err := writePrimitiveSliceCSV(w, v, ensureASCII); err != nil {
			return "", err
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// writeMapSliceCSV writes a slice of maps to CSV
func writeMapSliceCSV(w *csv.Writer, v reflect.Value, ensureASCII bool) error {
	if v.Len() == 0 {
		return nil
	}

	// Collect all unique keys across all maps
	keySet := make(map[string]bool)
	for i := 0; i < v.Len(); i++ {
		m := v.Index(i)
		for _, key := range m.MapKeys() {
			keySet[fmt.Sprint(key.Interface())] = true
		}
	}

	// Sort keys for consistent column ordering
	keys := make([]string, 0, len(keySet))
	for k := range keySet {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Write header
	if err := w.Write(keys); err != nil {
		return err
	}

	// Write rows
	for i := 0; i < v.Len(); i++ {
		m := v.Index(i)
		row := make([]string, len(keys))
		for j, key := range keys {
			val := m.MapIndex(reflect.ValueOf(key))
			if val.IsValid() {
				row[j] = formatValue(val.Interface(), ensureASCII)
			}
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// writeStructSliceCSV writes a slice of structs to CSV
func writeStructSliceCSV(w *csv.Writer, v reflect.Value, ensureASCII bool) error {
	if v.Len() == 0 {
		return nil
	}

	firstElem := v.Index(0)
	t := firstElem.Type()

	// Collect field names
	var fieldNames []string
	var fieldIndices []int

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}
		fieldNames = append(fieldNames, field.Name)
		fieldIndices = append(fieldIndices, i)
	}

	// Write header
	if err := w.Write(fieldNames); err != nil {
		return err
	}

	// Write rows
	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i)
		row := make([]string, len(fieldIndices))
		for j, idx := range fieldIndices {
			fieldVal := elem.Field(idx)
			row[j] = formatValue(fieldVal.Interface(), ensureASCII)
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// writeSliceSliceCSV writes a slice of slices to CSV
func writeSliceSliceCSV(w *csv.Writer, v reflect.Value, ensureASCII bool) error {
	for i := 0; i < v.Len(); i++ {
		row := v.Index(i)
		rowStrs := make([]string, row.Len())
		for j := 0; j < row.Len(); j++ {
			rowStrs[j] = formatValue(row.Index(j).Interface(), ensureASCII)
		}
		if err := w.Write(rowStrs); err != nil {
			return err
		}
	}
	return nil
}

// writePrimitiveSliceCSV writes a slice of primitives as a single column
func writePrimitiveSliceCSV(w *csv.Writer, v reflect.Value, ensureASCII bool) error {
	for i := 0; i < v.Len(); i++ {
		row := []string{formatValue(v.Index(i).Interface(), ensureASCII)}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}

// formatValue converts a value to its string representation for CSV
func formatValue(v interface{}, ensureASCII bool) string {
	if v == nil {
		return ""
	}

	var result string

	switch val := v.(type) {
	case string:
		result = val
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", val)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", val)
	case float32, float64:
		return strconv.FormatFloat(reflect.ValueOf(val).Float(), 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	case []string:
		// Handle string slices by taking the last (most specific) element
		// For hierarchical types like ["Entity", "ANATOMY"], we want just "ANATOMY"
		if len(val) > 0 {
			result = val[len(val)-1]
		} else {
			result = ""
		}
	case []interface{}:
		// Handle generic slices by taking the last element
		if len(val) > 0 {
			result = formatValue(val[len(val)-1], ensureASCII)
		} else {
			result = ""
		}
	default:
		// Check if it's a slice using reflection
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
			// Take the last (most specific) element for hierarchical types
			if rv.Len() > 0 {
				result = formatValue(rv.Index(rv.Len()-1).Interface(), ensureASCII)
			} else {
				result = ""
			}
		} else {
			// For other complex types, use JSON representation
			b, err := json.Marshal(v)
			if err != nil {
				result = fmt.Sprint(v)
			} else {
				result = string(b)
			}
		}
	}

	if ensureASCII {
		return escapeNonASCII(result)
	}
	return result
}

// escapeNonASCII escapes non-ASCII characters in a string
func escapeNonASCII(s string) string {
	var buf strings.Builder
	for _, r := range s {
		if r > unicode.MaxASCII {
			fmt.Fprintf(&buf, "\\u%04x", r)
		} else {
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

// logPrompts logs system and user prompts at debug level if a logger is available in context.
// This replaces the fmt.Printf statements throughout the prompts package.
// Prints with actual newlines preserved instead of escaped.
// Only prints if the context has "debug_prompts" set to true.
func logPrompts(context map[string]interface{}, sysPrompt, userPrompt string) {
	// Check if debug_prompts is enabled in context
	debugPrompts := false
	if val, ok := context["debug_prompts"]; ok {
		if b, ok := val.(bool); ok {
			debugPrompts = b
		}
	}

	if !debugPrompts {
		return
	}

	if logger, ok := context["logger"].(*slog.Logger); ok && logger != nil {
		// Log with preserved newlines using structured format
		logger.Debug("Generated prompts - System Prompt follows")
		fmt.Println("=== SYSTEM PROMPT ===")
		fmt.Println(sysPrompt)
		logger.Debug("Generated prompts - User Prompt follows")
		fmt.Println("=== USER PROMPT ===")
		fmt.Println(userPrompt)
		fmt.Println("=== END PROMPTS ===")
	}
}