package report

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"apitester/internal/models"
	"apitester/pkg/utils"
)

const (
	colorGreen  = "\033[32m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorReset  = "\033[0m"
)

type ConsoleReporter struct {
	mu          sync.Mutex
	spinnerChars []string
	spinnerIdx  int
	lastLineLen int
}

func NewConsoleReporter() *ConsoleReporter {
	return &ConsoleReporter{
		spinnerChars: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	}
}

func (r *ConsoleReporter) ReportProgress(progress models.CLIProgress) {
	r.mu.Lock()
	defer r.mu.Unlock()

	spinner := r.spinnerChars[r.spinnerIdx]
	r.spinnerIdx = (r.spinnerIdx + 1) % len(r.spinnerChars)

	percent := 0
	if progress.Total > 0 {
		percent = progress.Current * 100 / progress.Total
	}

	barWidth := 30
	filled := 0
	if progress.Total > 0 {
		filled = progress.Current * barWidth / progress.Total
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	name := utils.TruncateString(progress.CurrentName, 40)
	line := fmt.Sprintf("\r%s %s [%s] %d/%d (%d%%) %s",
		colorCyan+spinner+colorReset,
		name,
		bar,
		progress.Current,
		progress.Total,
		percent,
		progress.Status,
	)

	r.clearLine()
	fmt.Print(line)
	r.lastLineLen = len(line)
}

func (r *ConsoleReporter) ReportCaseResult(result models.TestResult) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.clearLine()

	var status, statusColor string
	switch {
	case result.Skipped:
		status = "SKIP"
		statusColor = colorYellow
	case result.Passed:
		status = "PASS"
		statusColor = colorGreen
	default:
		status = "FAIL"
		statusColor = colorRed
	}

	line := fmt.Sprintf("%s[%s]%s %s (%.2fms)",
		statusColor,
		status,
		colorReset,
		result.Name,
		float64(result.Duration.Microseconds())/1000.0,
	)

	fmt.Println(line)
	r.lastLineLen = 0
}

func (r *ConsoleReporter) ReportSummary(suite models.SuiteResult) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.clearLine()

	fmt.Println()
	fmt.Println(colorCyan + "════════════════════════════════════════════════════════════" + colorReset)
	fmt.Println(colorCyan + "  测试报告摘要" + colorReset)
	fmt.Println(colorCyan + "════════════════════════════════════════════════════════════" + colorReset)
	fmt.Println()
	fmt.Printf("  测试套件: %s\n", suite.Name)
	fmt.Printf("  开始时间: %s\n", utils.FormatTime(suite.StartTime))
	fmt.Printf("  结束时间: %s\n", utils.FormatTime(suite.EndTime))
	fmt.Println()
	fmt.Println("  " + colorCyan + "─────────────────────────────────────────────────────────" + colorReset)
	fmt.Printf("  %-15s %s\n", "总用例数:", fmt.Sprintf("%d", suite.Total))
	fmt.Printf("  %-15s %s\n", "通过:", colorGreen+fmt.Sprintf("%d", suite.Passed)+colorReset)
	fmt.Printf("  %-15s %s\n", "失败:", colorRed+fmt.Sprintf("%d", suite.Failed)+colorReset)
	fmt.Printf("  %-15s %s\n", "跳过:", colorYellow+fmt.Sprintf("%d", suite.Skipped)+colorReset)
	fmt.Printf("  %-15s %.2f%%\n", "通过率:", suite.PassRate)
	fmt.Printf("  %-15s %s\n", "总耗时:", utils.FormatDuration(suite.Duration))
	fmt.Println("  " + colorCyan + "─────────────────────────────────────────────────────────" + colorReset)
	fmt.Println()

	if suite.Failed > 0 {
		fmt.Println(colorRed + "  失败的测试用例:" + colorReset)
		for _, test := range suite.Tests {
			if !test.Passed && !test.Skipped {
				fmt.Printf("    - %s\n", test.Name)
			}
		}
		fmt.Println()
	}

	fmt.Println(colorCyan + "════════════════════════════════════════════════════════════" + colorReset)
	r.lastLineLen = 0
}

func (r *ConsoleReporter) ReportAssertionDetails(assertion models.Assertion) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if assertion.Passed {
		return
	}

	fmt.Println()
	fmt.Println(colorRed + "  断言失败详情:" + colorReset)
	fmt.Printf("    断言名称: %s\n", assertion.Name)
	if assertion.Description != "" {
		fmt.Printf("    描述: %s\n", assertion.Description)
	}
	fmt.Println()
	fmt.Println("    " + colorGreen + "期望值:" + colorReset)
	fmt.Printf("      %v\n", assertion.Expected)
	fmt.Println()
	fmt.Println("    " + colorRed + "实际值:" + colorReset)
	fmt.Printf("      %v\n", assertion.Actual)
	fmt.Println()

	expectedStr := fmt.Sprintf("%v", assertion.Expected)
	actualStr := fmt.Sprintf("%v", assertion.Actual)
	diff := r.highlightDiff(expectedStr, actualStr)
	if diff != "" {
		fmt.Println("    " + colorYellow + "差异:" + colorReset)
		fmt.Printf("      %s\n", diff)
		fmt.Println()
	}
}

func (r *ConsoleReporter) clearLine() {
	if r.lastLineLen > 0 {
		fmt.Print("\r" + strings.Repeat(" ", r.lastLineLen) + "\r")
	}
}

func (r *ConsoleReporter) highlightDiff(expected, actual string) string {
	if expected == actual {
		return ""
	}

	minLen := len(expected)
	if len(actual) < minLen {
		minLen = len(actual)
	}

	var diffStart int
	for diffStart = 0; diffStart < minLen; diffStart++ {
		if expected[diffStart] != actual[diffStart] {
			break
		}
	}

	if diffStart >= minLen {
		if len(expected) > len(actual) {
			return colorRed + "+ " + expected[minLen:] + colorReset
		}
		return colorRed + "- " + actual[minLen:] + colorReset
	}

	var result strings.Builder
	result.WriteString(expected[:diffStart])
	result.WriteString(colorRed)
	result.WriteString(expected[diffStart:])
	result.WriteString(colorReset)

	return result.String()
}

func (r *ConsoleReporter) Start() {
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			r.mu.Lock()
			r.spinnerIdx = (r.spinnerIdx + 1) % len(r.spinnerChars)
			r.mu.Unlock()
		}
	}()
}
