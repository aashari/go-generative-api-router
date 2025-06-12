package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// SensitiveDataMasker handles masking of sensitive information in logs
type SensitiveDataMasker struct {
	patterns []SensitivePattern
}

// SensitivePattern defines a pattern for detecting and masking sensitive data
type SensitivePattern struct {
	Name        string
	Regex       *regexp.Regexp
	Replacement string
	FieldNames  []string // Field names that should be masked
}

// NewSensitiveDataMasker creates a new data masker with default patterns
func NewSensitiveDataMasker() *SensitiveDataMasker {
	patterns := []SensitivePattern{
		{
			Name:        "OpenAI API Key",
			Regex:       regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`),
			Replacement: "sk-***MASKED***",
		},
		{
			Name:        "Anthropic API Key",
			Regex:       regexp.MustCompile(`sk-ant-[a-zA-Z0-9_-]{20,}`),
			Replacement: "sk-ant-***MASKED***",
		},
		{
			Name:        "Generic API Key",
			Regex:       regexp.MustCompile(`[a-zA-Z0-9]{32,}`),
			Replacement: "***MASKED_API_KEY***",
		},
		{
			Name:        "Bearer Token",
			Regex:       regexp.MustCompile(`Bearer\s+[a-zA-Z0-9._-]+`),
			Replacement: "Bearer ***MASKED***",
		},
		{
			Name:        "Authorization Header",
			Regex:       regexp.MustCompile(`(?i)authorization:\s*[^\s]+`),
			Replacement: "Authorization: ***MASKED***",
		},
		{
			Name: "Sensitive Fields",
			FieldNames: []string{
				"api_key", "apikey", "api-key",
				"secret", "password", "token",
				"authorization", "auth",
				"credential", "key", "value",
			},
		},
	}

	return &SensitiveDataMasker{
		patterns: patterns,
	}
}

// MaskSensitiveData masks sensitive information in any data structure
func (m *SensitiveDataMasker) MaskSensitiveData(data interface{}) interface{} {
	return m.maskValue(reflect.ValueOf(data)).Interface()
}

// maskValue recursively masks sensitive data in reflect.Value
func (m *SensitiveDataMasker) maskValue(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}

	switch v.Kind() {
	case reflect.String:
		return reflect.ValueOf(m.maskString(v.String()))

	case reflect.Map:
		return m.maskMap(v)

	case reflect.Slice, reflect.Array:
		return m.maskSlice(v)

	case reflect.Struct:
		return m.maskStruct(v)

	case reflect.Ptr:
		if v.IsNil() {
			return v
		}
		elem := m.maskValue(v.Elem())
		newPtr := reflect.New(elem.Type())
		newPtr.Elem().Set(elem)
		return newPtr

	case reflect.Interface:
		if v.IsNil() {
			return v
		}
		return reflect.ValueOf(m.MaskSensitiveData(v.Interface()))

	default:
		return v
	}
}

// maskString applies regex patterns to mask sensitive strings
func (m *SensitiveDataMasker) maskString(s string) string {
	masked := s
	for _, pattern := range m.patterns {
		if pattern.Regex != nil {
			masked = pattern.Regex.ReplaceAllString(masked, pattern.Replacement)
		}
	}
	return masked
}

// maskMap masks sensitive data in maps
func (m *SensitiveDataMasker) maskMap(v reflect.Value) reflect.Value {
	if v.IsNil() {
		return v
	}

	newMap := reflect.MakeMap(v.Type())
	for _, key := range v.MapKeys() {
		keyStr := fmt.Sprintf("%v", key.Interface())
		value := v.MapIndex(key)

		// Check if key indicates sensitive data
		if m.isSensitiveField(keyStr) {
			newMap.SetMapIndex(key, reflect.ValueOf("***MASKED***"))
		} else {
			maskedValue := m.maskValue(value)
			newMap.SetMapIndex(key, maskedValue)
		}
	}
	return newMap
}

// maskSlice masks sensitive data in slices and arrays
func (m *SensitiveDataMasker) maskSlice(v reflect.Value) reflect.Value {
	newSlice := reflect.MakeSlice(v.Type(), v.Len(), v.Cap())
	for i := 0; i < v.Len(); i++ {
		maskedValue := m.maskValue(v.Index(i))
		newSlice.Index(i).Set(maskedValue)
	}
	return newSlice
}

// maskStruct masks sensitive data in structs
func (m *SensitiveDataMasker) maskStruct(v reflect.Value) reflect.Value {
	newStruct := reflect.New(v.Type()).Elem()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := v.Type().Field(i)

		if !field.CanInterface() {
			continue
		}

		// Check if field name indicates sensitive data
		if m.isSensitiveField(fieldType.Name) || m.isSensitiveField(fieldType.Tag.Get("json")) {
			if newStruct.Field(i).CanSet() {
				newStruct.Field(i).Set(reflect.ValueOf("***MASKED***"))
			}
		} else {
			maskedValue := m.maskValue(field)
			if newStruct.Field(i).CanSet() && maskedValue.Type().AssignableTo(fieldType.Type) {
				newStruct.Field(i).Set(maskedValue)
			}
		}
	}
	return newStruct
}

// isSensitiveField checks if a field name indicates sensitive data
func (m *SensitiveDataMasker) isSensitiveField(fieldName string) bool {
	fieldLower := strings.ToLower(fieldName)

	for _, pattern := range m.patterns {
		for _, sensitiveField := range pattern.FieldNames {
			if strings.Contains(fieldLower, strings.ToLower(sensitiveField)) {
				return true
			}
		}
	}
	return false
}

// MaskJSON masks sensitive data in JSON strings
func (m *SensitiveDataMasker) MaskJSON(jsonStr string) string {
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		// If not valid JSON, apply string masking
		return m.maskString(jsonStr)
	}

	maskedData := m.MaskSensitiveData(data)
	maskedJSON, err := json.Marshal(maskedData)
	if err != nil {
		return m.maskString(jsonStr)
	}

	return string(maskedJSON)
}

// MaskHeaders masks sensitive headers (like Authorization)
func (m *SensitiveDataMasker) MaskHeaders(headers map[string][]string) map[string][]string {
	if headers == nil {
		return nil
	}

	maskedHeaders := make(map[string][]string)
	for key, values := range headers {
		keyLower := strings.ToLower(key)

		if m.isSensitiveField(keyLower) {
			maskedHeaders[key] = []string{"***MASKED***"}
		} else {
			maskedValues := make([]string, len(values))
			for i, value := range values {
				maskedValues[i] = m.maskString(value)
			}
			maskedHeaders[key] = maskedValues
		}
	}
	return maskedHeaders
}

// MaskCredentials specifically masks credential structures
func (m *SensitiveDataMasker) MaskCredentials(creds interface{}) interface{} {
	// For credential arrays/slices, mask the Value field specifically
	v := reflect.ValueOf(creds)
	if v.Kind() == reflect.Slice {
		newSlice := reflect.MakeSlice(v.Type(), v.Len(), v.Cap())
		for i := 0; i < v.Len(); i++ {
			item := v.Index(i)
			if item.Kind() == reflect.Struct {
				newItem := reflect.New(item.Type()).Elem()
				for j := 0; j < item.NumField(); j++ {
					field := item.Field(j)
					fieldType := item.Type().Field(j)

					if strings.ToLower(fieldType.Name) == "value" {
						// Mask the credential value
						if newItem.Field(j).CanSet() {
							newItem.Field(j).Set(reflect.ValueOf("***MASKED***"))
						}
					} else {
						if newItem.Field(j).CanSet() && field.CanInterface() {
							newItem.Field(j).Set(field)
						}
					}
				}
				newSlice.Index(i).Set(newItem)
			} else {
				newSlice.Index(i).Set(item)
			}
		}
		return newSlice.Interface()
	}

	return m.MaskSensitiveData(creds)
}

// GetMaskedString returns a masked version of any value as a string
func (m *SensitiveDataMasker) GetMaskedString(value interface{}) string {
	masked := m.MaskSensitiveData(value)
	return fmt.Sprintf("%v", masked)
}

// TruncateBase64InData truncates base64 strings in data URLs for logging
func TruncateBase64InData(data interface{}) interface{} {
	return truncateBase64Value(reflect.ValueOf(data)).Interface()
}

// truncateBase64Value recursively truncates base64 strings in reflect.Value
func truncateBase64Value(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}

	switch v.Kind() {
	case reflect.String:
		return reflect.ValueOf(truncateBase64String(v.String()))

	case reflect.Map:
		return truncateBase64Map(v)

	case reflect.Slice, reflect.Array:
		return truncateBase64Slice(v)

	case reflect.Struct:
		return truncateBase64Struct(v)

	case reflect.Ptr:
		if v.IsNil() {
			return v
		}
		elem := truncateBase64Value(v.Elem())
		newPtr := reflect.New(elem.Type())
		newPtr.Elem().Set(elem)
		return newPtr

	case reflect.Interface:
		if v.IsNil() {
			return v
		}
		return reflect.ValueOf(TruncateBase64InData(v.Interface()))

	default:
		return v
	}
}

// truncateBase64String truncates base64 data in data URLs and JSON content
func truncateBase64String(s string) string {
	// First, handle embedded data URLs anywhere inside the string (even if string doesn't start with "data:")
	// This regex captures the prefix (e.g., "data:image/png;base64,") and the base64 payload separately so we can truncate only the payload.
	dataURLRegex := regexp.MustCompile(`(?i)(data:[^;]+;base64,)([A-Za-z0-9+/]{100,}={0,2})`)
	s = dataURLRegex.ReplaceAllStringFunc(s, func(match string) string {
		submatches := dataURLRegex.FindStringSubmatch(match)
		if len(submatches) != 3 {
			return match // should not happen, but safeguard
		}
		prefix := submatches[1]
		payload := submatches[2]
		if len(payload) <= 100 {
			return match
		}
		truncated := payload[:50] + "...[" + fmt.Sprintf("%d chars truncated", len(payload)-100) + "]..." + payload[len(payload)-50:]
		return prefix + truncated
	})

	// Handle plain data URL if the entire string is itself a data URL (backwards compatibility)
	if strings.HasPrefix(s, "data:") && strings.Contains(s, ";base64,") {
		parts := strings.SplitN(s, ";base64,", 2)
		if len(parts) == 2 {
			base64Data := parts[1]
			if len(base64Data) > 100 {
				truncated := base64Data[:50] + "...[" + fmt.Sprintf("%d chars truncated", len(base64Data)-100) + "]..." + base64Data[len(base64Data)-50:]
				return parts[0] + ";base64," + truncated
			}
		}
	}

	// Handle base64 strings in JSON content using regex
	// This regex matches base64 strings that are:
	// 1. At least 100 characters long
	// 2. Contain only valid base64 characters (A-Z, a-z, 0-9, +, /, =)
	// 3. Are properly padded (ending with 0-2 '=' characters)
	// 4. Are within JSON string values (between quotes)
	base64Regex := regexp.MustCompile(`"([A-Za-z0-9+/]{100,}={0,2})"`)
	
	return base64Regex.ReplaceAllStringFunc(s, func(match string) string {
		// Remove the surrounding quotes to get just the base64 string
		base64Data := match[1 : len(match)-1]
		
		// Only truncate if it's longer than 100 characters
		if len(base64Data) > 100 {
			// Truncate to first 50 and last 50 characters with indicator
			truncated := base64Data[:50] + "...[" + fmt.Sprintf("%d chars truncated", len(base64Data)-100) + "]..." + base64Data[len(base64Data)-50:]
			return `"` + truncated + `"`
		}
		
		return match
	})
}

// truncateBase64Map truncates base64 data in maps
func truncateBase64Map(v reflect.Value) reflect.Value {
	if v.IsNil() {
		return v
	}

	newMap := reflect.MakeMap(v.Type())
	for _, key := range v.MapKeys() {
		value := v.MapIndex(key)
		truncatedValue := truncateBase64Value(value)
		newMap.SetMapIndex(key, truncatedValue)
	}
	return newMap
}

// truncateBase64Slice truncates base64 data in slices and arrays
func truncateBase64Slice(v reflect.Value) reflect.Value {
	newSlice := reflect.MakeSlice(v.Type(), v.Len(), v.Cap())
	for i := 0; i < v.Len(); i++ {
		truncatedValue := truncateBase64Value(v.Index(i))
		newSlice.Index(i).Set(truncatedValue)
	}
	return newSlice
}

// truncateBase64Struct truncates base64 data in structs
func truncateBase64Struct(v reflect.Value) reflect.Value {
	newStruct := reflect.New(v.Type()).Elem()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := v.Type().Field(i)

		if !field.CanInterface() {
			continue
		}

		truncatedValue := truncateBase64Value(field)
		if newStruct.Field(i).CanSet() && truncatedValue.Type().AssignableTo(fieldType.Type) {
			newStruct.Field(i).Set(truncatedValue)
		}
	}
	return newStruct
}
