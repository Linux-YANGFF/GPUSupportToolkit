# GST 当前状态

更新日期：2026-05-13

## 已实现能力

- 解析：支持 apitrace 聚合格式、profile、rawtrace，并支持大文件流式处理。
- 后端 API：解析、帧列表/详情、搜索、Top 慢帧、函数统计、shader、draw call、texture、bottleneck、overview、workflow、diagnose、export。
- Trace Inspector：从 rawtrace log 归纳 program/shader 关系、每帧 program 使用、program switch segment、draw call 明细分页；支持 source/program_binary/unknown 置信度。
- 诊断：空指针、资源泄漏、shader 错误、反模式、性能异常、线程安全、驱动错误、小批量 draw call、过量 `glGetError`。
- Web UI：Vue 3/Vite 应用已接入文件上传、帧列表、搜索、分析、Trace Inspector、Bug 诊断、导出等视图。
- 测试资料：`exmple_log/` 下有 rawtrace/profile 示例和大日志验证样本。

## 当前优先级

1. 收敛安全边界：限制 `/api/log/parse` 路径读取范围，收紧 `GST_LOG_DIR` 默认行为。
2. 修复影响正确性的问题：CSV 导出行为、CLI rawtrace 导出检测、资源泄漏复数创建 API 追踪。
3. 优化性能热点：搜索改用已构建的 KeywordIndex，诊断/综合分析考虑并发或缓存。
4. 后续扩展：MCP Server、帧间差异、日志对比、更深的渲染管线溯源。

## 已知风险

- 工作树当前有较多未提交改动；修改前先看 `git status --short`，不要回滚非本人改动。
- `docs/OVERVIEW.md` 和代码中的 workflow 路径可能存在命名不一致，实际路由以 `gst/cmd/gst-server/main.go` 为准。
- 搜索 handler 当前仍可能全文件扫描，已有 KeywordIndex 未充分利用。
- Trace Inspector 的 shader 源码只在日志保留 `glShaderSource` 或 `glProgramBinary` 信息时可还原；纯 log 无法恢复 GPU 内部二进制或运行时不可见状态。

## 最近变化

- 新增 Trace Inspector 后端分析器、API、Vue 前端视图和单元/handler 测试。
- `apiTrace.log` 大日志验证：231 帧、76 个 program；帧 0 有 121 个 draw call、10 个 program，draw call 明细分页正常。
- 顶层 `docs/OVERVIEW.md / ARCHITECTURE.md / TESTING.md` 已形成较完整的项目说明。
- 后端已扩展 draw call、texture、overview、bottleneck、workflow 等分析接口。
- 本目录新增为 AI 新会话的短上下文入口。
