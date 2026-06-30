package cmd

import (
	"fmt"
	"geektime-go/pkg/geektime"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var testMode bool

// downloadCmd 代表下载命令
var downloadCmd = &cobra.Command{
	Use:   "download [课程ID]",
	Short: "通过课程 ID 下载课程内容",
	Long:  `获取指定极客时间课程的所有文章，并将其本地保存为原始 JSON。`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cid := strings.TrimSpace(args[0])
		cookie := viper.GetString("cookie")
		if cookie == "" {
			fmt.Println("错误: 未找到 Cookie。请先执行 'geektime-go login [您的Cookie]' 进行登录。")
			return
		}

		cidInt, err := strconv.Atoi(cid)
		if err != nil {
			fmt.Printf("错误: 课程 ID 必须是数字，收到的是 %q。请从课程详情页 URL 中获取数字 ID。\n", cid)
			return
		}

		client := geektime.NewClient(cookie)
		fmt.Printf("正在获取课程 ID 为 %s 的内容...\n", cid)

		infoJson, err := client.GetColumnInfo(cidInt)
		if err != nil {
			fmt.Printf("获取课程基本信息失败: %v\n", err)
			checkCookieExpiration(err)
			return
		}
		courseTitle := client.ParseColumnTitle(infoJson)
		fmt.Printf("找到课程: 【%s】\n", courseTitle)

		articles, err := client.GetArticles(cid)
		if err != nil {
			fmt.Printf("获取课程文章列表失败: %v\n", err)
			checkCookieExpiration(err)
			return
		}

		fmt.Printf("共包含 %d 篇文章\n", len(articles))

		if testMode {
			if len(articles) > 0 {
				fmt.Printf("\n[测试模式] 第一篇文章名称: %s (ID: %s)\n", articles[0]["title"], articles[0]["id"])
			}
			return
		}

		outputDir := geektime.SanitizeFileName(courseTitle)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			fmt.Printf("创建目录失败: %v\n", err)
			return
		}

		for i, article := range articles {
			title := article["title"]
			id := article["id"]

			safeTitle := geektime.SanitizeFileName(title)
			fileName := fmt.Sprintf("%02d_%s.json", i+1, safeTitle)
			filePath := filepath.Join(outputDir, fileName)

			if _, err := os.Stat(filePath); err == nil {
				fmt.Printf("[%d/%d] 跳过已存在: %s\n", i+1, len(articles), title)
				continue
			}

			fmt.Printf("[%d/%d] 正在下载: %s...", i+1, len(articles), title)

			detail, err := client.GetArticle(id)
			if err != nil {
				fmt.Printf("失败: %v\n", err)
				checkCookieExpiration(err)
				if strings.Contains(err.Error(), "451") {
					fmt.Println("\n警告: 触发了频率限制 (451)。建议等待几分钟或更换 IP 后再试。")
					return
				}
				continue
			}

			err = os.WriteFile(filePath, []byte(detail), 0644)
			if err != nil {
				fmt.Printf("保存失败: %v\n", err)
				continue
			}
			fmt.Println("完成")

			sleepTime := time.Duration(2+rand.Intn(4)) * time.Second
			time.Sleep(sleepTime)
		}

		fmt.Printf("\n下载完成！目录: %s\n", outputDir)
	},
}

// checkCookieExpiration 检查错误信息是否暗示 Cookie 已失效
func checkCookieExpiration(err error) {
	msg := err.Error()
	if strings.Contains(msg, "未购买") || strings.Contains(msg, "请先登录") || strings.Contains(msg, "请登录") {
		fmt.Println("\n提示: 收到“未购买”或“未登录”错误。如果您确认已购买该课程，则说明您的 Cookie 已过期，请重新获取并登录。")
	}
}

func init() {
	rootCmd.AddCommand(downloadCmd)
	downloadCmd.Flags().BoolVarP(&testMode, "test", "t", false, "开启测试模式，仅输出第一篇文章名称")
}
