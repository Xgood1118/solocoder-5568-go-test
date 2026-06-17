package assert

import (
	"apitester/internal/assert/assertions"
	"apitester/internal/models"
	"fmt"
	"strings"
)

type AssertionEngine struct {
}

func NewAssertionEngine() *AssertionEngine {
	return &AssertionEngine{}
}

func (e *AssertionEngine) Assert(response *models.Response, assertionsList []*models.Assertion) []*models.AssertionResult {
	results := make([]*models.AssertionResult, 0, len(assertionsList))

	for _, assertion := range assertionsList {
		result := e.assertSingle(response, assertion)
		results = append(results, result)
	}

	return results
}

func (e *AssertionEngine) AssertAndExtract(response *models.Response, assertionsList []*models.Assertion, extractConfig map[string]string) ([]*models.AssertionResult, map[string]string, error) {
	assertionResults := e.Assert(response, assertionsList)

	extractedVars, err := Extract(response, extractConfig)
	if err != nil {
		return assertionResults, extractedVars, err
	}

	return assertionResults, extractedVars, nil
}

func (e *AssertionEngine) assertSingle(response *models.Response, assertion *models.Assertion) *models.AssertionResult {
	assertType := strings.ToLower(strings.TrimSpace(assertion.Type))

	switch assertType {
	case "status", "status_code", "statuscode":
		return assertions.AssertStatus(response, assertion)
	case "header", "headers":
		return assertions.AssertHeader(response, assertion)
	case "body_jsonpath", "jsonpath", "body.jsonpath":
		return assertions.AssertBodyJSONPath(response, assertion)
	case "body_xpath", "xpath", "body.xpath":
		return assertions.AssertBodyXPath(response, assertion)
	case "body_regex", "regex", "body.regex":
		return assertions.AssertBodyRegex(response, assertion)
	case "body_schema", "schema", "json_schema", "body.schema":
		return assertions.AssertBodySchema(response, assertion)
	case "response_time", "latency", "responsetime":
		return assertions.AssertResponseTime(response, assertion)
	default:
		return &models.AssertionResult{
			Passed:   false,
			Type:     assertion.Type,
			Property: assertion.Property,
			Expected: assertion.Value,
			Operator: assertion.Operator,
			Message:  assertion.Message,
			Error:    fmt.Sprintf("unsupported assertion type: %s", assertion.Type),
		}
	}
}
