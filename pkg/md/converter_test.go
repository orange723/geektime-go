package md

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLocalImageName 验证本地图片文件名生成：去查询参数、保留后缀、相同 URL 稳定、不同 URL 不同。
func TestLocalImageName(t *testing.T) {
	u1 := "https://static.geekbang.org/resource/image/a/b/abc.png?wh=100x200"
	n1 := localImageName(u1)
	if !strings.HasSuffix(n1, ".png") {
		t.Errorf("expected .png suffix, got %q", n1)
	}
	if strings.Contains(n1, "?") {
		t.Errorf("query params should be stripped, got %q", n1)
	}
	// 同一 URL 多次生成应稳定
	if localImageName(u1) != n1 {
		t.Error("localImageName should be stable for same URL")
	}
	// 不同 URL 应得到不同文件名（即使原始文件名相同）
	u2 := "https://static.geekbang.org/resource/image/x/y/abc.png?wh=1x1"
	if localImageName(u2) == n1 {
		t.Error("different URLs should produce different names")
	}
	// 无后缀时默认 .jpg
	if got := localImageName("https://example.com/img/noext"); !strings.HasSuffix(got, ".jpg") {
		t.Errorf("expected default .jpg, got %q", got)
	}
}

// TestFormatMarkdown 验证 Markdown 头部格式化
func TestFormatMarkdown(t *testing.T) {
	out := FormatMarkdown("标题", "作者", "正文内容")
	if !strings.HasPrefix(out, "# 标题\n\n") {
		t.Errorf("missing title heading: %q", out)
	}
	if !strings.Contains(out, "> 作者: 作者") {
		t.Errorf("missing author line: %q", out)
	}
	if !strings.HasSuffix(out, "正文内容") {
		t.Errorf("missing body: %q", out)
	}

	// 无作者时不应出现作者行
	noAuthor := FormatMarkdown("T", "", "B")
	if strings.Contains(noAuthor, "作者:") {
		t.Errorf("should not contain author line: %q", noAuthor)
	}
}

// TestConvertHTMLToMarkdown 验证占位符被清理且基本转换工作
func TestConvertHTMLToMarkdown(t *testing.T) {
	c := NewConverter()
	out, err := c.ConvertHTMLToMarkdown("<p>hello[[[read_end]]] world</p>")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "read_end") {
		t.Errorf("placeholder not removed: %q", out)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("content lost: %q", out)
	}
}

// TestProcessImages 验证图片被下载到本地，且 HTML 中的 src 被替换为本地相对路径（单/双引号均支持）。
func TestProcessImages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fake-image-bytes"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	url1 := srv.URL + "/resource/image/aa/bb/pic.png?wh=10x10"
	url2 := srv.URL + "/resource/image/cc/dd/pic.png?wh=20x20" // 同名不同 URL
	html := `<p><img src="` + url1 + `" alt="x"><img src='` + url2 + `'></p>`

	c := NewConverter()
	out, err := c.ProcessImages(html, dir, "测试")
	if err != nil {
		t.Fatalf("ProcessImages error: %v", err)
	}

	// 原始远程 URL 不应再出现
	if strings.Contains(out, srv.URL) {
		t.Errorf("remote URL not replaced: %q", out)
	}
	if !strings.Contains(out, "images/") {
		t.Errorf("local path not present: %q", out)
	}

	// 两张同名不同 URL 的图片应生成两个不同的本地文件
	entries, err := os.ReadDir(filepath.Join(dir, "images"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 downloaded images, got %d", len(entries))
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}
