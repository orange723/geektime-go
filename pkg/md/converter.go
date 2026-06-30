package md

import (
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	mdlib "github.com/JohannesKaufmann/html-to-markdown"
)

const (
	// userAgent 下载图片时使用的 UA，需与 API 客户端保持一致以通过防盗链
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36"
	// imageDownloadTimeout 单张图片下载的超时时间
	imageDownloadTimeout = 30 * time.Second
	// maxConcurrentDownloads 图片并发下载的最大数量
	maxConcurrentDownloads = 5
)

// imgSrcRegexp 匹配 <img> 标签的 src 属性，同时兼容单引号和双引号
var imgSrcRegexp = regexp.MustCompile(`(?i)<img[^>]+src\s*=\s*["']([^"']+)["']`)

// Converter 封装了 HTML 到 Markdown 的转换逻辑
type Converter struct {
	mdConverter *mdlib.Converter
	httpClient  *http.Client
}

// NewConverter 创建一个新的转换器
func NewConverter() *Converter {
	return &Converter{
		mdConverter: mdlib.NewConverter("", true, nil),
		httpClient:  &http.Client{Timeout: imageDownloadTimeout},
	}
}

// localImageName 根据图片 URL 生成稳定且唯一的本地文件名。
// 使用 URL 的 sha1 短哈希作前缀，彻底避免不同图片清洗后重名互相覆盖的问题。
func localImageName(imgURL string) string {
	// 去掉查询参数，提取主文件名与后缀
	clean := imgURL
	if i := strings.IndexByte(clean, '?'); i >= 0 {
		clean = clean[:i]
	}
	base := path.Base(clean)
	ext := path.Ext(base)
	if ext == "" {
		ext = ".jpg" // 默认后缀
	}

	sum := sha1.Sum([]byte(imgURL))
	return fmt.Sprintf("%x%s", sum[:6], ext)
}

// ProcessImages 下载 HTML 中的图片到本地，并返回替换为本地路径后的 HTML。
// 图片并发下载（带并发上限），相同 URL 只下载一次。
func (c *Converter) ProcessImages(html, outputDir, articleTitle string) (string, error) {
	imagesDir := filepath.Join(outputDir, "images")
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		return html, err
	}

	matches := imgSrcRegexp.FindAllStringSubmatch(html, -1)

	// 收集去重后的图片 URL 及其对应的本地文件名
	urlToName := make(map[string]string)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		imgURL := match[1]
		// 跳过已是本地相对路径或 data URI 的情况
		if imgURL == "" || strings.HasPrefix(imgURL, "data:") || strings.HasPrefix(imgURL, "images/") {
			continue
		}
		if _, ok := urlToName[imgURL]; !ok {
			urlToName[imgURL] = localImageName(imgURL)
		}
	}

	// 并发下载
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrentDownloads)
	var mu sync.Mutex
	var firstErr error

	for imgURL, imgName := range urlToName {
		localPath := filepath.Join(imagesDir, imgName)
		// 已存在则跳过下载
		if _, err := os.Stat(localPath); err == nil {
			continue
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(u, dst string) {
			defer wg.Done()
			defer func() { <-sem }()

			if err := c.downloadImage(u, dst); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}(imgURL, localPath)
	}
	wg.Wait()

	// 精确替换：仅替换 src 属性中的 URL，避免误伤其它文本或子串。
	result := imgSrcRegexp.ReplaceAllStringFunc(html, func(tag string) string {
		sub := imgSrcRegexp.FindStringSubmatch(tag)
		if len(sub) < 2 {
			return tag
		}
		name, ok := urlToName[sub[1]]
		if !ok {
			return tag
		}
		relPath := path.Join("images", name)
		return strings.Replace(tag, sub[1], relPath, 1)
	})

	return result, firstErr
}

// downloadImage 下载单张图片到指定路径，写入临时文件成功后再重命名，避免半截文件。
func (c *Converter) downloadImage(imgURL, dst string) error {
	req, err := http.NewRequest(http.MethodGet, imgURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Referer", "https://time.geekbang.org")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载图片失败 %s (状态码: %d)", imgURL, resp.StatusCode)
	}

	tmp := dst + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}

	return os.Rename(tmp, dst)
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
