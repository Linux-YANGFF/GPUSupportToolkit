# GST 架构

## 分层结构

```
┌─ 表现层 ─────────────────────────────────────────┐
│  gst-server (HTTP API)  │  gst-cli (命令行)      │
│  Vue 3 Web UI           │                        │
├─ Handler 层 ─────────────────────────────────────┤
│  handlers.go — 单例 Handler + sync.RWMutex       │
│  15 个 REST 端点，持有当前 ParsedLog              │
├─ 核心引擎层 ─────────────────────────────────────┤
│  parser/    3 个解析器 (API/Profile/RawTrace)     │
│  analyzer/  帧分析/函数统计/Shader/Buffer/DrawCall │
│             纹理分析/综合摘要/瓶颈分类/Trace Inspector│
│  bug/       9 个诊断器 + Registry + GL状态机      │
│  search/    关键字搜索 + 时间范围 + 倒排索引      │
│  exporter/  TXT/CSV/JSON 导出                     │
│  types.go   共享数据类型                          │
├─ 平台层 ─────────────────────────────────────────┤
│  platform/  文件读取/OS检测/slog日志               │
└──────────────────────────────────────────────────┘
```

## 目录结构

```
gst/
├── cmd/
│   ├── gst-server/          # HTTP 服务入口
│   │   ├── main.go          # 路由注册
│   │   └── internal/handlers/  # 所有 API handler
│   ├── cli/                 # CLI 工具
│   └── functionaltest/      # 集成测试
├── internal/
│   ├── core/
│   │   ├── parser/          # 日志解析（3种格式）
│   │   ├── search/          # 检索引擎
│   │   ├── analyzer/        # 分析器：统计、资源、瓶颈、Trace Inspector
│   │   ├── bug/             # Bug诊断（9个检测器）
│   │   ├── exporter/        # 导出
│   │   └── types.go         # 共享类型
│   └── platform/            # 文件/OS/日志
├── web/                     # Vue 3 前端
│   └── src/
│       ├── components/      # 6 个 Vue 组件
│       └── composables/     # useLogAnalysis.ts
└── packaging/               # deb/rpm 打包
```

## 关键设计模式

- **Strategy**: Parser 接口 + 自动检测选择解析器
- **Plugin Registry**: Diagnoser 接口，注册即执行
- **State Machine**: GLStateTracker 按需跟踪 GL 状态
- **Singleton + Mutex**: Handler 持有一个 ParsedLog，读写锁保护

## API 端点一览

| 端点 | 方法 | 功能 |
|------|------|------|
| `/api/log/parse` | POST | 解析日志 |
| `/api/overview` | GET | 综合摘要 |
| `/api/log/frames` | GET | 帧列表（分页） |
| `/api/log/frames/:id` | GET | 帧详情 |
| `/api/log/frames/:id/funcs` | GET | 帧函数统计 |
| `/api/log/search` | GET | 关键字搜索 |
| `/api/log/analyze/top` | GET | Top N 慢帧 |
| `/api/log/analyze/shaders` | GET | Shader 统计 |
| `/api/log/analyze/funcs` | GET | 函数统计 |
| `/api/log/analyze/drawcalls` | GET | Draw Call 统计 |
| `/api/log/analyze/textures` | GET | 纹理分析 |
| `/api/log/analyze/bottleneck` | GET | 瓶颈分析 |
| `/api/log/trace/programs` | GET | 全局 program/shader 归纳 |
| `/api/log/trace/programs/:id` | GET | program 详情，含 shader 源码摘要 |
| `/api/log/frames/:id/programs` | GET | 单帧 program 使用和切换片段 |
| `/api/log/frames/:id/drawcalls` | GET | 单帧 draw call 明细分页 |
| `/api/log/analyze/workflow` | POST | 分析工作流 |
| `/api/log/export` | POST | 导出 |
| `/api/diagnose` | POST | Bug 诊断 |
| `/api/shutdown` | POST | 关闭服务 |
| `/health` | GET | 健康检查 |

## 数据模型

核心类型在 `internal/core/types.go`：
- `ParsedLog` → `[]FrameInfo` → `[]APILogEntry`
- `DiagnosisReport` → `[]Finding`（9 个检测器的输出）
- `OverviewResult` / `WorkflowResult` / `BottleneckAnalysis`
- `DrawCallStats` / `TextureInfo` / `BufferInfo`
- `TraceAnalysis` / `ProgramInfo` / `FrameProgramInsight` / `DrawCallInsight`
