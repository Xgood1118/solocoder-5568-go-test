package assertions

import (
	"apitester/internal/models"
	"encoding/json"
	"strings"

	"github.com/oliveagle/jsonpath"
)

func AssertBodyJSONPath(response *models.Response, assertion *models.Assertion) *models.AssertionResult {
	result := &models.AssertionResult{
		Type:     "body_jsonpath",
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

	var data any
	if err := json.Unmarshal([]byte(bodyStr), &data); err != nil {
		result.Passed = false
		result.Error = "invalid JSON: " + err.Error()
		return result
	}

	expr := assertion.Property
	if !strings.HasPrefix(expr, "$.") && !strings.HasPrefix(expr, "$[") {
		expr = "$." + expr
	}

	res, err := jsonpath.JsonPathLookup(data, expr)
	if err != nil {
		result.Passed = false
		result.Error = "JSONPath lookup failed: " + err.Error()
		return result
	}

	result.Actual = res

	operator := strings.ToLower(assertion.Operator)

	switch operator {
	case "exists", "present":
		result.Passed = res != nil
		return result
	case "not_exists", "absent":
		result.Passed = res == nil
		return result
	}

	if result.Operator == "" {
		result.Operator = "=="
	}

	passed, err := Compare(result.Operator, res, assertion.Value)
	if err != nil {
		result.Passed = false
		result.Error = err.Error()
		return result
	}

	result.Passed = passed
	return result
}

func ExtractJSONPath(body string, expr string) (string, error) {
	if body == "" {
		return "", nil
	}

	var data any
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return "", err
	}

	if !strings.HasPrefix(expr, "$.") && !strings.HasPrefix(expr, "$[") {
		expr = "$." + expr
	}

	res, err := jsonpath.JsonPathLookup(data, expr)
	if err != nil {
		return "", err
	}

	if res == nil {
		return "", nil
	}

	switch v := res.(type) {
	case string:
		return v, nil
	default:
		b, _ := json.Marshal(v)
		return string(b), nil
	}
}
