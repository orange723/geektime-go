package geektime

import "testing"

// TestSanitizeFileName 验证文件名非法字符被正确替换
func TestSanitizeFileName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{`I/O 模型`, `I_O 模型`},
		{`a:b*c?d"e<f>g|h\i/j`, `a_b_c_d_e_f_g_h_i_j`},
		{`  正常标题  `, `正常标题`},
		{`没有非法字符`, `没有非法字符`},
		{``, ``},
	}
	for _, c := range cases {
		if got := SanitizeFileName(c.in); got != c.want {
			t.Errorf("SanitizeFileName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestParseColumnTitle 验证从课程信息 JSON 中解析标题
func TestParseColumnTitle(t *testing.T) {
	client := NewClient("dummy")
	json := `{"code":0,"data":{"title":"Go语言第一课"}}`
	if got := client.ParseColumnTitle(json); got != "Go语言第一课" {
		t.Errorf("ParseColumnTitle = %q, want %q", got, "Go语言第一课")
	}
}

// TestParseArticles 验证文章列表解析
func TestParseArticles(t *testing.T) {
	client := NewClient("dummy")
	json := `{"data":{"list":[{"id":1,"article_title":"开篇词"},{"id":2,"article_title":"第一讲"}]}}`
	articles := client.ParseArticles(json)
	if len(articles) != 2 {
		t.Fatalf("got %d articles, want 2", len(articles))
	}
	if articles[0]["id"] != "1" || articles[0]["title"] != "开篇词" {
		t.Errorf("unexpected first article: %v", articles[0])
	}
	if articles[1]["title"] != "第一讲" {
		t.Errorf("unexpected second article: %v", articles[1])
	}
}

// TestParseArticlesEmpty 验证空列表返回非 nil 空切片
func TestParseArticlesEmpty(t *testing.T) {
	client := NewClient("dummy")
	articles := client.ParseArticles(`{"data":{"list":[]}}`)
	if articles == nil {
		t.Error("expected non-nil slice")
	}
	if len(articles) != 0 {
		t.Errorf("got %d, want 0", len(articles))
	}
}

// TestIsRetriable 验证重试判断逻辑
func TestIsRetriable(t *testing.T) {
	if !isRetriable(0, errTest) {
		t.Error("network error should be retriable")
	}
	if !isRetriable(503, nil) {
		t.Error("5xx should be retriable")
	}
	if isRetriable(451, nil) {
		t.Error("451 should not be retriable")
	}
	if isRetriable(200, nil) {
		t.Error("200 should not be retriable")
	}
}

var errTest = &testError{}

type testError struct{}

func (e *testError) Error() string { return "test error" }
