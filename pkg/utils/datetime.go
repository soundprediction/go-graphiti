package utils

import (
	"fmt"
	"reflect"
	"time"
)

// UTCNow returns the current UTC datetime with timezone information
func UTCNow() time.Time {
	return time.Now().UTC()
}

// EnsureUTC ensures a datetime is timezone-aware and in UTC.
// If the datetime is naive (no timezone), assumes it's in UTC.
// If the datetime has a different timezone, converts it to UTC.
// Returns nil if input is nil.
func EnsureUTC(dt *time.Time) *time.Time {
	if dt == nil {
		return nil
	}

	if dt.Location() == time.UTC {
		return dt
	}

	// Convert to UTC
	utc := dt.UTC()
	return &utc
}

// ConvertDatetimesToStrings recursively converts all time.Time values in a data structure to ISO format strings
func ConvertDatetimesToStrings(obj interface{}) interface{} {
	if obj == nil {
		return nil
	}

	v := reflect.ValueOf(obj)

	switch v.Kind() {
	case reflect.Map:
		result := make(map[string]interface{})
		for _, key := range v.MapKeys() {
			keyStr := fmt.Sprintf("%v", key.Interface())
			result[keyStr] = ConvertDatetimesToStrings(v.MapIndex(key).Interface())
		}
		return result

	case reflect.Slice, reflect.Array:
		result := make([]interface{}, v.Len())
		for i := 0; i < v.Len(); i++ {
			result[i] = ConvertDatetimesToStrings(v.Index(i).Interface())
		}
		return result

	case reflect.Struct:
		if t, ok := obj.(time.Time); ok {
			return t.Format(time.RFC3339)
		}

		// For other structs, convert to map and then process
		result := make(map[string]interface{})
		typ := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := typ.Field(i)
			if field.IsExported() {
				fieldValue := v.Field(i).Interface()
				result[field.Name] = ConvertDatetimesToStrings(fieldValue)
			}
		}
		return result

	case reflect.Ptr:
		if v.IsNil() {
			return nil
		}
		if t, ok := obj.(*time.Time); ok {
			return t.Format(time.RFC3339)
		}
		return ConvertDatetimesToStrings(v.Elem().Interface())

	default:
		return obj
	}
}

// FormatTimeForDB formats a time for database storage
func FormatTimeForDB(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// ParseTimeFromDB parses a time from database string format
func ParseTimeFromDB(timeStr string) (time.Time, error) {
	return time.Parse(time.RFC3339, timeStr)
}

// TimeToMilliseconds converts a time to milliseconds since Unix epoch
func TimeToMilliseconds(t time.Time) int64 {
	return t.UnixMilli()
}

// MillisecondsToTime converts milliseconds since Unix epoch to time
func MillisecondsToTime(ms int64) time.Time {
	return time.UnixMilli(ms)
}
