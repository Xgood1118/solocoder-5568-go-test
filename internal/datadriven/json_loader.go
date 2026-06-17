package datadriven

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func LoadJSONArray(source string) ([]map[string]any, error) {
	var data []byte
	var err error

	if isJSONString(source) {
		data = []byte(source)
	} else {
		data, err = os.ReadFile(source)
		if err != nil {
			return nil, fmt.Errorf("failed to read json file: %w", err)
		}
	}

	var result []map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse json: %w", err)
	}

	return result, nil
}

func isJSONString(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return false
	}
	return (s[0] == '[' && s[len(s)-1] == ']') || (s[0] == '{' && s[len(s)-1] == '}')
}
