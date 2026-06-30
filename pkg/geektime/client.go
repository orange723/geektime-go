package geektime

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/imroc/req/v3"
	"github.com/tidwall/gjson"
)

const (
	BaseURL = "https://time.geekbang.org"
	// UserAgent 模拟浏览器的 User-Agent，用于通过极客时间的校验
	UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36"

	// requestTimeout 单次 API 请求的超时时间，避免网络异常导致无限挂起
	requestTimeout = 30 * time.Second
	// maxRetries API 请求遇到可重试错误时的最大重试次数
	maxRetries = 3
	// articlePageSize 文章列表分页时每页拉取的数量
	articlePageSize = 100
)

// illegalFileNameChars 匹配文件名中的非法字符（包级变量，避免重复编译正则）
var illegalFileNameChars = regexp.MustCompile(`[\\/:*?"<>|]`)

// Client 极客时间 API 客户端
type Client struct {
	httpClient *req.Client
	cookie     string
}

// NewClient 创建一个新的极客时间客户端
func NewClient(cookie string) *Client {
	c := req.C().
		SetBaseURL(BaseURL).
		SetTimeout(requestTimeout).
		SetCommonHeader("User-Agent", UserAgent).
		SetCommonHeader("Referer", BaseURL).
		SetCommonHeader("Origin", BaseURL).
		SetCommonHeader("Accept", "application/json, text/plain, */*").
		SetCommonHeader("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8").
		SetCommonHeader("Content-Type", "application/json").
		SetCommonHeader("Cookie", cookie)

	return &Client{
		httpClient: c,
		cookie:     cookie,
	}
}

// isRetriable 判断错误或状态码是否值得重试（网络抖动或 5xx 服务端临时错误）
// 注意：451（频率限制）不重试，避免雪上加霜。
func isRetriable(statusCode int, err error) bool {
	if err != nil {
		return true
	}
	return statusCode >= 500
}

// doPost 执行一次 POST 请求并带指数退避重试，返回响应体字符串。
func (c *Client) doPost(path string, body map[string]interface{}) (string, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避：1s, 2s, 4s ...
			backoff := time.Duration(1<<(attempt-1)) * time.Second
			time.Sleep(backoff)
		}

		resp, err := c.httpClient.R().
			SetContext(context.Background()).
			SetBody(body).
			Post(path)

		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
		}

		if err != nil {
			lastErr = err
			if isRetriable(statusCode, err) {
				continue
			}
			return "", err
		}

		result, cerr := c.checkError(resp)
		if cerr != nil {
			lastErr = cerr
			// 仅对服务端临时错误重试；业务错误（如未购买、451 限流）直接返回。
			if isRetriable(statusCode, nil) {
				continue
			}
			return result, cerr
		}
		return result, nil
	}
	return "", lastErr
}

// checkError 检查响应中的业务错误
func (c *Client) checkError(resp *req.Response) (string, error) {
	if !resp.IsSuccess() {
		// 尝试解析 451 等非 200 状态码下的 JSON 错误（如果有）
		body := resp.String()
		if gjson.Valid(body) {
			errMsg := gjson.Get(body, "error.0").String()
			if errMsg == "" {
				errMsg = gjson.Get(body, "msg").String()
			}
			if errMsg != "" {
				return body, fmt.Errorf("API 错误: %s (状态码: %d)", errMsg, resp.StatusCode)
			}
		}
		return body, fmt.Errorf("请求失败: %s (状态码: %d)", resp.Status, resp.StatusCode)
	}

	body := resp.String()
	code := gjson.Get(body, "code").Int()
	if code != 0 {
		errMsg := gjson.Get(body, "error.0").String()
		if errMsg == "" {
			errMsg = gjson.Get(body, "msg").String()
		}
		return body, fmt.Errorf("业务错误: %s (代码: %d)", errMsg, code)
	}

	return body, nil
}

// GetColumnInfo 获取课程信息 (v3/column/info)
func (c *Client) GetColumnInfo(productID int) (string, error) {
	return c.doPost("/serv/v3/column/info", map[string]interface{}{
		"product_id":             productID,
		"with_recommend_article": true,
	})
}

// getArticlesPage 获取一页文章列表，prev 为上一页最后一篇文章的 id（0 表示第一页）
func (c *Client) getArticlesPage(cid string, prev int64) (string, error) {
	return c.doPost("/serv/v1/column/articles", map[string]interface{}{
		"cid":    cid,
		"size":   articlePageSize,
		"prev":   prev,
		"order":  "earliest",
		"sample": false,
	})
}

// GetArticles 获取指定课程 ID (cid) 的全部文章信息，自动翻页直到取完。
// 返回每篇文章的 id 与标题。
func (c *Client) GetArticles(cid string) ([]map[string]string, error) {
	var articles []map[string]string
	var prev int64 = 0

	for {
		pageJSON, err := c.getArticlesPage(cid, prev)
		if err != nil {
			return articles, err
		}

		page := c.ParseArticles(pageJSON)
		if len(page) == 0 {
			break
		}
		articles = append(articles, page...)

		// 极客时间用 data.page.more 标记是否还有更多页。
		hasMore := gjson.Get(pageJSON, "data.page.more").Bool()
		if !hasMore && len(page) < articlePageSize {
			break
		}

		// 以本页最后一篇文章的 id 作为下一页游标。
		lastID, perr := parseInt64(page[len(page)-1]["id"])
		if perr != nil || lastID == 0 || lastID == prev {
			// 无法推进游标，避免死循环。
			break
		}
		prev = lastID
	}

	return articles, nil
}

// GetArticle 获取指定文章 ID (id) 的详细内容
func (c *Client) GetArticle(id string) (string, error) {
	return c.doPost("/serv/v1/article", map[string]interface{}{
		"id":                id,
		"include_neighbors": true,
		"is_freelyread":     true,
	})
}

// ParseColumnTitle 从课程信息响应中解析课程标题
func (c *Client) ParseColumnTitle(jsonStr string) string {
	return gjson.Get(jsonStr, "data.title").String()
}

// ParseArticles 解析返回的文章列表，返回包含 ID 和标题的切片
func (c *Client) ParseArticles(jsonStr string) []map[string]string {
	result := gjson.Get(jsonStr, "data.list")
	articles := make([]map[string]string, 0, len(result.Array()))
	for _, item := range result.Array() {
		articles = append(articles, map[string]string{
			"id":    item.Get("id").String(),
			"title": item.Get("article_title").String(),
		})
	}
	return articles
}

// parseInt64 解析字符串为 int64
func parseInt64(s string) (int64, error) {
	var n int64
	_, err := fmt.Sscan(s, &n)
	return n, err
}

// SanitizeFileName 过滤文件名中的非法字符
func SanitizeFileName(name string) string {
	name = illegalFileNameChars.ReplaceAllString(name, "_")
	return strings.TrimSpace(name)
}
