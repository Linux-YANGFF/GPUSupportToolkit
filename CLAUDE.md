# GST 项目工作指南

GST (GPU Support Toolkit) — GPU 日志分析工具，将 GB 级 apitrace 日志结构化，服务于 AI agent 和 FAE/研发。

**主目录**: `gst/`

## 项目文档

所有项目文档在 `docs/` 目录：
- `docs/ai-context/README.md` — 新 Codex/AI 会话优先读取入口
- `docs/OVERVIEW.md` — 项目功能、已实现/未实现需求、技术栈
- `docs/ARCHITECTURE.md` — 架构、目录结构、API 端点、数据模型
- `docs/TESTING.md` — 测试日志、验证标准

## 常用命令

```bash
cd /root/code/GPUSupportToolkit/GPUSupportToolkit/gst
export PATH=/usr/local/go/bin:$PATH
export GOPROXY=https://goproxy.cn,direct

# 编译
go build -o bin/gst-server ./cmd/gst-server
go build -o bin/gst-cli ./cmd/cli

# 运行
GST_LOG_DIR=/root/code/GPUSupportToolkit ./bin/gst-server -port 8080 -browser=false

# 测试
go test ./... -v
go test ./internal/core/... -v

# 代码检查
go vet ./... && go fmt ./...
```

## 注意事项

- Go 1.22，零外部依赖，CGO_ENABLED=0
- 核心模块测试不依赖 GUI
- 格式检测扫描 500 行（兼容 GDB 输出头部的日志）
- `(nil)` 空指针匹配已包含无 `ptr=` 前缀的格式
