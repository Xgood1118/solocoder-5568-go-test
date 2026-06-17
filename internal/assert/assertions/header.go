package assertions

import (
	"apitester/internal/models"
	"strings"
)

func AssertHeader(response *models.Response, assertion *models.Assertion) *models.AssertionResult {
	result := &models.AssertionResult{
		Type:     "header",
		Property: assertion.Property,
		Operator: assertion.Operator,
		Expected: assertion.Value,
		Message:  assertion.Message,
	}

	headerName := assertion.Property
	actualValue := ""

	for k, v := range response.Headers {
		if strings.EqualFold(k, headerName) {
			actualValue = v
			break
		}
	}

	result.Actual = actualValue

	operator := strings.ToLower(assertion.Operator)

	switch operator {
	case "exists", "present":
		result.Passed = actualValue != ""
		return result
	case "not_exists", "absent":
		result.Passed = actualValue == ""
		return result
	}

	if actualValue == "" {
		result.Passed = false
		result.Error = "header not found"
		return result
	}

	if result.Operator == "" {
		result.Operator = "=="
	}

	passed, err := Compare(result.Operator, actualValue, assertion.Value)
	if err != nil {
		result.Passed = false
		result.Error = err.Error()
		return result
	}

	result.Passed = passed
	return result
}
