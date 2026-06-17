package assertions

import (
	"apitester/internal/models"
	"strings"

	"github.com/antchfx/xmlquery"
)

func AssertBodyXPath(response *models.Response, assertion *models.Assertion) *models.AssertionResult {
	result := &models.AssertionResult{
		Type:     "body_xpath",
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

	doc, err := xmlquery.Parse(strings.NewReader(bodyStr))
	if err != nil {
		result.Passed = false
		result.Error = "invalid XML: " + err.Error()
		return result
	}

	expr := assertion.Property
	node, err := xmlquery.Query(doc, expr)
	if err != nil {
		result.Passed = false
		result.Error = "XPath query failed: " + err.Error()
		return result
	}

	var actualValue string
	if node != nil {
		actualValue = node.InnerText()
	}

	result.Actual = actualValue

	operator := strings.ToLower(assertion.Operator)

	switch operator {
	case "exists", "present":
		result.Passed = node != nil
		return result
	case "not_exists", "absent":
		result.Passed = node == nil
		return result
	}

	if node == nil {
		result.Passed = false
		result.Error = "XPath element not found"
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

func ExtractXPath(body string, expr string) (string, error) {
	if body == "" {
		return "", nil
	}

	doc, err := xmlquery.Parse(strings.NewReader(body))
	if err != nil {
		return "", err
	}

	node, err := xmlquery.Query(doc, expr)
	if err != nil {
		return "", err
	}

	if node == nil {
		return "", nil
	}

	return node.InnerText(), nil
}
