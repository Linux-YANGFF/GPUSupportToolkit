# GST 日志分析工具 - 技术设计文档

## 1. 概述

### 1.1 目标
快速实现一个 GPU 日志分析工具，支持解析 apitrace 日志和 profile 日志，提供关键字检索、时间段检索、耗时分析等功能。

### 1.2 日志格式

**API Trace 格式** (`*.api.txt`):
```
glBindBuffer: count=491, time=588 us
glBindFramebuffer: count=29, time=25377 us
glDrawElements: count=493, time=11214 us
...
libGL: FPS = 8.9
swapBuffers: 3033 us
423 frame cost 109ms
```

**Profile 格式** (`*.prof.txt`):
```
glBindBuffer: count=491, time=588 us
glBindFramebuffer: count=29, time=25377 us
...
```

两种格式本质相同，都是以 `gl` 开头的 OpenGL API 调用日志。

## 2. 架构设计

### 2.1 分层架构

```
┌─────────────────────────────────────────────┐
│           UI 层 (Fyne GUI)                  │
│  - 主窗口 / 文件选择 / 结果展示 / 导出       │
├─────────────────────────────────────────────┤
│           业务逻辑层                         │
│  - LogParser: 日志解析                       │
│  - SearchEngine: 检索引擎                     │
│  - Analyzer: 统计分析                        │
│  - Exporter: 结果导出                        │
├─────────────────────────────────────────────┤
│           平台层                             │
│  - FileReader: 大文件流式读取                │
│  - PlatformDetector: OS检测 (Kylin/Ubuntu)  │
└─────────────────────────────────────────────┘
```

### 2.2 目录结构

```
gst/
├── cmd/
│   └── gst/main.go           # 程序入口
├── internal/
│   ├── ui/                   # GUI层
│   │   ├── main_window.go
│   │   ├── pages/
│   │   │   ├── home_page.go      # 首页（文件选择）
│   │   │   ├── search_page.go    # 检索页面
│   │   │   ├── frame_page.go     # 帧分析页面
│   │   │   └── shader_page.go     # Shader统计页面
│   │   └── widgets/
│   │       ├── virtual_table.go  # 虚拟表格（大数据）
│   │       └── progress.go       # 进度条
│   ├── core/                  # 业务逻辑层
│   │   ├── parser/
│   │   │   ├── parser.go     # 解析器接口
│   │   │   ├── api_parser.go # API日志解析
│   │   │   └── profile_parser.go # Profile解析
│   │   ├── search/
│   │   │   ├── keyword_search.go  # 关键字检索
│   │   │   └── time_range_search.go # 时间段检索
│   │   ├── analyzer/
│   │   │   ├── frame_analyzer.go  # 帧分析
│   │   │   ├── func_analyzer.go   # 函数统计
│   │   │   └── shader_analyzer.go # Shader统计
│   │   └── exporter/
│   │       └── exporter.go   # 导出功能
│   └── platform/
│       ├── file_reader.go     # 大文件流读
│       └── os_detector.go     # OS检测
├── assets/
│   └── icon.png
├── docs/
├── Makefile
└── README.md
```

### 2.3 核心数据流

```
用户选择文件
    │
    ▼
FileReader 流式读取（避免全量加载）
    │
    ▼
Parser 解析 ─────────────────────┐
    │                            │
    ▼                            ▼
内存中构建索引              写入 .index 文件
    │
    ├── 关键字索引 (倒排)
    ├── 帧索引 (每帧边界)
    └── 时间索引
    │
    ▼
SearchEngine / Analyzer 处理查询
    │
    ▼
结果写入临时文件 / 导出
```

## 3. 核心模块设计

### 3.1 大文件处理策略

**问题**: 日志文件 > 1GB，无法一次性加载到内存

**解决方案**:
1. **流式读取**: 使用 `bufio.Reader` + `scanner` 按行读取
2. **内存索引**: 只保存关键信息（行号索引、帧边界、时间点）
3. **磁盘缓存**: 生成 `.gstidx` 索引文件，二次打开秒开
4. **按需加载**: 检索结果只加载匹配的行，不加载全文件

```go
type LogIndex struct {
    FilePath    string
    FileSize    int64
    FrameStarts []int64    // 每帧在文件中的偏移量
    TimePoints  []TimePoint // 时间索引 {LineNum, TimeUs}
    KeywordPos  map[string][]int64 // 关键字 → 行号列表
}
```

### 3.2 日志解析器

```go
type APILogEntry struct {
    APIName  string
    Count    int
    TimeUs   int64
    LineNum  int
}

type FrameInfo struct {
    FrameNum   int
    StartLine  int
    EndLine    int
    TotalTimeUs int64
    APICalls   []APILogEntry
}

type ParsedLog struct {
    Frames     []FrameInfo
    TotalTimeUs int64
    FPS        float64
}
```

**帧识别逻辑**:
- 特征: `swapBuffers: X us` 或 `frame cost Xms`
- 每遇到一次上述特征，计数器+1，形成一帧

### 3.3 检索功能

#### 3.3.1 关键字检索
- 输入: 关键字列表（支持 AND/OR）
- 输出: 匹配行的文件片段
- 优化: 匹配结果分页（每页1000条）

#### 3.3.2 时间段检索
- 输入: 开始时间(us)、结束时间(us)
- 输出: 该时间段内的所有 API 调用

#### 3.3.3 最耗时帧检索
- 统计: `swapBuffers` 或 `frame cost` 最大的 N 帧
- 输出: TopN 帧的详细调用列表

### 3.4 统计分析

#### 3.4.1 函数耗时统计
```go
type FuncStats struct {
    FuncName    string
    CallCount   int
    TotalTimeUs int64
    AvgTimeUs   int64
}
```

#### 3.4.2 Shader 统计
- 从 `glShaderSource` 或类似调用中提取 shader 信息
- 统计每个 shader 的编译次数和耗时

## 4. UI 设计

### 4.1 页面结构

```
┌────────────────────────────────────────────────────┐
│ [GST Logo]  文件  视图  帮助              [─][□][×] │
├────────────────────────────────────────────────────┤
│ ┌──────────┐                                       │
│ │ 📁 打开文件│                                      │
│ │ ──────────── │                                     │
│ │ 🔍 关键字检索│                                    │
│ │ ⏱️ 时间段检索│                                    │
│ │ 📊 帧分析    │                                    │
│ │ 🎨 Shader   │                                    │
│ │ 📥 导出结果  │                                    │
│ └──────────┘                                       │
├────────────────────────────────────────────────────┤
│                                                    │
│              [当前页面内容区]                       │
│                                                    │
│  - 首页: 文件拖拽区 + 快捷操作                      │
│  - 检索: 搜索框 + 结果表格 + 导出按钮              │
│  - 帧分析: TopN列表 + 详情视图                     │
│                                                    │
├────────────────────────────────────────────────────┤
│ 状态: 就绪  │  文件: example.api.txt (1.2GB)       │
└────────────────────────────────────────────────────┘
```

### 4.2 虚拟表格

对于大数据量（百万行），使用虚拟滚动:

```go
type VirtualTable struct {
    TotalRows    int
    VisibleRows  int
    ScrollOffset int
    RenderRow(i int) []string
}
```

- 只渲染可见区域的行
- 滚动时动态计算可见范围
- 支持快速跳转（跳到第 N 行）

## 5. 导出功能

### 5.1 导出格式

| 格式 | 说明 | 适用场景 |
|:---|:---|:---|
| `.txt` | 纯文本片段 | 直接查看 |
| `.csv` | 表格数据 | Excel分析 |
| `.json` | 结构化数据 | 程序处理 |

### 5.2 导出内容

- **检索结果**: 匹配行的原始内容
- **帧详情**: 该帧所有API调用
- **统计分析**: CSV格式的统计报表

## 6. 平台适配

### 6.1 支持的 OS

| 系统 | 版本 | 架构 |
|:---|:---|:---|
| Ubuntu | 20.04+ | amd64, arm64 |
| Kylin V10 | V10 | amd64 |
| 统信 UOS | - | amd64 |
| 其他国产Linux | - | amd64 |

### 6.2 OS 检测

```go
func DetectOS() string {
    // 读取 /etc/os-release
    // 返回: "ubuntu", "kylin", "uos", "other"
}
```

### 6.3 图形环境

- 主要基于 **Fyne** (Go 原生 GUI)
- 检测 `DISPLAY` 环境变量
- 备选: 命令行模式（无图形环境时）

## 7. 性能指标

| 指标 | 目标 |
|:---|:---|
| 解析速度 | > 100MB/s |
| 内存占用 | < 500MB (解析1GB文件) |
| 检索响应 | < 2s (1GB文件) |
| 索引文件 | < 原始文件5% |

## 8. 实施计划

### Phase 1: 核心功能（1-2周）
- [ ] 项目骨架搭建
- [ ] 流式文件读取
- [ ] API日志解析
- [ ] 关键字检索
- [ ] 图形界面框架

### Phase 2: 增强功能（1周）
- [ ] 时间段检索
- [ ] 帧分析
- [ ] 函数统计
- [ ] 导出功能

### Phase 3: 完善（1周）
- [ ] Shader统计
- [ ] 索引缓存
- [ ] OS适配测试
- [ ] 打包发布
