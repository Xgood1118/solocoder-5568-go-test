package main

import (
	"fmt"

	"apitester/internal/openapi"
	"apitester/internal/parser"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	generateOutput string
	generateFormat string
)

var generateCmd = &cobra.Command{
	Use:   "generate [source]",
	Short: "从 OpenAPI 文档生成测试用例",
	Long:  `从 OpenAPI 3.0 文档（文件或 URL）自动生成测试用例模板`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		source := args[0]

		color.Cyan("📄 正在解析 OpenAPI 文档: %s", source)
		fmt.Println()

		generator := openapi.NewOpenAPIGenerator()

		var suite interface{}
		var err error

		if source[:4] == "http" {
			suite, err = generator.GenerateFromURL(source)
		} else {
			suite, err = generator.GenerateFromFile(source)
		}

		if err != nil {
			color.Red("❌ 生成失败: %v", err)
			return
		}

		loader := parser.NewParser()

		if generateOutput != "" {
			var outputData []byte
			if generateFormat == "json" {
				outputData, err = loader.ToJSON(suite)
			} else {
				outputData, err = loader.ToYAML(suite)
			}

			if err != nil {
				color.Red("❌ 序列化失败: %v", err)
				return
			}

			if err := loader.WriteFile(generateOutput, outputData); err != nil {
				color.Red("❌ 写入文件失败: %v", err)
				return
			}

			color.Green("✅ 用例已生成到: %s", generateOutput)
		} else {
			var outputData []byte
			if generateFormat == "json" {
				outputData, err = loader.ToJSON(suite)
			} else {
				outputData, err = loader.ToYAML(suite)
			}

			if err != nil {
				color.Red("❌ 序列化失败: %v", err)
				return
			}

			fmt.Println(string(outputData))
		}

		s, _ := suite.(interface{ GetTestCases() int })
		if s != nil {
			color.Green("✅ 共生成 %d 个测试用例", s.GetTestCases())
		}
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().StringVarP(&generateOutput, "output", "o", "", "输出文件路径，默认输出到控制台")
	generateCmd.Flags().StringVarP(&generateFormat, "format", "f", "yaml", "输出格式：yaml 或 json")
}
