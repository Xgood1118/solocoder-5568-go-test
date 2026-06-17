package report

import (
	"encoding/json"
	"fmt"
	"time"

	"apitester/internal/models"
)

type JSONReporter struct{}

func NewJSONReporter() *JSONReporter {
	return &JSONReporter{}
}

type jsonReport struct {
	Version   string              `json:"version"`
	Generated string              `json:"generated"`
	Suite     models.SuiteResult  `json:"suite"`
	Summary   jsonReportSummary   `json:"summary"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type jsonReportSummary struct {
	Total      int     `json:"total"`
	Passed     int     `json:"passed"`
	Failed     int     `json:"failed"`
	Skipped    int     `json:"skipped"`
	PassRate   float64 `json:"pass_rate"`
	Duration   float64 `json:"duration_seconds"`
	DurationStr string `json:"duration"`
	StartTime  string  `json:"start_time"`
	EndTime    string  `json:"end_time"`
}

func (r *JSONReporter) Generate(suite models.SuiteResult) (string, error) {
	report := jsonReport{
		Version:   "1.0",
		Generated: time.Now().Format(time.RFC3339),
		Suite:   suite,
		Summary: jsonReportSummary{
			Total:       suite.Total,
			Passed:      suite.Passed,
			Failed:      suite.Failed,
			Skipped:     suite.Skipped,
			PassRate:    suite.PassRate,
			Duration:    suite.Duration.Seconds(),
			DurationStr: suite.Duration.String(),
			StartTime:   suite.StartTime.Format(time.RFC3339),
			EndTime:     suite.EndTime.Format(time.RFC3339),
		},
		Metadata: map[string]string{
			"generator": "apitester",
		},
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal json: %w", err)
	}

	return string(data), nil
}
