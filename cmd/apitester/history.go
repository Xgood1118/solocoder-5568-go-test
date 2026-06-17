package main

import (
	"fmt"
	"strings"

	"apitester/internal/models"
	"apitester/internal/report"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	historyLimit int
	historySuite string
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "查看测试历史记录",
	Long:  `查看测试执行历史记录和趋势`,
	Run: func(cmd *cobra.Command, args []string) {
		historyManager := report.NewHistoryManager()
		records, err := historyManager.LoadRecords()
		if err != nil {
			color.Red("❌ 加载历史记录失败: %v", err)
			return
		}

		if len(records) == 0 {
			color.Yellow("⚠️  暂无历史记录")
			return
		}

		if historySuite != "" {
			var filtered []*models.HistoryRecord
			for _, r := range records {
				if strings.Contains(r.SuiteName, historySuite) {
					filtered = append(filtered, r)
				}
			}
			records = filtered
		}

		if historyLimit > 0 && len(records) > historyLimit {
			records = records[:historyLimit]
		}

		color.Cyan("📊 测试历史记录")
		color.Cyan(strings.Repeat("═", 100))
		fmt.Printf("%-20s %-30s %8s %8s %8s %8s %10s %12s\n",
			"时间", "套件", "总数", "通过", "失败", "跳过", "通过率", "耗时")
		color.Cyan(strings.Repeat("─", 100))

		for _, r := range records {
			statusColor := color.GreenString
			if r.Failed > 0 {
				statusColor = color.RedString
			}

			fmt.Printf("%-20s %-30s %8d %8d %8d %8d %10s %12s\n",
				r.Timestamp.Format("2006-01-02 15:04:05"),
				truncateString(r.SuiteName, 30),
				r.Total,
				r.Passed,
				r.Failed,
				r.Skipped,
				statusColor("%.1f%%", r.PassRate),
				fmt.Sprintf("%dms", r.DurationMs))
		}

		trendData := historyManager.GetTrendData(historySuite, 10)
		if len(trendData) > 1 {
			fmt.Println()
			color.Cyan("📈 最近 10 次执行趋势")
			printTrendChart(trendData)
		}
	},
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func printTrendChart(data []*models.HistoryRecord) {
	if len(data) == 0 {
		return
	}

	maxRate := 100.0
	minRate := 0.0

	for i, r := range data {
		barLen := int((r.PassRate - minRate) / (maxRate - minRate) * 50)
		if barLen < 0 {
			barLen = 0
		}
		bar := strings.Repeat("█", barLen)

		statusColor := color.GreenString
		if r.Failed > 0 {
			statusColor = color.RedString
		}

		fmt.Printf("%2d: %s %s\n",
			len(data)-i,
			statusColor(bar),
			fmt.Sprintf("%.1f%% (%dms)", r.PassRate, r.DurationMs))
	}
}

func init() {
	rootCmd.AddCommand(historyCmd)

	historyCmd.Flags().IntVarP(&historyLimit, "limit", "l", 20, "显示最近 N 条记录")
	historyCmd.Flags().StringVarP(&historySuite, "suite", "s", "", "按套件名称过滤")
}
