package filter

import (
	"apitester/internal/models"
	"apitester/pkg/utils"
	"fmt"
	"regexp"
	"strings"
)

type DryRunResult struct {
	Valid   bool
	CaseID  string
	CaseName string
	Errors  []string
}

var validAssertionTypes = map[string]bool{
	"status":          true,
	"status_code":     true,
	"header":          true,
	"body":            true,
	"json":            true,
	"jsonpath":        true,
	"xpath":           true,
	"regex":           true,
	"contains":        true,
	"equals":          true,
	"not_equals":      true,
	"greater_than":    true,
	"less_than":       true,
	"greater_equals":  true,
	"less_equals":     true,
	"empty":           true,
	"not_empty":       true,
	"null":            true,
	"not_null":        true,
	"array_contains":  true,
	"array_length":    true,
	"schema":          true,
	"latency":         true,
	"duration":        true,
}

var validOperators = map[string]bool{
	"==":    true,
	"!=":    true,
	"<":     true,
	">":     true,
	"<=":    true,
	">=":    true,
	"contains":   true,
	"matches":    true,
	"startsWith": true,
	"endsWith":   true,
}

func DryRunValidate(cases []*models.TestCase, globalVars map[string]any) []*DryRunResult {
	results := make([]*DryRunResult, 0, len(cases))

	for _, tc := range cases {
		result := &DryRunResult{
			CaseID:   tc.ID,
			CaseName: tc.Name,
			Valid:    true,
			Errors:   make([]string, 0),
		}

		validateCase(tc, globalVars, result)

		if len(result.Errors) > 0 {
			result.Valid = false
		}

		results = append(results, result)
	}

	return results
}

func validateCase(tc *models.TestCase, globalVars map[string]any, result *DryRunResult) {
	allVars := make(map[string]bool)
	for k := range globalVars {
		allVars[k] = true
	}
	for k := range tc.Variables {
		allVars[k] = true
	}

	if tc.Request != nil {
		validateRequest(tc.Request, allVars, result)
	}

	if tc.WebSocket != nil {
		validateWebSocket(tc.WebSocket, allVars, result)
	}

	if tc.SSE != nil {
		validateSSE(tc.SSE, allVars, result)
	}

	for i, assertion := range tc.Assertions {
		validateAssertion(assertion, i, result)
	}

	for key, expr := range tc.Extract {
		validateExtract(key, expr, result)
	}
}

func validateRequest(req *models.Request, vars map[string]bool, result *DryRunResult) {
	if req.Method == "" {
		result.Errors = append(result.Errors, "Request method is empty")
	}

	if req.URL == "" {
		result.Errors = append(result.Errors, "Request URL is empty")
	}

	validateStringVariables(req.URL, vars, "URL", result)

	for k, v := range req.Headers {
		validateStringVariables(v, vars, fmt.Sprintf("Header[%s]", k), result)
	}

	for k, v := range req.QueryParams {
		validateStringVariables(v, vars, fmt.Sprintf("QueryParam[%s]", k), result)
	}

	if req.Body != nil {
		validateRequestBody(req.Body, vars, result)
	}
}

func validateRequestBody(body *models.RequestBody, vars map[string]bool, result *DryRunResult) {
	if body.Type == "" {
		result.Errors = append(result.Errors, "Request body type is empty")
	}

	switch body.Type {
	case "json":
		if body.JSON != nil {
			validateJSONVariables(body.JSON, vars, "Body.JSON", result)
		}
	case "form":
		for k, v := range body.Form {
			validateStringVariables(v, vars, fmt.Sprintf("Body.Form[%s]", k), result)
		}
	case "multipart":
		for i, field := range body.Multipart {
			validateStringVariables(field.Value, vars, fmt.Sprintf("Body.Multipart[%d].Value", i), result)
		}
	case "raw":
		validateStringVariables(body.Raw, vars, "Body.Raw", result)
	case "graphql":
		if body.GraphQL != nil {
			validateStringVariables(body.GraphQL.Query, vars, "Body.GraphQL.Query", result)
			validateJSONVariables(body.GraphQL.Variables, vars, "Body.GraphQL.Variables", result)
		}
	}
}

func validateWebSocket(ws *models.WebSocketConfig, vars map[string]bool, result *DryRunResult) {
	if ws.URL == "" {
		result.Errors = append(result.Errors, "WebSocket URL is empty")
	}

	validateStringVariables(ws.URL, vars, "WebSocket.URL", result)

	for k, v := range ws.Headers {
		validateStringVariables(v, vars, fmt.Sprintf("WebSocket.Header[%s]", k), result)
	}

	for i, msg := range ws.Messages {
		validateStringVariables(msg.Content, vars, fmt.Sprintf("WebSocket.Message[%d].Content", i), result)

		for j, assertion := range msg.Assertions {
			validateAssertion(assertion, j, result)
		}

		for key, expr := range msg.Extract {
			validateExtract(key, expr, result)
		}
	}
}

func validateSSE(sse *models.SSEConfig, vars map[string]bool, result *DryRunResult) {
	if sse.URL == "" {
		result.Errors = append(result.Errors, "SSE URL is empty")
	}

	validateStringVariables(sse.URL, vars, "SSE.URL", result)

	for k, v := range sse.Headers {
		validateStringVariables(v, vars, fmt.Sprintf("SSE.Header[%s]", k), result)
	}

	for _, event := range sse.Events {
		for j, assertion := range event.Assertions {
			validateAssertion(assertion, j, result)
		}

		for key, expr := range event.Extract {
			validateExtract(key, expr, result)
		}
	}
}

func validateAssertion(assertion *models.Assertion, index int, result *DryRunResult) {
	if assertion.Type == "" {
		result.Errors = append(result.Errors, fmt.Sprintf("Assertion[%d]: type is empty", index))
		return
	}

	if !validAssertionTypes[strings.ToLower(assertion.Type)] {
		result.Errors = append(result.Errors, fmt.Sprintf("Assertion[%d]: invalid type '%s'", index, assertion.Type))
	}

	if assertion.Operator != "" && !validOperators[assertion.Operator] {
		result.Errors = append(result.Errors, fmt.Sprintf("Assertion[%d]: invalid operator '%s'", index, assertion.Operator))
	}

	prop := assertion.Property
	if prop != "" {
		if strings.HasPrefix(prop, "$.") || strings.Contains(prop, "$[") {
			if err := validateJSONPath(prop); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Assertion[%d]: invalid JSONPath '%s': %v", index, prop, err))
			}
		} else if strings.HasPrefix(prop, "//") || strings.HasPrefix(prop, "./") {
			if err := validateXPath(prop); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Assertion[%d]: invalid XPath '%s': %v", index, prop, err))
			}
		}
	}
}

func validateExtract(key string, expr string, result *DryRunResult) {
	if key == "" {
		result.Errors = append(result.Errors, "Extract: key is empty")
	}

	if expr == "" {
		result.Errors = append(result.Errors, fmt.Sprintf("Extract[%s]: expression is empty", key))
		return
	}

	if strings.HasPrefix(expr, "$.") || strings.Contains(expr, "$[") {
		if err := validateJSONPath(expr); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Extract[%s]: invalid JSONPath '%s': %v", key, expr, err))
		}
	} else if strings.HasPrefix(expr, "//") || strings.HasPrefix(expr, "./") {
		if err := validateXPath(expr); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Extract[%s]: invalid XPath '%s': %v", key, expr, err))
		}
	} else if strings.HasPrefix(expr, "header.") {
		headerName := strings.TrimPrefix(expr, "header.")
		if headerName == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("Extract[%s]: invalid header extraction '%s'", key, expr))
		}
	}
}

func validateStringVariables(s string, vars map[string]bool, fieldName string, result *DryRunResult) {
	if !utils.HasVariables(s) {
		return
	}

	extractedVars := utils.ExtractVariables(s)
	for _, v := range extractedVars {
		varName := v
		if idx := strings.Index(v, "."); idx > 0 {
			varName = v[:idx]
		}
		if !vars[varName] {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: reference to undefined variable '{{%s}}'", fieldName, v))
		}
	}
}

func validateJSONVariables(v interface{}, vars map[string]bool, fieldName string, result *DryRunResult) {
	switch val := v.(type) {
	case string:
		validateStringVariables(val, vars, fieldName, result)
	case map[string]any:
		for k, vv := range val {
			validateJSONVariables(vv, vars, fmt.Sprintf("%s.%s", fieldName, k), result)
		}
	case []any:
		for i, vv := range val {
			validateJSONVariables(vv, vars, fmt.Sprintf("%s[%d]", fieldName, i), result)
		}
	}
}

func validateJSONPath(expr string) error {
	if expr == "" {
		return fmt.Errorf("empty expression")
	}

	if !strings.HasPrefix(expr, "$") && !strings.HasPrefix(expr, "@") {
		return fmt.Errorf("must start with $ or @")
	}

	validChars := regexp.MustCompile(`^[$@a-zA-Z0-9_\.\[\]\(\)\*\+\-\?\,\:'" ]+$`)
	if !validChars.MatchString(expr) {
		return fmt.Errorf("contains invalid characters")
	}

	bracketDepth := 0
	parenDepth := 0
	inString := false
	stringChar := rune(0)

	for i, ch := range expr {
		if inString {
			if ch == stringChar && (i == 0 || expr[i-1] != '\\') {
				inString = false
			}
			continue
		}

		switch ch {
		case '\'', '"':
			inString = true
			stringChar = ch
		case '[':
			bracketDepth++
		case ']':
			bracketDepth--
			if bracketDepth < 0 {
				return fmt.Errorf("unmatched ']'")
			}
		case '(':
			parenDepth++
		case ')':
			parenDepth--
			if parenDepth < 0 {
				return fmt.Errorf("unmatched ')'")
			}
		}
	}

	if bracketDepth != 0 {
		return fmt.Errorf("unmatched '['")
	}
	if parenDepth != 0 {
		return fmt.Errorf("unmatched '('")
	}
	if inString {
		return fmt.Errorf("unterminated string")
	}

	return nil
}

func validateXPath(expr string) error {
	if expr == "" {
		return fmt.Errorf("empty expression")
	}

	validChars := regexp.MustCompile(`^[a-zA-Z0-9_\.\[\]\(\)\*\+\-\?\,\:@/=!<>'" ]+$`)
	if !validChars.MatchString(expr) {
		return fmt.Errorf("contains invalid characters")
	}

	bracketDepth := 0
	parenDepth := 0
	inString := false
	stringChar := rune(0)

	for i, ch := range expr {
		if inString {
			if ch == stringChar && (i == 0 || expr[i-1] != '\\') {
				inString = false
			}
			continue
		}

		switch ch {
		case '\'', '"':
			inString = true
			stringChar = ch
		case '[':
			bracketDepth++
		case ']':
			bracketDepth--
			if bracketDepth < 0 {
				return fmt.Errorf("unmatched ']'")
			}
		case '(':
			parenDepth++
		case ')':
			parenDepth--
			if parenDepth < 0 {
				return fmt.Errorf("unmatched ')'")
			}
		}
	}

	if bracketDepth != 0 {
		return fmt.Errorf("unmatched '['")
	}
	if parenDepth != 0 {
		return fmt.Errorf("unmatched '('")
	}
	if inString {
		return fmt.Errorf("unterminated string")
	}

	return nil
}
