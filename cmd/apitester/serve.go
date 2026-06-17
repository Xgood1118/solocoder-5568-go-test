package main

import (
	"fmt"
	"net/http"

	"apitester/internal/report"
	"apitester/web"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	servePort int
	serveHost string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "启动 Web 报告服务",
	Long:  `启动 Web 服务器查看测试报告和历史趋势，默认端口 8080`,
	Run: func(cmd *cobra.Command, args []string) {
		historyManager := report.NewHistoryManager()
		server := web.NewServer(historyManager)

		addr := fmt.Sprintf("%s:%d", serveHost, servePort)
		color.Cyan("🌐 Web 报告服务已启动")
		color.Cyan("   地址: http://%s", addr)
		color.Cyan("   首页: http://%s/", addr)
		color.Cyan("   API:  http://%s/api/history", addr)
		color.Cyan("   按 Ctrl+C 停止服务")
		fmt.Println()

		if err := server.Run(addr); err != nil && err != http.ErrServerClosed {
			color.Red("❌ 服务启动失败: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "服务端口")
	serveCmd.Flags().StringVar(&serveHost, "host", "0.0.0.0", "绑定主机地址")
}
