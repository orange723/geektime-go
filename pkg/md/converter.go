package md

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
)

// Converter 封装了 HTML 到 Markdown 的转换逻辑
type Converter struct {
	mdConverter *md.Converter
}

// NewConverter 创建一个新的转换器
func NewConverter() *Converter {
	return &Converter{
		mdConverter: md.NewConverter("", true, nil),
	}
}

// ProcessImages 下载 HTML 中的图片到本地，并返回替换为本地路径后的 HTML
func (c *Converter) ProcessImages(html, outputDir, articleTitle string) (string, error) {
	// 创建图片存放目录
	imagesDir := filepath.Join(outputDir, "images")
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		return html, err
	}

	// 匹配 <img> 标签的 src 属性
	re := regexp.MustCompile(`<img[^>]+src="([^">]+)"`)
	matches := re.FindAllStringSubmatch(html, -1)

	client := &http.Client{}

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		imgURL := match[1]
		
		// 1. 提取原始文件名并彻底去掉查询参数（?wh=...）
		parts := strings.Split(imgURL, "/")
		imgNameWithQuery := parts[len(parts)-1]
		imgName := strings.Split(imgNameWithQuery, "?")[0] // 仅保留主文件名

		if !strings.Contains(imgName, ".") {
			imgName += ".jpg" // 默认后缀
		}
		
		// 2. 为了防止重名，加上路径中的一部分
		if len(parts) > 2 {
			// 取 URL 路径的倒数第二部分作为前缀
			prefix := strings.Split(parts[len(parts)-2], "?")[0]
			imgName = prefix + "_" + imgName
		}

		localPath := filepath.Join(imagesDir, imgName)
		relPath := filepath.Join("images", imgName)

		// 3. 下载图片（如果本地不存在）
		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			req, _ := http.NewRequest("GET", imgURL, nil)
			req.Header.Set("Referer", "https://time.geekbang.org")
			req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36")

			resp, err := client.Do(req)
			if err == nil && resp.StatusCode == 200 {
				out, err := os.Create(localPath)
				if err == nil {
					io.Copy(out, resp.Body)
					out.Close()
				}
				resp.Body.Close()
			}
		}

		// 4. 替换 HTML 中的远程链接为清理后的本地相对链接
		// 这里必须用全匹配替换，确保带有参数的原始 URL 被正确替换为干净的本地路径
		html = strings.ReplaceAll(html, imgURL, relPath)
	}

	return html, nil
}

// ConvertHTMLToMarkdown 将 HTML 转换为 Markdown
func (c *Converter) ConvertHTMLToMarkdown(html string) (string, error) {
	markdown, err := c.mdConverter.ConvertString(html)
	if err != nil {
		return "", fmt.Errorf("HTML 转换失败: %v", err)
	}

	// 清理占位符
	markdown = strings.ReplaceAll(markdown, "[[[read_end]]]", "")
	
	return markdown, nil
}

// FormatMarkdown 为文章添加标题和元数据
func FormatMarkdown(title, author, content string) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("# %s\n\n", title))
	if author != "" {
		builder.WriteString(fmt.Sprintf("> 作者: %s\n\n", author))
	}
	builder.WriteString(content)
	return builder.String()
}
