package parser

import (
	"fmt"
	"strings"

	"apitester/internal/models"
)

var validAssertionTypes = map[string]bool{
	"status":          true,
	"header":          true,
	"body_jsonpath":   true,
	"body_xpath":      true,
	"body_regex":      true,
	"body_schema":     true,
	"response_time":   true,
}

type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

func (v *Validator) ValidateSuite(suite *models.TestSuite) error {
	if suite == nil {
		return fmt.Errorf("suite is nil")
	}

	if strings.TrimSpace(suite.Name) == "" {
		return fmt.Errorf("suite name is required")
	}

	for i, tc := range suite.TestCases {
		if err := v.ValidateCase(tc); err != nil {
			return fmt.Errorf("test case[%d]: %w", i, err)
		}
	}

	for i, tc := range suite.Setup {
		if err := v.ValidateCase(tc); err != nil {
			return fmt.Errorf("setup case[%d]: %w", i, err)
		}
	}

	for i, tc := range suite.Teardown {
		if err := v.ValidateCase(tc); err != nil {
			return fmt.Errorf("teardown case[%d]: %w", i, err)
		}
	}

	return nil
}

func (v *Validator) ValidateCase(tc *models.TestCase) error {
	if tc == nil {
		return fmt.Errorf("test case is nil")
	}

	if strings.TrimSpace(tc.Name) == "" {
		return fmt.Errorf("test case name is required")
	}

	if tc.Request == nil {
		return fmt.Errorf("test case '%s': request is required", tc.Name)
	}

	if strings.TrimSpace(tc.Request.Method) == "" {
		return fmt.Errorf("test case '%s': request.method is required", tc.Name)
	}

	if strings.TrimSpace(tc.Request.URL) == "" {
		return fmt.Errorf("test case '%s': request.url is required", tc.Name)
	}

	for i, a := range tc.Assertions {
		if err := v.validateAssertion(a); err != nil {
			return fmt.Errorf("test case '%s' assertion[%d]: %w", tc.Name, i, err)
		}
	}

	return nil
}

func (v *Validator) validateAssertion(a *models.Assertion) error {
	if a == nil {
		return fmt.Errorf("assertion is nil")
	}

	if !validAssertionTypes[a.Type] {
		return fmt.Errorf("invalid assertion type: '%s', must be one of: %s",
			a.Type, strings.Join(getValidAssertionTypes(), ", "))
	}

	return nil
}

func getValidAssertionTypes() []string {
	types := make([]string, 0, len(validAssertionTypes))
	for t := range validAssertionTypes {
		types = append(types, t)
	}
	return types
}
