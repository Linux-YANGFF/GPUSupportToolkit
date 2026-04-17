# GST - GPU Support Toolkit

GPU 日志分析工具，支持 apitrace/profile 日志的拆分、解析、检索和导出。

## 功能特性

| 功能 | 说明 |
|:---|:---|
| 日志解析 | 支持 apitrace 和 profile 格式的大文件 (>1GB) 流式解析 |
| 关键字检索 | 多关键字 AND 匹配，不区分大小写 |
| 时间段检索 | 按时间范围检索 API 调用 |
| 帧分析 | 找出最耗时的 Top N 帧 |
| 函数统计 | 统计每类函数的调用次数和总耗时 |
| Shader 统计 | 统计 Shader 相关调用 |
| 多格式导出 | 支持 TXT/CSV/JSON 格式导出 |
| 跨平台 | 支持 Ubuntu/Kylin/UOS 等 Linux 系统 |
| 图形界面 | 基于 Fyne 的 GUI 操作 |

## 项目结构

```
gst/
├── cmd/
│   └── gst/main.go           # 程序入口
├── internal/
│   ├── core/
│   │   ├── parser/           # 日志解析器
│   │   ├── search/           # 检索引擎
│   │   ├── analyzer/         # 分析器 (帧/函数/Shader)
│   │   ├── exporter/         # 导出功能
│   │   └── types.go          # 核心数据类型
│   ├── platform/
│   │   ├── file_reader.go    # 大文件流式读取
│   │   └── os_detector.go    # OS 检测
│   └── ui/                   # Fyne GUI
│       ├── pages/            # 页面 (home/search/frame/shader)
│       └── widgets/           # 组件
├── assets/                   # 资源文件
└── Makefile
```

## 快速开始

### 编译

```bash
cd gst

# 安装 GUI 依赖 (Ubuntu)
sudo apt install -y libgl1-mesa-dev libx11-dev libxkbfile-dev libxcursor-dev libxi-dev libxrandr-dev

# 编译
export GOPROXY=https://goproxy.cn,direct
export CGO_ENABLED=1
go build -o bin/gst ./cmd/gst
```

### 运行

```bash
# 运行 GUI
./bin/gst

# 或开发模式
make dev
```

### 测试

```bash
go test ./internal/core/... -v
```

## 日志格式

GST 支持以下日志格式：

```
glBindBuffer: count=491, time=588 us
glBindFramebuffer: count=29, time=25377 us
glDrawElements: count=493, time=11214 us
...
swapBuffers: 3033 us
423 frame cost 109ms
```

- `glXxx: count=X, time=Y us` - API 调用记录
- `swapBuffers: X us` - 帧交换标志
- `frame cost Xms` - 帧耗时记录

## 技术栈

| 技术 | 说明 |
|:---|:---|
| Go 1.18+ | 开发语言 |
| Fyne v2.4.0 | 跨平台 GUI 框架 |
| bufio.Scanner | 大文件流式读取 |
| 正则表达式 | 日志解析和检索 |

## 文档

- [产品需求文档](docs/PRODUCT.md) - 功能需求和用户故事
- [技术设计文档](docs/DESIGN.md) - 架构设计和实现细节

## 版本

- v0.1.0 - 初始版本，包含核心功能

## License

MIT
