package ai

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"reflect"
	"regexp"
	"time"
)

//go:embed schema.json
var schemaFS embed.FS

var analysisSchema map[string]any

func init() {
	raw, err := schemaFS.ReadFile("schema.json")
	if err != nil || json.Unmarshal(raw, &analysisSchema) != nil {
		panic("invalid embedded AI analysis schema")
	}
}

// ValidateJSONSchema validates provider output against the embedded public
// contract. Validate performs additional domain and grounding-independent checks.
func ValidateJSONSchema(raw []byte) error {
	var value any
	decoder := json.NewDecoder(bytesReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("AI output does not satisfy JSON Schema: %w", err)
	}
	if err := validateNode(value, analysisSchema, "$"); err != nil {
		return fmt.Errorf("AI output does not satisfy JSON Schema: %w", err)
	}
	return nil
}

func validateNode(value any, schema map[string]any, path string) error {
	if ref, _ := schema["$ref"].(string); ref != "" {
		if ref != "#/$defs/boundedStrings" {
			return fmt.Errorf("%s: unsupported schema reference", path)
		}
		defs, _ := analysisSchema["$defs"].(map[string]any)
		target, _ := defs["boundedStrings"].(map[string]any)
		return validateNode(value, target, path)
	}
	if enum, ok := schema["enum"].([]any); ok {
		for _, candidate := range enum {
			if reflect.DeepEqual(value, candidate) {
				return nil
			}
		}
		return fmt.Errorf("%s: value is not in enum", path)
	}
	typeName, _ := schema["type"].(string)
	switch typeName {
	case "object":
		object, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("%s: expected object", path)
		}
		properties, _ := schema["properties"].(map[string]any)
		if required, ok := schema["required"].([]any); ok {
			for _, name := range required {
				if _, exists := object[name.(string)]; !exists {
					return fmt.Errorf("%s.%s: required", path, name)
				}
			}
		}
		if additional, exists := schema["additionalProperties"]; exists && additional == false {
			for name := range object {
				if _, allowed := properties[name]; !allowed {
					return fmt.Errorf("%s.%s: unknown property", path, name)
				}
			}
		}
		for name, child := range object {
			if childSchema, ok := properties[name].(map[string]any); ok {
				if err := validateNode(child, childSchema, path+"."+name); err != nil {
					return err
				}
			}
		}
	case "array":
		array, ok := value.([]any)
		if !ok {
			return fmt.Errorf("%s: expected array", path)
		}
		if err := checkCount(len(array), schema, path, "Items"); err != nil {
			return err
		}
		if unique, _ := schema["uniqueItems"].(bool); unique {
			seen := map[string]struct{}{}
			for _, item := range array {
				encoded, _ := json.Marshal(item)
				key := string(encoded)
				if _, duplicate := seen[key]; duplicate {
					return fmt.Errorf("%s: duplicate item", path)
				}
				seen[key] = struct{}{}
			}
		}
		if itemSchema, ok := schema["items"].(map[string]any); ok {
			for index, item := range array {
				if err := validateNode(item, itemSchema, fmt.Sprintf("%s[%d]", path, index)); err != nil {
					return err
				}
			}
		}
	case "string":
		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("%s: expected string", path)
		}
		if err := checkCount(len([]rune(text)), schema, path, "Length"); err != nil {
			return err
		}
		if pattern, _ := schema["pattern"].(string); pattern != "" {
			matched, _ := regexp.MatchString(pattern, text)
			if !matched {
				return fmt.Errorf("%s: pattern mismatch", path)
			}
		}
		if format, _ := schema["format"].(string); format == "uri" {
			parsed, err := url.ParseRequestURI(text)
			if err != nil || parsed.Scheme == "" || parsed.Host == "" {
				return fmt.Errorf("%s: invalid URI", path)
			}
		} else if format == "date-time" {
			if _, err := time.Parse(time.RFC3339, text); err != nil {
				return fmt.Errorf("%s: invalid date-time", path)
			}
		}
	case "integer":
		number, ok := value.(json.Number)
		if !ok {
			return fmt.Errorf("%s: expected integer", path)
		}
		integer, err := number.Int64()
		if err != nil {
			return fmt.Errorf("%s: expected integer", path)
		}
		if minimum, ok := numberValue(schema["minimum"]); ok && float64(integer) < minimum {
			return fmt.Errorf("%s: below minimum", path)
		}
		if maximum, ok := numberValue(schema["maximum"]); ok && float64(integer) > maximum {
			return fmt.Errorf("%s: above maximum", path)
		}
	}
	return nil
}

func checkCount(count int, schema map[string]any, path, suffix string) error {
	if minimum, ok := numberValue(schema["min"+suffix]); ok && float64(count) < minimum {
		return fmt.Errorf("%s: too short", path)
	}
	if maximum, ok := numberValue(schema["max"+suffix]); ok && float64(count) > maximum {
		return fmt.Errorf("%s: too long", path)
	}
	return nil
}

func numberValue(value any) (float64, bool) {
	switch number := value.(type) {
	case float64:
		return number, true
	case json.Number:
		parsed, err := number.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

type byteReader []byte

func (reader *byteReader) Read(target []byte) (int, error) {
	if len(*reader) == 0 {
		return 0, io.EOF
	}
	count := copy(target, *reader)
	*reader = (*reader)[count:]
	return count, nil
}

func bytesReader(raw []byte) *byteReader { reader := byteReader(raw); return &reader }
