package geektime

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/imroc/req/v3"
	"github.com/tidwall/gjson"
)

const (
	BaseURL = "https://time.geekbang.org"
)

// Client 极客时间 API 客户端
type Client struct {
	httpClient *req.Client
	cookie     string
}

// NewClient 创建一个新的极客时间客户端
func NewClient(cookie string) *Client {
	c := req.C().
		SetBaseURL(BaseURL).
		SetCommonHeader("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36").
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
func (c *Client) GetColumnInfo(productID interface{}) (string, error) {
	resp, err := c.httpClient.R().
		SetBody(map[string]interface{}{
			"product_id":             productID,
			"with_recommend_article": true,
		}).
		Post("/serv/v3/column/info")

	if err != nil {
		return "", err
	}

	return c.checkError(resp)
}

// GetArticles 获取指定课程 ID (cid) 的全部文章信息
func (c *Client) GetArticles(cid string) (string, error) {
	resp, err := c.httpClient.R().
		SetBody(map[string]interface{}{
			"cid":    cid,
			"size":   500,
			"prev":   0,
			"order":  "earliest",
			"sample": false,
		}).
		Post("/serv/v1/column/articles")

	if err != nil {
		return "", err
	}

	return c.checkError(resp)
}

// GetArticle 获取指定文章 ID (id) 的详细内容
func (c *Client) GetArticle(id string) (string, error) {
	resp, err := c.httpClient.R().
		SetBody(map[string]interface{}{
			"id":                id,
			"include_neighbors": true,
			"is_freelyread":     true,
		}).
		Post("/serv/v1/article")

	if err != nil {
		return "", err
	}

	return c.checkError(resp)
}

// ParseColumnTitle 从课程信息响应中解析课程标题
func (c *Client) ParseColumnTitle(jsonStr string) string {
	return gjson.Get(jsonStr, "data.title").String()
}

// ParseArticles 解析返回的文章列表，返回包含 ID 和标题的切片
func (c *Client) ParseArticles(jsonStr string) []map[string]string {
	result := gjson.Get(jsonStr, "data.list")
	var articles []map[string]string
	for _, item := range result.Array() {
		articles = append(articles, map[string]string{
			"id":    item.Get("id").String(),
			"title": item.Get("article_title").String(),
		})
	}
	return articles
}

// SanitizeFileName 过滤文件名中的非法字符
func SanitizeFileName(name string) string {
	re := regexp.MustCompile(`[\\/:*?"<>|]`)
	name = re.ReplaceAllString(name, "_")
	return strings.TrimSpace(name)
}
