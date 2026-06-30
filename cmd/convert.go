package cmd

import (
	"fmt"
	"geektime-go/pkg/md"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"
)

// convertCmd 代表转换命令
var convertCmd = &cobra.Command{
	Use:   "convert [目录路径]",
	Short: "将已下载的 JSON 文件转换为 Markdown 并下载图片",
	Long:  `扫描指定目录下的所有 .json 文件，解析极客时间文章内容，下载图片并将其转换为本地 .md 文件。`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dir := args[0]

		// 检查目录是否存在
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			fmt.Printf("错误: 目录 %s 不存在\n", dir)
			return
		}

		// 获取目录下所有 JSON 文件
		files, err := os.ReadDir(dir)
		if err != nil {
			fmt.Printf("读取目录失败: %v\n", err)
			return
		}

		fmt.Printf("正在转换目录 %s 中的文章并处理图片...\n", dir)

		// 初始化转换器
		converter := md.NewConverter()

		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
				jsonPath := filepath.Join(dir, file.Name())
				mdName := strings.TrimSuffix(file.Name(), ".json") + ".md"
				mdPath := filepath.Join(dir, mdName)

				fmt.Printf("正在处理: %s...", file.Name())

				// 读取 JSON
				data, err := os.ReadFile(jsonPath)
				if err != nil {
					fmt.Printf("读取失败: %v\n", err)
					continue
				}

				// 解析关键字段
				jsonStr := string(data)
				title := gjson.Get(jsonStr, "data.article_title").String()
				author := gjson.Get(jsonStr, "data.author_name").String()
				htmlContent := gjson.Get(jsonStr, "data.article_content").String()

				if htmlContent == "" {
					fmt.Println("未找到正文内容，跳过。")
					continue
				}

				// 1. 处理图片下载并替换 URL
				htmlContent, err = converter.ProcessImages(htmlContent, dir, title)
				if err != nil {
					fmt.Printf("图片处理出错: %v (继续转换文本)...", err)
				}

				// 2. 将处理后的 HTML 转换为 Markdown
				markdown, err := converter.ConvertHTMLToMarkdown(htmlContent)
				if err != nil {
					fmt.Printf("转换失败: %v\n", err)
					continue
				}

				// 3. 格式化最终文档
				finalContent := md.FormatMarkdown(title, author, markdown)

				// 4. 写入文件 (强制重写以应用图片更改)
				err = os.WriteFile(mdPath, []byte(finalContent), 0644)
				if err != nil {
					fmt.Printf("保存失败: %v\n", err)
					continue
				}
				fmt.Println("完成")
			}
		}

		fmt.Println("\n所有转换任务已完成！图片保存在 images/ 目录下。")
	},
}

func init() {
	// 将转换命令添加到根命令
	rootCmd.AddCommand(convertCmd)
}
