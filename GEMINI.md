# geektime-go 项目文档

## 1. 项目概述
`geektime-go` 是一个基于 Golang 的命令行工具 (CLI)，用于下载已购买的极客时间课程内容，并将其转换为包含本地图片的 Markdown 文件，方便离线阅读和归档。

## 2. 核心功能与工作流
工具采用标准的分步处理流程：
1. **登录 (Login)**: 将 Cookie 保存至本地配置 (`~/.geektime-go.yaml`)。
2. **下载 (Download)**: 获取课程元数据及文章详情，保存为原始 JSON 文件（支持断点续传）。
3. **转换 (Convert)**: 扫描 JSON 文件，下载文内图片，生成 Markdown 文档。

## 3. 技术栈
- **CLI 框架**: `github.com/spf13/cobra`
- **配置管理**: `github.com/spf13/viper`
- **HTTP 客户端**: `github.com/imroc/req/v3` (处理 API 请求与防盗链)
- **JSON 解析**: `github.com/tidwall/gjson`
- **Markdown 转换**: `github.com/JohannesKaufmann/html-to-markdown`

## 4. 项目结构
```text
├── cmd/                # 命令行入口与交互逻辑
│   ├── root.go         # 全局 Flag (--debug) 与配置加载
│   ├── login.go        # 登录命令，处理 Cookie 存储
│   ├── download.go     # 下载命令，处理 API 调用与 JSON 保存
│   └── convert.go      # 转换命令，处理 HTML转MD 与 图片下载
├── pkg/                # 核心业务逻辑库
│   ├── geektime/       # 极客时间 API 客户端 (Client, API 封装)
│   └── md/             # Markdown 转换器 (图片清洗、本地化替换)
└── main.go             # 程序入口
```

## 5. API 协议参考
> 关键请求头：`Referer` 和 `Origin` 必须设置为 `https://time.geekbang.org` 以通过校验。

### 5.1 获取课程详情 (v3)
用于获取课程标题等元数据。
- **URL**: `https://time.geekbang.org/serv/v3/column/info`
- **Method**: POST
- **Payload**:
  ```json
  { "product_id": 100093501, "with_recommend_article": true }
  ```

### 5.2 获取文章列表 (v1)
- **URL**: `https://time.geekbang.org/serv/v1/column/articles`
- **Method**: POST
- **Payload**:
  ```json
  { "cid": "100093501", "size": 500, "prev": 0, "order": "earliest", "sample": false }
  ```

### 5.3 获取文章详情 (v1)
- **URL**: `https://time.geekbang.org/serv/v1/article`
- **Method**: POST
- **Payload**:
  ```json
  { "id": "835197", "include_neighbors": true, "is_freelyread": true }
  ```

## 6. 开发与协作规范
- **注释风格**: 所有 Exported 函数及关键逻辑块必须包含**中文注释**。
- **错误处理**: 
  - 遇到 `451` 错误时，视为频率限制，需提示用户等待。
  - 遇到“未购买”相关错误时，提示检查 Cookie 有效性。
- **文件处理**:
  - 文件名必须经过 `SanitizeFileName` 处理，去除 `/`、`|` 等非法字符。
  - 图片下载时需清洗 URL 参数（如 `?wh=...`），确保本地文件名合法。
- **调试**: 使用 `--debug` 标志打印原始 API 响应，便于排查字段变动。
