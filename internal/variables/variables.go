package variables

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"apitester/internal/models"
)

type VariableStore struct {
	builtin     map[string]any
	testCase    map[string]any
	global      map[string]any
	environment map[string]any
}

func NewVariableStore() *VariableStore {
	return &VariableStore{
		builtin:     make(map[string]any),
		testCase:    make(map[string]any),
		global:      make(map[string]any),
		environment: make(map[string]any),
	}
}

func (vs *VariableStore) Set(scope models.VariableScope, key string, value any) {
	switch scope {
	case models.ScopeBuiltin:
		vs.builtin[key] = value
	case models.ScopeTestCase:
		vs.testCase[key] = value
	case models.ScopeGlobal:
		vs.global[key] = value
	case models.ScopeEnvironment:
		vs.environment[key] = value
	}
}

func (vs *VariableStore) Get(key string) (any, bool) {
	if v, ok := vs.environment[key]; ok {
		return v, true
	}
	if v, ok := vs.global[key]; ok {
		return v, true
	}
	if v, ok := vs.testCase[key]; ok {
		return v, true
	}
	if v, ok := vs.builtin[key]; ok {
		return v, true
	}
	return nil, false
}

func (vs *VariableStore) GetAll() map[string]any {
	result := make(map[string]any)
	for k, v := range vs.builtin {
		result[k] = v
	}
	for k, v := range vs.testCase {
		result[k] = v
	}
	for k, v := range vs.global {
		result[k] = v
	}
	for k, v := range vs.environment {
		result[k] = v
	}
	return result
}

func (vs *VariableStore) SetBuiltinVariables() {
	now := time.Now()
	vs.builtin["$timestamp"] = fmt.Sprintf("%d", now.Unix())
	vs.builtin["$date"] = now.Format("2006-01-02")
	vs.builtin["$time"] = now.Format("15:04:05")
	vs.builtin["$datetime"] = now.Format("2006-01-02 15:04:05")
	vs.builtin["$random"] = randomString(8)
	vs.builtin["$uuid"] = generateUUID()
	vs.builtin["$timestampMs"] = fmt.Sprintf("%d", now.UnixMilli())
	vs.builtin["$timestampNano"] = fmt.Sprintf("%d", now.UnixNano())
	vs.builtin["$year"] = fmt.Sprintf("%d", now.Year())
	vs.builtin["$month"] = fmt.Sprintf("%02d", now.Month())
	vs.builtin["$day"] = fmt.Sprintf("%02d", now.Day())
	vs.builtin["$hour"] = fmt.Sprintf("%02d", now.Hour())
	vs.builtin["$minute"] = fmt.Sprintf("%02d", now.Minute())
	vs.builtin["$second"] = fmt.Sprintf("%02d", now.Second())

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			vs.builtin["$env."+parts[0]] = parts[1]
		}
	}

	homeDir, _ := os.UserHomeDir()
	vs.builtin["$env.HOME"] = homeDir
}

func (vs *VariableStore) ClearScope(scope models.VariableScope) {
	switch scope {
	case models.ScopeBuiltin:
		vs.builtin = make(map[string]any)
	case models.ScopeTestCase:
		vs.testCase = make(map[string]any)
	case models.ScopeGlobal:
		vs.global = make(map[string]any)
	case models.ScopeEnvironment:
		vs.environment = make(map[string]any)
	}
}

func (vs *VariableStore) SetMap(scope models.VariableScope, vars map[string]any) {
	for k, v := range vars {
		vs.Set(scope, k, v)
	}
}

func randomString(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)[:n]
}

func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func CallFunction(name string, args []string) (any, error) {
	switch name {
	case "$random":
		return randomString(8), nil
	case "$randomInt":
		if len(args) != 2 {
			return nil, fmt.Errorf("$randomInt requires 2 arguments: min and max")
		}
		min, err := strconv.Atoi(strings.TrimSpace(args[0]))
		if err != nil {
			return nil, fmt.Errorf("invalid min value: %s", args[0])
		}
		max, err := strconv.Atoi(strings.TrimSpace(args[1]))
		if err != nil {
			return nil, fmt.Errorf("invalid max value: %s", args[1])
		}
		if min >= max {
			return nil, fmt.Errorf("min must be less than max")
		}
		b := make([]byte, 8)
		rand.Read(b)
		n := int64(b[0])<<56 | int64(b[1])<<48 | int64(b[2])<<40 | int64(b[3])<<32 |
			int64(b[4])<<24 | int64(b[5])<<16 | int64(b[6])<<8 | int64(b[7])
		n = int64(math.Abs(float64(n)))
		return min + int(n%int64(max-min+1)), nil
	case "$timestamp":
		return fmt.Sprintf("%d", time.Now().Unix()), nil
	case "$timestampMs":
		return fmt.Sprintf("%d", time.Now().UnixMilli()), nil
	case "$uuid":
		return generateUUID(), nil
	case "$date":
		format := "2006-01-02"
		if len(args) > 0 {
			format = strings.Trim(strings.TrimSpace(args[0]), "'\"")
			format = convertDateFormat(format)
		}
		return time.Now().Format(format), nil
	case "$time":
		format := "15:04:05"
		if len(args) > 0 {
			format = strings.Trim(strings.TrimSpace(args[0]), "'\"")
			format = convertDateFormat(format)
		}
		return time.Now().Format(format), nil
	case "$now":
		return time.Now().Format("2006-01-02 15:04:05"), nil
	case "$env":
		if len(args) < 1 {
			return nil, fmt.Errorf("$env requires 1 argument: name")
		}
		name := strings.Trim(strings.TrimSpace(args[0]), "'\"")
		if v, ok := os.LookupEnv(name); ok {
			return v, nil
		}
		return "", nil
	case "$formatDate":
		if len(args) < 2 {
			return nil, fmt.Errorf("$formatDate requires 2 arguments: timestamp and format")
		}
		tsStr := strings.TrimSpace(args[0])
		format := strings.Trim(strings.TrimSpace(args[1]), "'\"")
		format = convertDateFormat(format)

		ts, err := strconv.ParseInt(tsStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid timestamp: %s", tsStr)
		}
		var t time.Time
		if len(tsStr) == 13 {
			t = time.UnixMilli(ts)
		} else if len(tsStr) == 19 {
			t = time.Unix(0, ts)
		} else {
			t = time.Unix(ts, 0)
		}
		return t.Format(format), nil
	default:
		return nil, fmt.Errorf("unknown function: %s", name)
	}
}

func convertDateFormat(format string) string {
	format = strings.ReplaceAll(format, "YYYY", "2006")
	format = strings.ReplaceAll(format, "YY", "06")
	format = strings.ReplaceAll(format, "MM", "01")
	format = strings.ReplaceAll(format, "DD", "02")
	format = strings.ReplaceAll(format, "HH", "15")
	format = strings.ReplaceAll(format, "hh", "03")
	format = strings.ReplaceAll(format, "mm", "04")
	format = strings.ReplaceAll(format, "ss", "05")
	format = strings.ReplaceAll(format, "A", "PM")
	format = strings.ReplaceAll(format, "a", "pm")
	return format
}
