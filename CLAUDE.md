# GST 项目工作指南

## 项目概述

GST (GPU Support Toolkit) 是一个基于 Go + Fyne 的 GPU 日志分析工具，用于解析和分析 apitrace/profile 日志。

**主目录**: `/root/code/GPUSupportToolkit/GPUSupportToolkit/gst`

## 常用命令

```bash
cd /root/code/GPUSupportToolkit/GPUSupportToolkit/gst

# 下载依赖
export GOPROXY=https://goproxy.cn,direct
go mod tidy

# 编译 GUI
export CGO_ENABLED=1
go build -o bin/gst ./cmd/gst

# 运行开发版本
make dev

# 运行测试
go test ./... -v
go test ./internal/core/... -v

# 代码检查
go vet ./...
go fmt ./...
```

## 模块说明

| 模块 | 路径 | 职责 |
|:---|:---|:---|
| parser | `internal/core/parser/` | 解析 apitrace/profile 日志 |
| search | `internal/core/search/` | 关键字和时间段检索 |
| analyzer | `internal/core/analyzer/` | 帧分析、函数统计、Shader统计 |
| exporter | `internal/core/exporter/` | 导出 TXT/CSV/JSON |
| platform | `internal/platform/` | 文件读取、OS检测 |
| ui | `internal/ui/` | Fyne 图形界面 |

## 开发流程

1. **修改代码**
2. **运行测试**: `go test ./internal/core/... -v`
3. **编译验证**: `go build ./...`
4. **提交**: `git add . && git commit -m "描述"`

## 测试日志

示例日志位于: `../exmple_log/`

- `1frame_demo_api.txt` - API 日志示例
- `1frame_profile_demo.txt` - Profile 日志示例

## 注意事项

- GUI 编译需要 X11 开发库 (`libx11-dev` 等)
- WSL 环境下可能无法编译 GUI，需在原生 Linux
- 核心模块测试不依赖 GUI，可在任何环境运行
