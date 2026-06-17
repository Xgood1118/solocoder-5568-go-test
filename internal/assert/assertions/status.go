package assertions

import (
	"apitester/internal/models"
)

func AssertStatus(response *models.Response, assertion *models.Assertion) *models.AssertionResult {
	result := &models.AssertionResult{
		Type:     "status",
		Operator: assertion.Operator,
		Expected: assertion.Value,
		Actual:   response.StatusCode,
		Message:  assertion.Message,
	}

	if result.Operator == "" {
		result.Operator = "=="
	}

	passed, err := Compare(result.Operator, response.StatusCode, assertion.Value)
	if err != nil {
		result.Passed = false
		result.Error = err.Error()
		return result
	}

	result.Passed = passed
	return result
}
