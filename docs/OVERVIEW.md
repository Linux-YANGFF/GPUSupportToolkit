# GST 项目概述

GST (GPU Support Toolkit) — GPU 日志分析工具，将 GB 级 apitrace 日志结构化，服务于 AI agent 和 FAE/研发。

## 核心定位

AI 和 GPU 日志之间的**翻译层**：把几百 GB 原始文本压缩为 KB 级结构化信息。

**主目录**: `gst/`

## 已实现功能

### 解析引擎
- 支持 3 种日志格式：apitrace（聚合）、profile、rawtrace（原始）
- 自动格式检测，流式解析大文件（>600MB ~40s）

### 分析能力
| 功能 | API | 说明 |
|------|-----|------|
| 综合摘要 | `GET /api/overview` | 一次调用返回全貌 |
| 帧分析 | `/api/log/frames` | 帧列表、帧详情、Top N 慢帧 |
| 检索 | `/api/log/search` | 关键字 AND 搜索 |
| 函数统计 | `/api/log/analyze/funcs` | API 调用频率和耗时 |
| Shader 统计 | `/api/log/analyze/shaders` | 编译统计、源码 |
| Draw Call 分析 | `/api/log/analyze/drawcalls` | 每帧统计、类型分类、热帧 |
| 纹理分析 | `/api/log/analyze/textures` | 生命周期、泄漏检测 |
| 瓶颈分析 | `/api/log/analyze/bottleneck` | CPU/GPU bound 判断 |
| Trace Inspector | `/api/log/trace/programs`, `/api/log/frames/:id/programs`, `/api/log/frames/:id/drawcalls` | 归纳 program/shader、帧内 program 使用和 draw call 明细 |
| 分析工作流 | `POST /api/analyze/workflow` | performance/crash/rendering/memory |
| 导出 | `POST /api/log/export` | TXT/CSV/JSON |

### Bug 诊断引擎（9 个检测器）
| 检测器 | 检测内容 |
|--------|---------|
| NullPointerDetector | 空指针、无 VBO 绑定的客户端指针 |
| ResourceLeakDetector | buffer/texture/shader/program/VAO 泄漏 |
| ShaderErrorDetector | 重复编译、未检查编译状态 |
| AntiPatternDetector | 冗余状态切换、无 program 的 draw call |
| PerfAnomalyDetector | 帧时间统计异常 |
| ThreadSafetyDetector | 多线程 GL context 访问 |
| DriverErrorDetector | __glSetError 解析 |
| UnbatchedDrawCallDetector | 连续小批量 draw call |
| ExcessiveGetErrorDetector | 过量 glGetError 调用 |

### Web UI (Vue 3)
6 个 Tab：帧列表、搜索、分析（Top N + Shader + 概览能力）、Trace Inspector、Bug 诊断、导出

## 未实现需求

| 需求 | 优先级 | 说明 |
|------|--------|------|
| 安全边界收敛 | 高 | 限制 `/api/log/parse` 路径读取范围，收紧默认日志目录 |
| MCP Server 集成 | 中 | 让 AI agent 直接调用 GST |
| 帧间差异对比 | 中 | 对比两帧的 API 调用差异 |
| 日志对比 | 低 | 优化前后性能对比 |
| 渲染管线分析 | 低 | 基于 Trace Inspector 继续扩展资源和状态溯源 |

## 技术栈

Go 1.22（零外部依赖）+ Vue 3 + Vite + TypeScript

## 构建与运行

```bash
cd gst
export PATH=/usr/local/go/bin:$PATH
export GOPROXY=https://goproxy.cn,direct
go build -o bin/gst-server ./cmd/gst-server
GST_LOG_DIR=/path/to/logs ./bin/gst-server -port 8080 -browser=false
```
