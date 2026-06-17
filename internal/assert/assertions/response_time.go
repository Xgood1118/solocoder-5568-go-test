package assertions

import (
	"apitester/internal/models"
	"strconv"
	"time"
)

func AssertResponseTime(response *models.Response, assertion *models.Assertion) *models.AssertionResult {
	result := &models.AssertionResult{
		Type:     "response_time",
		Operator: assertion.Operator,
		Expected: assertion.Value,
		Message:  assertion.Message,
	}

	latencyMs := float64(response.Latency) / float64(time.Millisecond)
	result.Actual = latencyMs

	if result.Operator == "" {
		result.Operator = "<="
	}

	expectedMs, err := parseExpectedFloat(assertion.Value)
	if err != nil {
		result.Passed = false
		result.Error = "invalid expected value: " + err.Error()
		return result
	}

	result.Expected = expectedMs

	passed, err := Compare(result.Operator, latencyMs, expectedMs)
	if err != nil {
		result.Passed = false
		result.Error = err.Error()
		return result
	}

	result.Passed = passed
	return result
}

func parseExpectedFloat(v any) (float64, error) {
	switch val := v.(type) {
	case int:
		return float64(val), nil
	case int8:
		return float64(val), nil
	case int16:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case uint:
		return float64(val), nil
	case uint8:
		return float64(val), nil
	case uint16:
		return float64(val), nil
	case uint32:
		return float64(val), nil
	case uint64:
		return float64(val), nil
	case float32:
		return float64(val), nil
	case float64:
		return val, nil
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, err
		}
		return f, nil
	default:
		return 0, nil
	}
}
