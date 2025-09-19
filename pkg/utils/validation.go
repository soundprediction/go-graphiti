package utils

import (
	"fmt"
	"reflect"
)

// EntityTypeValidationError represents an error when validating entity types
type EntityTypeValidationError struct {
	EntityTypeName string
	FieldName      string
}

func (e EntityTypeValidationError) Error() string {
	return fmt.Sprintf("entity type %q has conflicting field %q", e.EntityTypeName, e.FieldName)
}

// ValidateEntityTypes validates that entity types don't conflict with base node fields
func ValidateEntityTypes(entityTypes map[string]interface{}) error {
	if len(entityTypes) == 0 {
		return nil
	}

	// Base entity node fields that should not be overridden
	// These correspond to the fields in the core Node struct
	baseFields := map[string]bool{
		"id":         true,
		"uuid":       true,
		"name":       true,
		"type":       true,
		"group_id":   true,
		"created_at": true,
		"updated_at": true,
		"embedding":  true,
		"metadata":   true,
		"valid_from": true,
		"valid_to":   true,
		"source_ids": true,
		// Entity-specific fields
		"entity_type": true,
		"summary":     true,
		// Episode-specific fields
		"episode_type": true,
		"content":      true,
		"reference":    true,
		// Community-specific fields
		"level": true,
	}

	for entityTypeName, entityTypeModel := range entityTypes {
		// Use reflection to get struct fields
		v := reflect.ValueOf(entityTypeModel)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		if v.Kind() != reflect.Struct {
			continue // Skip non-struct types
		}

		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			fieldName := field.Name

			// Check JSON tag if available
			if jsonTag := field.Tag.Get("json"); jsonTag != "" {
				// Parse JSON tag (field name is before the first comma)
				if commaIdx := len(jsonTag); commaIdx > 0 {
					for j, c := range jsonTag {
						if c == ',' {
							commaIdx = j
							break
						}
					}
					if commaIdx > 0 {
						fieldName = jsonTag[:commaIdx]
					}
				}
			}

			// Convert to lowercase for comparison
			fieldNameLower := fieldName
			if fieldNameLower != "-" && baseFields[fieldNameLower] {
				return EntityTypeValidationError{
					EntityTypeName: entityTypeName,
					FieldName:      fieldName,
				}
			}
		}
	}

	return nil
}

// ValidateStringSlice validates that all strings in a slice meet certain criteria
func ValidateStringSlice(slice []string, validator func(string) error) error {
	for _, item := range slice {
		if err := validator(item); err != nil {
			return err
		}
	}
	return nil
}

// ValidateNonEmpty validates that a string is not empty
func ValidateNonEmpty(value string) error {
	if value == "" {
		return fmt.Errorf("value cannot be empty")
	}
	return nil
}

// ValidateUUID validates that a string looks like a UUID
func ValidateUUID(uuid string) error {
	if len(uuid) != 36 {
		return fmt.Errorf("invalid UUID length: expected 36, got %d", len(uuid))
	}

	// Basic UUID format check: 8-4-4-4-12 characters separated by hyphens
	if uuid[8] != '-' || uuid[13] != '-' || uuid[18] != '-' || uuid[23] != '-' {
		return fmt.Errorf("invalid UUID format: %s", uuid)
	}

	return nil
}

// ValidateRequired validates that all required fields are present and non-empty
func ValidateRequired(fields map[string]string) error {
	for fieldName, value := range fields {
		if value == "" {
			return fmt.Errorf("required field %q is missing or empty", fieldName)
		}
	}
	return nil
}

// ValidateRange validates that a numeric value is within a specified range
func ValidateRange(value, min, max float64) error {
	if value < min || value > max {
		return fmt.Errorf("value %f is outside valid range [%f, %f]", value, min, max)
	}
	return nil
}

// ValidateSliceLength validates that a slice has the expected length
func ValidateSliceLength[T any](slice []T, expectedLength int) error {
	if len(slice) != expectedLength {
		return fmt.Errorf("slice length %d does not match expected length %d", len(slice), expectedLength)
	}
	return nil
}

// ValidateEmbeddingDimensions validates that embeddings have consistent dimensions
func ValidateEmbeddingDimensions(embeddings [][]float32, expectedDim int) error {
	for i, embedding := range embeddings {
		if len(embedding) != expectedDim {
			return fmt.Errorf("embedding %d has dimension %d, expected %d", i, len(embedding), expectedDim)
		}
	}
	return nil
}