package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"apitester/internal/engine"
	"apitester/internal/models"
	"apitester/internal/report"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	runEnvironment  string
	runTags         []string
	runSkipTags     []string
	runFilter       string
	runDryRun       bool
	runConcurrency  int
	runRetries      int
	runTimeout      int
	runReportFormat string
	runReportOutput string
	runMockMode     bool
	runMockPort     int
)

var runCmd = &cobra.Command{
	Use:   "run [files...]",
	Short: "运行测试用例",
	Long:  `运行指定的测试用例文件，支持 YAML 和 JSON 格式`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		opts := models.ExecutionOptions{
			Environment:   runEnvironment,
			Tags:          runTags,
			SkipTags:      runSkipTags,
			Filter:        runFilter,
			DryRun:        runDryRun,
			Concurrency:   runConcurrency,
			Retries:       runRetries,
			Timeout:       runTimeout,
			ReportFormats: strings.Split(runReportFormat, ","),
			ReportOutput:  runReportOutput,
			MockMode:      runMockMode,
			MockPort:      runMockPort,
		}

		if opts.DryRun {
			color.Yellow("⚠️  Dry Run 模式：只校验用例，不实际发送请求")
			fmt.Println()
		}

		s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.Suffix = "  正在加载用例..."
		s.Start()

		testEngine := engine.NewTestEngine(opts)

		s.Stop()

		var allResults []*models.SuiteResult

		for i, file := range args {
			color.Cyan("📦 测试套件 %d/%d: %s", i+1, len(args), file)
			fmt.Println()

			s = spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			s.Suffix = "  正在执行测试..."
			s.Start()

			result, err := testEngine.RunSuite(file)

			s.Stop()

			if err != nil {
				color.Red("❌ 执行失败: %v", err)
				continue
			}

			allResults = append(allResults, result)

			consoleReporter := report.NewConsoleReporter()
			consoleReporter.ReportSuiteResult(result)
			fmt.Println()
		}

		if len(allResults) > 0 {
			generateReports(allResults, opts)

			historyManager := report.NewHistoryManager()
			for _, r := range allResults {
				record := &models.HistoryRecord{
					ID:          fmt.Sprintf("run-%d", time.Now().UnixNano()),
					Timestamp:   r.StartTime,
					SuiteName:   r.SuiteName,
					Total:       r.Total,
					Passed:      r.Passed,
					Failed:      r.Failed,
					Skipped:     r.Skipped,
					PassRate:    r.PassRate,
					DurationMs:  r.Duration.Milliseconds(),
					Environment: opts.Environment,
				}
				historyManager.SaveRecord(record)
			}

			printOverallSummary(allResults)
		}
	},
}

func generateReports(results []*models.SuiteResult, opts models.ExecutionOptions) {
	if opts.ReportOutput == "" {
		return
	}

	for _, format := range opts.ReportFormats {
		format = strings.TrimSpace(format)
		if format == "" {
			continue
		}

		switch format {
		case "html":
			htmlReporter := report.NewHTMLReporter()
			output := opts.ReportOutput
			if !strings.HasSuffix(output, ".html") {
				output += ".html"
			}
			for i, result := range results {
				fileOutput := output
				if len(results) > 1 {
					fileOutput = strings.Replace(output, ".html", fmt.Sprintf("_%d.html", i+1), 1)
				}
				if err := htmlReporter.Generate(result, fileOutput); err != nil {
					color.Red("❌ 生成 HTML 报告失败: %v", err)
				} else {
					color.Green("✅ HTML 报告已生成: %s", fileOutput)
				}
			}
		case "junit", "xml":
			junitReporter := report.NewJUnitReporter()
			output := opts.ReportOutput
			if !strings.HasSuffix(output, ".xml") {
				output += ".xml"
			}
			for i, result := range results {
				fileOutput := output
				if len(results) > 1 {
					fileOutput = strings.Replace(output, ".xml", fmt.Sprintf("_%d.xml", i+1), 1)
				}
				if err := junitReporter.Generate(result, fileOutput); err != nil {
					color.Red("❌ 生成 JUnit 报告失败: %v", err)
				} else {
					color.Green("✅ JUnit 报告已生成: %s", fileOutput)
				}
			}
		case "json":
			jsonReporter := report.NewJSONReporter()
			output := opts.ReportOutput
			if !strings.HasSuffix(output, ".json") {
				output += ".json"
			}
			for i, result := range results {
				fileOutput := output
				if len(results) > 1 {
					fileOutput = strings.Replace(output, ".json", fmt.Sprintf("_%d.json", i+1), 1)
				}
				if err := jsonReporter.Generate(result, fileOutput); err != nil {
					color.Red("❌ 生成 JSON 报告失败: %v", err)
				} else {
					color.Green("✅ JSON 报告已生成: %s", fileOutput)
				}
			}
		}
	}
}

func printOverallSummary(results []*models.SuiteResult) {
	total := 0
	passed := 0
	failed := 0
	skipped := 0
	totalDuration := time.Duration(0)

	for _, r := range results {
		total += r.Total
		passed += r.Passed
		failed += r.Failed
		skipped += r.Skipped
		totalDuration += r.Duration
	}

	passRate := 0.0
	if total > 0 {
		passRate = float64(passed) / float64(total) * 100
	}

	fmt.Println()
	color.Cyan("═══════════════════════════════════════════════════")
	color.Cyan("                    📊 总体统计")
	color.Cyan("═══════════════════════════════════════════════════")
	fmt.Printf("  测试套件: %d 个\n", len(results))
	fmt.Printf("  用例总数: %d\n", total)
	fmt.Printf("  %s: %d\n", color.GreenString("✅ 通过"), passed)
	fmt.Printf("  %s: %d\n", color.RedString("❌ 失败"), failed)
	fmt.Printf("  %s: %d\n", color.YellowString("⏭️  跳过"), skipped)
	fmt.Printf("  通过率: %.1f%%\n", passRate)
	fmt.Printf("  总耗时: %s\n", totalDuration)
	color.Cyan("═══════════════════════════════════════════════════")

	if failed > 0 {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVarP(&runEnvironment, "env", "e", "", "环境名称")
	runCmd.Flags().StringSliceVarP(&runTags, "tags", "t", []string{}, "要运行的标签，逗号分隔")
	runCmd.Flags().StringSliceVar(&runSkipTags, "skip-tags", []string{}, "要跳过的标签，逗号分隔")
	runCmd.Flags().StringVarP(&runFilter, "filter", "f", "", "过滤表达式，如 status==200&&latency<500")
	runCmd.Flags().BoolVar(&runDryRun, "dry-run", false, "只校验用例，不实际发送请求")
	runCmd.Flags().IntVarP(&runConcurrency, "concurrency", "c", 5, "并发执行的 worker 数量")
	runCmd.Flags().IntVarP(&runRetries, "retries", "r", 0, "失败重试次数")
	runCmd.Flags().IntVar(&runTimeout, "timeout", 30, "请求超时时间（秒）")
	runCmd.Flags().StringVar(&runReportFormat, "report-format", "html", "报告格式：html,junit,json，逗号分隔")
	runCmd.Flags().StringVarP(&runReportOutput, "report-output", "o", "", "报告输出路径")
	runCmd.Flags().BoolVar(&runMockMode, "mock", false, "使用内置 Mock Server")
	runCmd.Flags().IntVar(&runMockPort, "mock-port", 8081, "Mock Server 端口")
}
