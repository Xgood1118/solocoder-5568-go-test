package assert

import (
	"apitester/internal/assert/assertions"
	"apitester/internal/models"
	"fmt"
	"strconv"
	"strings"
)

func Extract(response *models.Response, extractConfig map[string]string) (map[string]string, error) {
	result := make(map[string]string)

	if extractConfig == nil || len(extractConfig) == 0 {
		return result, nil
	}

	bodyStr := response.BodyString
	if bodyStr == "" && len(response.Body) > 0 {
		bodyStr = string(response.Body)
	}

	for varName, expr := range extractConfig {
		value, err := extractValue(response, bodyStr, expr)
		if err != nil {
			return result, fmt.Errorf("extract '%s' failed: %w", varName, err)
		}
		result[varName] = value
	}

	return result, nil
}

func extractValue(response *models.Response, bodyStr string, expr string) (string, error) {
	expr = strings.TrimSpace(expr)

	if strings.HasPrefix(expr, "header.") {
		headerName := strings.TrimPrefix(expr, "header.")
		for k, v := range response.Headers {
			if strings.EqualFold(k, headerName) {
				return v, nil
			}
		}
		return "", nil
	}

	if strings.HasPrefix(expr, "regex:") {
		pattern := strings.TrimPrefix(expr, "regex:")
		pattern = strings.Trim(pattern, "\"'")
		return assertions.ExtractRegex(bodyStr, pattern)
	}

	if strings.HasPrefix(expr, "regexp:") {
		pattern := strings.TrimPrefix(expr, "regexp:")
		pattern = strings.Trim(pattern, "\"'")
		return assertions.ExtractRegex(bodyStr, pattern)
	}

	if strings.HasPrefix(expr, "xpath:") {
		xpathExpr := strings.TrimPrefix(expr, "xpath:")
		return assertions.ExtractXPath(bodyStr, xpathExpr)
	}

	if expr == "status" || expr == "status_code" {
		return strconv.Itoa(response.StatusCode), nil
	}

	if strings.HasPrefix(expr, "$.") || strings.HasPrefix(expr, "$[") || strings.Contains(expr, ".") {
		return assertions.ExtractJSONPath(bodyStr, expr)
	}

	return assertions.ExtractJSONPath(bodyStr, expr)
}
