package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"apitester/internal/mock"
	"apitester/internal/parser"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	mockPort int
)

var mockCmd = &cobra.Command{
	Use:   "mock [config-file]",
	Short: "启动 Mock Server",
	Long:  `根据配置文件启动 Mock Server，支持自定义响应状态、延迟、响应体`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		server := mock.NewMockServer(mockPort)

		if len(args) > 0 {
			color.Cyan("📄 正在加载 Mock 配置: %s", args[0])

			parser := parser.NewParser()
			suite, err := parser.LoadSuite(args[0])
			if err != nil {
				color.Red("❌ 加载配置失败: %v", err)
				return
			}

			if len(suite.MockRules) > 0 {
				server.AddRules(suite.MockRules)
				color.Green("✅ 已加载 %d 条 Mock 规则", len(suite.MockRules))
			}
		}

		if err := server.Start(); err != nil {
			color.Red("❌ 启动 Mock Server 失败: %v", err)
			return
		}

		color.Cyan("🎭 Mock Server 已启动")
		color.Cyan("   地址: http://localhost:%d", mockPort)
		color.Cyan("   规则数量: %d", len(server.Rules()))
		color.Cyan("   按 Ctrl+C 停止服务")
		fmt.Println()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		color.Yellow("\n⏹️  正在停止 Mock Server...")
		server.Stop()
		color.Green("✅ Mock Server 已停止")
	},
}

func init() {
	rootCmd.AddCommand(mockCmd)

	mockCmd.Flags().IntVarP(&mockPort, "port", "p", 8081, "Mock Server 端口")
}
