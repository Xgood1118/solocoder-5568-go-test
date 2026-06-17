package assertions

import (
	"apitester/internal/models"
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

func AssertBodySchema(response *models.Response, assertion *models.Assertion) *models.AssertionResult {
	result := &models.AssertionResult{
		Type:     "body_schema",
		Property: assertion.Property,
		Operator: assertion.Operator,
		Expected: assertion.Value,
		Message:  assertion.Message,
	}

	bodyStr := response.BodyString
	if bodyStr == "" && len(response.Body) > 0 {
		bodyStr = string(response.Body)
	}

	if bodyStr == "" {
		result.Passed = false
		result.Error = "response body is empty"
		return result
	}

	schemaStr, err := toJSONString(assertion.Value)
	if err != nil {
		result.Passed = false
		result.Error = "invalid schema: " + err.Error()
		return result
	}

	result.Expected = schemaStr

	schemaLoader := gojsonschema.NewStringLoader(schemaStr)
	documentLoader := gojsonschema.NewStringLoader(bodyStr)

	validationResult, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		result.Passed = false
		result.Error = "schema validation failed: " + err.Error()
		result.Actual = err.Error()
		return result
	}

	if validationResult.Valid() {
		result.Passed = true
		result.Actual = "valid"
	} else {
		result.Passed = false
		errorMsg := ""
		for i, e := range validationResult.Errors() {
			if i > 0 {
				errorMsg += "; "
			}
			errorMsg += fmt.Sprintf("%s: %s", e.Field(), e.Description())
		}
		result.Error = errorMsg
		result.Actual = errorMsg
	}

	return result
}

func toJSONString(v any) (string, error) {
	switch val := v.(type) {
	case string:
		return val, nil
	case []byte:
		return string(val), nil
	default:
		buf := new(bytes.Buffer)
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(v); err != nil {
			return "", err
		}
		return buf.String(), nil
	}
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}
