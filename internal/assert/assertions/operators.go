package assertions

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func Compare(operator string, actual, expected any) (bool, error) {
	switch strings.ToLower(operator) {
	case "==", "eq", "equals":
		return compareEquals(actual, expected)
	case "!=", "ne", "not_equals":
		result, err := compareEquals(actual, expected)
		return !result, err
	case ">", "gt":
		return compareGreater(actual, expected)
	case "<", "lt":
		return compareLess(actual, expected)
	case ">=", "gte":
		result, err := compareLess(actual, expected)
		return !result, err
	case "<=", "lte":
		result, err := compareGreater(actual, expected)
		return !result, err
	case "contains":
		return compareContains(actual, expected)
	case "not_contains":
		result, err := compareContains(actual, expected)
		return !result, err
	case "in":
		return compareIn(actual, expected)
	case "not_in":
		result, err := compareIn(actual, expected)
		return !result, err
	case "regex", "regexp", "matches":
		return compareRegex(actual, expected)
	case "starts_with":
		return compareStartsWith(actual, expected)
	case "ends_with":
		return compareEndsWith(actual, expected)
	case "exists", "present":
		return actual != nil, nil
	case "not_exists", "absent":
		return actual == nil, nil
	default:
		return false, fmt.Errorf("unsupported operator: %s", operator)
	}
}

func compareEquals(actual, expected any) (bool, error) {
	actualFloat, actualOk := toFloat64(actual)
	expectedFloat, expectedOk := toFloat64(expected)

	if actualOk && expectedOk {
		return actualFloat == expectedFloat, nil
	}

	actualStr := toString(actual)
	expectedStr := toString(expected)
	return actualStr == expectedStr, nil
}

func compareGreater(actual, expected any) (bool, error) {
	actualFloat, actualOk := toFloat64(actual)
	expectedFloat, expectedOk := toFloat64(expected)

	if actualOk && expectedOk {
		return actualFloat > expectedFloat, nil
	}

	actualStr := toString(actual)
	expectedStr := toString(expected)
	return actualStr > expectedStr, nil
}

func compareLess(actual, expected any) (bool, error) {
	actualFloat, actualOk := toFloat64(actual)
	expectedFloat, expectedOk := toFloat64(expected)

	if actualOk && expectedOk {
		return actualFloat < expectedFloat, nil
	}

	actualStr := toString(actual)
	expectedStr := toString(expected)
	return actualStr < expectedStr, nil
}

func compareContains(actual, expected any) (bool, error) {
	actualStr := toString(actual)
	expectedStr := toString(expected)
	return strings.Contains(actualStr, expectedStr), nil
}

func compareIn(actual, expected any) (bool, error) {
	actualStr := toString(actual)

	switch v := expected.(type) {
	case []any:
		for _, item := range v {
			if toString(item) == actualStr {
				return true, nil
			}
		}
		return false, nil
	case []string:
		for _, item := range v {
			if item == actualStr {
				return true, nil
			}
		}
		return false, nil
	case []int:
		actualFloat, actualOk := toFloat64(actual)
		if actualOk {
			for _, item := range v {
				if float64(item) == actualFloat {
					return true, nil
				}
			}
		}
		return false, nil
	case string:
		parts := strings.Split(strings.Trim(v, "[]"), ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == actualStr {
				return true, nil
			}
		}
		return false, nil
	default:
		return false, fmt.Errorf("expected value must be a list for 'in' operator")
	}
}

func compareRegex(actual, expected any) (bool, error) {
	actualStr := toString(actual)
	pattern := toString(expected)

	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, fmt.Errorf("invalid regex pattern: %w", err)
	}

	return re.MatchString(actualStr), nil
}

func compareStartsWith(actual, expected any) (bool, error) {
	actualStr := toString(actual)
	expectedStr := toString(expected)
	return strings.HasPrefix(actualStr, expectedStr), nil
}

func compareEndsWith(actual, expected any) (bool, error) {
	actualStr := toString(actual)
	expectedStr := toString(expected)
	return strings.HasSuffix(actualStr, expectedStr), nil
}

func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case int:
		return float64(val), true
	case int8:
		return float64(val), true
	case int16:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case uint:
		return float64(val), true
	case uint8:
		return float64(val), true
	case uint16:
		return float64(val), true
	case uint32:
		return float64(val), true
	case uint64:
		return float64(val), true
	case float32:
		return float64(val), true
	case float64:
		return val, true
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f, true
		}
		return 0, false
	default:
		return 0, false
	}
}
