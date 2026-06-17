package mock

import (
	"net/http"
	"strings"

	"apitester/internal/models"
)

func MatchRequest(r *http.Request, rules []*models.MockRule) *models.MockRule {
	for _, rule := range rules {
		if matchPath(r.URL.Path, rule.MatchPath) &&
			matchMethod(r.Method, rule.MatchMethod) &&
			matchHeaders(r.Header, rule.MatchHeaders) {
			return rule
		}
	}
	return nil
}

func matchPath(requestPath, pattern string) bool {
	if pattern == "" {
		return true
	}

	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(requestPath, "/")

	if len(patternParts) != len(pathParts) && !strings.HasSuffix(pattern, "/*") {
		return false
	}

	for i, patternPart := range patternParts {
		if patternPart == "*" {
			if i == len(patternParts)-1 {
				return true
			}
			continue
		}

		if i >= len(pathParts) {
			return false
		}

		if patternPart != pathParts[i] {
			return false
		}
	}

	return len(patternParts) == len(pathParts)
}

func matchMethod(requestMethod, ruleMethod string) bool {
	if ruleMethod == "" {
		return true
	}
	return strings.EqualFold(requestMethod, ruleMethod)
}

func matchHeaders(requestHeaders http.Header, ruleHeaders map[string]string) bool {
	if len(ruleHeaders) == 0 {
		return true
	}

	for key, value := range ruleHeaders {
		requestValue := requestHeaders.Get(key)
		if requestValue == "" || requestValue != value {
			return false
		}
	}

	return true
}
