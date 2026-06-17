package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "apitester",
	Short: "API Tester - 功能强大的 API 测试运行器",
	Long: `API Tester 是一个功能强大的 API 测试运行器，支持：
- YAML/JSON 用例，支持 include 复用
- 多种 HTTP 方法和 body 类型
- 5 种认证方式 + token 自动刷新
- 变量系统 + 动态表达式
- setup/teardown + 依赖管理
- 并发执行 + 指数退避重试
- 多种断言 + Extract 提取
- 彩色控制台 + 多种报告格式
- WebSocket/SSE 支持
- OpenAPI 自动生成用例
- 内置 Mock Server
- 数据驱动测试`,
	Version: "1.0.0",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
}
