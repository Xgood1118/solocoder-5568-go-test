package openapi

import (
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

func GenerateExampleFromSchema(schema *openapi3.Schema) any {
	if schema == nil {
		return nil
	}

	if schema.Example != nil {
		return schema.Example
	}

	if schema.Default != nil {
		return schema.Default
	}

	if len(schema.Enum) > 0 {
		return schema.Enum[0]
	}

	switch {
	case schema.Type != nil && schema.Type.Is("string"):
		return generateStringExample(schema)
	case schema.Type != nil && schema.Type.Is("number"):
		return generateNumberExample(schema)
	case schema.Type != nil && schema.Type.Is("integer"):
		return generateIntegerExample(schema)
	case schema.Type != nil && schema.Type.Is("boolean"):
		return true
	case schema.Type != nil && schema.Type.Is("array"):
		return generateArrayExample(schema)
	case schema.Type != nil && schema.Type.Is("object"):
		return generateObjectExample(schema)
	default:
		if (schema.Type == nil || schema.Type.IsEmpty()) && len(schema.Properties) > 0 {
			return generateObjectExample(schema)
		}
		return nil
	}
}

func generateStringExample(schema *openapi3.Schema) string {
	switch schema.Format {
	case "date":
		return time.Now().Format("2006-01-02")
	case "date-time":
		return time.Now().UTC().Format(time.RFC3339)
	case "email":
		return "user@example.com"
	case "password":
		return "password123"
	case "uuid":
		return generateUUID()
	case "uri":
		return "https://example.com/resource"
	case "hostname":
		return "example.com"
	case "ipv4":
		return generateRandomIPv4()
	case "ipv6":
		return generateRandomIPv6()
	case "byte":
		return "SGVsbG8gV29ybGQ="
	case "binary":
		return "binary_data"
	default:
		if schema.Pattern != "" {
			return generateStringFromPattern(schema.Pattern)
		}
		minLen := 0
		if schema.MinLength > 0 {
			minLen = int(schema.MinLength)
		}
		maxLen := 10
		if schema.MaxLength != nil {
			maxLen = int(*schema.MaxLength)
		}
		return generateRandomString(minLen, maxLen)
	}
}

func generateNumberExample(schema *openapi3.Schema) float64 {
	min := 0.0
	if schema.Min != nil {
		min = *schema.Min
		if schema.ExclusiveMin.IsTrue() {
			min += 0.1
		}
	}
	max := 100.0
	if schema.Max != nil {
		max = *schema.Max
		if schema.ExclusiveMax.IsTrue() {
			max -= 0.1
		}
	}
	if schema.MultipleOf != nil {
		multiple := *schema.MultipleOf
		return mathRound(min/multiple) * multiple
	}
	return min + rand.Float64()*(max-min)
}

func generateIntegerExample(schema *openapi3.Schema) int64 {
	min := int64(0)
	if schema.Min != nil {
		min = int64(*schema.Min)
		if schema.ExclusiveMin.IsTrue() {
			min++
		}
	}
	max := int64(100)
	if schema.Max != nil {
		max = int64(*schema.Max)
		if schema.ExclusiveMax.IsTrue() {
			max--
		}
	}
	if schema.MultipleOf != nil {
		multiple := int64(*schema.MultipleOf)
		return (min/multiple + 1) * multiple
	}
	return min + rand.Int63n(max-min+1)
}

func generateArrayExample(schema *openapi3.Schema) []any {
	if schema.Items == nil || schema.Items.Value == nil {
		return []any{}
	}

	minItems := 1
	if schema.MinItems > 0 {
		minItems = int(schema.MinItems)
	}
	maxItems := 3
	if schema.MaxItems != nil {
		maxItems = int(*schema.MaxItems)
	}
	if minItems < 1 {
		minItems = 1
	}
	if maxItems < minItems {
		maxItems = minItems
	}

	count := minItems + rand.Intn(maxItems-minItems+1)
	result := make([]any, count)
	for i := 0; i < count; i++ {
		result[i] = GenerateExampleFromSchema(schema.Items.Value)
	}
	return result
}

func generateObjectExample(schema *openapi3.Schema) map[string]any {
	result := make(map[string]any)

	for name, propRef := range schema.Properties {
		if propRef != nil && propRef.Value != nil {
			result[name] = GenerateExampleFromSchema(propRef.Value)
		}
	}

	for _, required := range schema.Required {
		if _, exists := result[required]; !exists {
			if propRef, ok := schema.Properties[required]; ok && propRef != nil && propRef.Value != nil {
				result[required] = GenerateExampleFromSchema(propRef.Value)
			}
		}
	}

	if len(schema.Properties) == 0 && schema.AdditionalProperties.Schema != nil {
		if schema.AdditionalProperties.Schema.Value != nil {
			result["key1"] = GenerateExampleFromSchema(schema.AdditionalProperties.Schema.Value)
			result["key2"] = GenerateExampleFromSchema(schema.AdditionalProperties.Schema.Value)
		}
	}

	return result
}

func generateRandomIPv4() string {
	return fmt.Sprintf("%d.%d.%d.%d",
		rand.Intn(223)+1,
		rand.Intn(256),
		rand.Intn(256),
		rand.Intn(256))
}

func generateRandomIPv6() string {
	ip := make(net.IP, 16)
	for i := range ip {
		ip[i] = byte(rand.Intn(256))
	}
	return ip.String()
}

func generateRandomString(minLen, maxLen int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	length := minLen + rand.Intn(maxLen-minLen+1)
	if length < 1 {
		length = 5
	}
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func generateStringFromPattern(pattern string) string {
	pattern = strings.ReplaceAll(pattern, "^", "")
	pattern = strings.ReplaceAll(pattern, "$", "")
	pattern = strings.ReplaceAll(pattern, "\\d", "1")
	pattern = strings.ReplaceAll(pattern, "\\w", "a")
	pattern = strings.ReplaceAll(pattern, "\\s", " ")
	pattern = strings.ReplaceAll(pattern, ".", "x")
	pattern = strings.ReplaceAll(pattern, "*", "")
	pattern = strings.ReplaceAll(pattern, "+", "")
	pattern = strings.ReplaceAll(pattern, "?", "")
	pattern = strings.ReplaceAll(pattern, "[", "")
	pattern = strings.ReplaceAll(pattern, "]", "")
	pattern = strings.ReplaceAll(pattern, "(", "")
	pattern = strings.ReplaceAll(pattern, ")", "")
	pattern = strings.ReplaceAll(pattern, "|", "")
	pattern = strings.ReplaceAll(pattern, "{", "")
	pattern = strings.ReplaceAll(pattern, "}", "")
	return pattern
}

func mathRound(f float64) float64 {
	if f >= 0 {
		return float64(int64(f + 0.5))
	}
	return float64(int64(f - 0.5))
}

func generateUUID() string {
	b := make([]byte, 16)
	for i := range b {
		b[i] = byte(rand.Intn(256))
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
