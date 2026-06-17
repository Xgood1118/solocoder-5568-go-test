package utils

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var varRegex = regexp.MustCompile(`\{\{\s*([^}]+)\s*\}\}`)

func RandomString(n int) string {
	b := make([]byte, n/2)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func ExponentialBackoff(attempt int, base time.Duration, max time.Duration) time.Duration {
	backoff := base * time.Duration(math.Pow(2, float64(attempt)))
	if backoff > max {
		backoff = max
	}
	return backoff
}

func ReadFile(path string) ([]byte, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(absPath)
}

func WriteFile(path string, data []byte) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(absPath, data, 0644)
}

func ToJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

func ToJSONBytes(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

func FromJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func ContainsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func IntersectStrings(a, b []string) []string {
	set := make(map[string]bool)
	for _, s := range a {
		set[s] = true
	}
	var result []string
	for _, s := range b {
		if set[s] {
			result = append(result, s)
		}
	}
	return result
}

func MergeMaps(a, b map[string]any) map[string]any {
	result := make(map[string]any)
	for k, v := range a {
		result[k] = v
	}
	for k, v := range b {
		result[k] = v
	}
	return result
}

func MergeStringMaps(a, b map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range a {
		result[k] = v
	}
	for k, v := range b {
		result[k] = v
	}
	return result
}

func HasVariables(s string) bool {
	return varRegex.MatchString(s)
}

func ExtractVariables(s string) []string {
	matches := varRegex.FindAllStringSubmatch(s, -1)
	vars := make([]string, 0, len(matches))
	for _, m := range matches {
		vars = append(vars, strings.TrimSpace(m[1]))
	}
	return vars
}

func GetEnv(key, defaultValue string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return defaultValue
}

func NowUnixNano() int64 {
	return time.Now().UnixNano()
}

func FormatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dμs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Microseconds())/1000.0)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

func ParseBool(s string, defaultValue bool) bool {
	switch strings.ToLower(s) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return defaultValue
	}
}

func ResolvePath(basePath, relPath string) string {
	if filepath.IsAbs(relPath) {
		return relPath
	}
	return filepath.Join(filepath.Dir(basePath), relPath)
}

func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

func FormatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func PrettyJSON(v interface{}) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(data)
}

func GetHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home
}

func ExpandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		return filepath.Join(GetHomeDir(), path[1:])
	}
	return path
}

func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
