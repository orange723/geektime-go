# geektime-go

> **声明：本项目代码逻辑、架构设计及文档编写全由 Gemini CLI (AI) 独立完成。**

`geektime-go` 是一个用 Go 语言编写的命令行工具，旨在帮助用户将已购买的极客时间课程内容备份到本地。它支持下载原始 JSON 数据，并能将其转换为包含本地图片的 Markdown 文件，实现真正的离线阅读。

## 1. 核心特性

- **多步处理流程**: 登录、下载、转换逻辑解耦，支持断点续传，保护账号安全。
- **本地图片化**: 自动下载 HTML 中的远程图片并替换为本地相对路径，解决编辑器防盗链导致的图片无法显示问题。
- **鲁棒性设计**: 
    - 自动清理文件名非法字符（如 `I/O` -> `I_O`）。
    - 智能识别 API 频率限制 (451 错误) 并自动熔断。
    - 详细的调试模式 (`--debug`)。
- **纯中文交互**: 所有的帮助信息、错误提示及代码注释均为中文，对开发者友好。

## 2. 快速开始

### 2.1 编译
确保您已安装 Go 1.26+ 环境：
```bash
go build -o geektime-go
```

### 2.2 登录 (配置 Cookie)
从浏览器开发者工具中获取极客时间的 Cookie：
```bash
./geektime-go login "您的Cookie字符串"
```

### 2.3 下载课程 (保存为 JSON)
使用课程的数字 ID 进行下载：
```bash
# CID 可从课程详情页 URL 获取
./geektime-go download 100093501
```

### 2.4 转换为 Markdown (含图片下载)
对下载后的课程目录执行转换：
```bash
./geektime-go convert "Tony Bai · Go语言第一课"
```

## 3. 命令行参数

- `--config`: 指定配置文件路径 (默认为 `~/.geektime-go.yaml`)。
- `--debug`: 开启调试模式，打印完整的 API 原始响应 JSON。
- `download -t`: 测试模式，仅验证 API 是否能获取第一篇文章名称。

---

## 附录：原始需求文档 (GEMINI.md 早期版本参考)

以下为本项目开发初期的原始需求，供架构演进参考：

```markdown
# geektime-go

## 项目描述
以 golang 为编程语言，获取极客时间已经购买的课程 获取课程文章并转换为 markdown 文件的 cli 工具

## 项目实现
api
- 获取课程info: https://time.geekbang.org/serv/v3/column/info
- 获取课程的全部文章信息: https://time.geekbang.org/serv/v1/column/articles
- 获取文章内容: https://time.geekbang.org/serv/v1/article

## 工具
使用以下包实现
- https://github.com/imroc/req
- https://github.com/spf13/viper
- https://github.com/spf13/cobra
- https://github.com/tidwall/gjson

## 协作偏好
- 每个函数，在上面写上中文注释，便于查看
```
