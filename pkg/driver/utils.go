package driver

import (
	"fmt"
	"reflect"
)

// convertNodeToMap converts a graph database node to a map of properties.
// It handles various node types by using reflection to extract properties
// from either a Props()/Properties() method or direct field access.
func convertNodeToMap(nodeInterface interface{}) (map[string]interface{}, error) {
	// Check for nil input
	if nodeInterface == nil {
		return nil, fmt.Errorf("node interface is nil")
	}

	result := make(map[string]interface{})

	// Use reflection to access the node's properties
	nodeValue := reflect.ValueOf(nodeInterface)

	// Check if the reflect.Value is valid (not zero)
	if !nodeValue.IsValid() {
		return nil, fmt.Errorf("invalid node value")
	}

	// Handle pointer types
	if nodeValue.Kind() == reflect.Ptr {
		nodeValue = nodeValue.Elem()
	}

	// Try to get Props or Properties method
	propsMethod := nodeValue.MethodByName("Props")
	if !propsMethod.IsValid() {
		propsMethod = nodeValue.MethodByName("Properties")
	}

	if propsMethod.IsValid() {
		// Call Props() or Properties()
		results := propsMethod.Call(nil)
		if len(results) > 0 {
			if props, ok := results[0].Interface().(map[string]interface{}); ok {
				// Copy all properties to result
				for k, v := range props {
					result[k] = v
				}
			}
		}
	} else {
		// Try to access fields directly
		nodeType := nodeValue.Type()
		for i := 0; i < nodeValue.NumField(); i++ {
			field := nodeType.Field(i)
			fieldValue := nodeValue.Field(i)

			// Look for a field that contains properties
			if field.Name == "Props" || field.Name == "Properties" {
				if fieldValue.Kind() == reflect.Map {
					for _, key := range fieldValue.MapKeys() {
						result[key.String()] = fieldValue.MapIndex(key).Interface()
					}
				}
			}
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no properties found in node")
	}

	return result, nil
}
