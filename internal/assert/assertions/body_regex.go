package assertions

import (
	"apitester/internal/models"
	"regexp"
	"strings"
)

func AssertBodyRegex(response *models.Response, assertion *models.Assertion) *models.AssertionResult {
	result := &models.AssertionResult{
		Type:     "body_regex",
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

	pattern := toString(assertion.Value)
	if pattern == "" {
		result.Passed = false
		result.Error = "regex pattern is empty"
		return result
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		result.Passed = false
		result.Error = "invalid regex pattern: " + err.Error()
		return result
	}

	matches := re.FindStringSubmatch(bodyStr)
	if len(matches) > 0 {
		if len(matches) > 1 {
			result.Actual = matches[1]
		} else {
			result.Actual = matches[0]
		}
	} else {
		result.Actual = ""
	}

	if result.Operator == "" {
		result.Operator = "regex"
	}

	operator := strings.ToLower(result.Operator)
	if operator == "regex" || operator == "regexp" || operator == "matches" {
		result.Passed = len(matches) > 0
		return result
	}

	if len(matches) == 0 {
		result.Passed = false
		result.Error = "regex match failed"
		return result
	}

	passed, err := Compare(result.Operator, result.Actual, assertion.Value)
	if err != nil {
		result.Passed = false
		result.Error = err.Error()
		return result
	}

	result.Passed = passed
	return result
}

func ExtractRegex(body string, pattern string) (string, error) {
	if body == "" || pattern == "" {
		return "", nil
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}

	matches := re.FindStringSubmatch(body)
	if len(matches) > 1 {
		return matches[1], nil
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	return "", nil
}
