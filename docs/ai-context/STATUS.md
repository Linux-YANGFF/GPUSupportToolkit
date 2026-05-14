# GST 当前状态

更新日期：2026-05-14

## 已实现能力

- 解析：支持 apitrace 聚合格式、profile、rawtrace；大 rawtrace/hybrid apitrace 已改为索引式流式解析，避免 700MB+ 日志 OOM。
- 后端 API：解析、帧列表/详情、搜索、Top 慢帧、函数统计、shader、draw call、texture、bottleneck、overview、export；workflow/diagnose 属于 legacy。
- 帧明细：帧列表显示真实 start/end line、frame cost、swap/API 耗时、API 数和 draw call 数；新增 `/raw-lines` 文本预览和 `/download` 指定帧完整日志下载。
- OpenGL 统计：新增 `internal/core/glstats`，集中维护 Draw、Buffer、Texture、Shader/Program、Vertex Input、Framebuffer、State、Sync/Query/Readback、Resource、Context 分类。
- v2/AI API：新增 `/api/v2/cases/current/overview`、`/api/v2/cases/current/frames/:id/stats`、`/api/v2/cases/current/ai-summary`。
- Trace Inspector：从 rawtrace log 归纳 program/shader 关系、每帧 program 使用、program switch segment、draw call 明细分页；支持 source/program_binary/unknown 置信度。
- Web UI：Vue 3/Vite + Element Plus；帧列表和 Top 帧共用统一帧详情弹窗，原始 API 以分页文本显示，不再按列拆表。
- 测试：Go 单测、Vue build、Playwright E2E smoke test 已可运行；Playwright 使用系统 Google Chrome。
- 测试资料：`exmple_log/` 下有 rawtrace/profile 示例和大日志验证样本，注意该目录无需上库。

## 当前优先级

1. 用浏览器完整手测：帧列表分类列、帧详情分类统计、Trace Inspector、API 分页、drawcall/program 过滤。
2. 把 v2 current case 逐步演进为持久 case/index；SQLite 只作为索引层，不复制原始日志。
3. 收敛 legacy：前端不再展示 Bug 诊断和工作流分析，新能力只写到 `glstats`/v2。
4. 收敛安全边界：限制 `/api/log/parse` 路径读取范围，收紧 `GST_LOG_DIR` 默认行为。
5. 后续扩展：MCP Server、帧间差异、日志对比、更深的渲染管线溯源。

## 已知风险

- 工作树当前有较多未提交改动；修改前先看 `git status --short`，不要回滚非本人改动。
- `.planning/`、`.sisyphus/`、`exmple_log/` 只留本地，不应上库；`AI/` 是用户截图/问题资料目录，也不要误删。
- `docs/OVERVIEW.md` 和代码中的 workflow 路径可能存在命名不一致；workflow 已不属于新产品主线。
- 搜索 handler 仍是全文件扫描，但已改为服务端分页保留当前页结果，避免命中集全部常驻内存。
- Trace Inspector 的 shader 源码只在日志保留 `glShaderSource` 或 `glProgramBinary` 信息时可还原；纯 log 无法恢复 GPU 内部运行时状态。
- Playwright 自带 Chromium 下载失败过：官方 CDN 超时，npmmirror 缺当前版本包；当前配置改用系统 `/usr/bin/google-chrome`。

## 最近变化

- 新增 `indexed_raw_trace_parser.go`，tab22 级别大日志解析不再常驻全部 API 调用。
- `tab22_api.log` 验证：717MB，706 帧，`total_time_us=212938000`；帧数不再多两帧。
- 第 659 帧验证：`api_count=155883`，`draw_call_count=38358`，Program 194 贡献 38211 次 draw。
- Overview 验证：P95/P99 有真实值，top 慢帧、函数统计、瓶颈判断有数据。
- 纹理分析 indexed fallback：`tab22_api.log` 返回 `total_textures=7651`、`active_textures=7523`。
- Playwright 已安装并新增 smoke test；`npm --prefix gst/web run test:e2e` 通过。
- 新增 v2 OpenGL 帧统计：`glDrawElements/glDrawArrays`、buffer/texture 传输、shader/program、framebuffer、sync/readback 等分类集中在 `glstats`。
- profile 源帧号现在会保留，例如 `423 frame cost 109ms` 对应帧号 423。
- CLI 新增 `-ai-summary` 和 `-frame-stats <N>`，输出 AI 友好的 JSON。
- 指定帧原始日志：`/api/log/frames/:id/raw-lines` 默认去除 `[line]` 前缀；`/download` 流式输出完整帧日志。
- Playwright E2E 覆盖页面加载、路径解析、帧详情原始日志预览和帧日志下载。
- 帧详情弹窗新增完整 API 调用统计表，展示每个 API 的 count、total time、avg time。
- Shader 统计新增 fallback：没有 `glShaderSource` 源码时，也会展示 Shader/Program 相关 API 统计，例如 `glUseProgram`、`glProgramUniform*`。
- CLI/HTTP 对外接口文档已补齐：CLI 见 `gst/docs/cli.md`，HTTP API 见 `gst/docs/http-api.md`；HTTP 已补 `/api/log/search/time` 对齐 CLI `-time`。
- 打包链路已收敛到 `gst/Makefile`：`make deb-amd64`、`make deb-arm64`、`make rpm-amd64`、`make rpm-arm64`；deb 使用 `dpkg-deb`，rpm 需要 `rpmbuild`。
- 构建默认使用 `/usr/local/go/bin/go`（Go 1.22.10）和 `/tmp/gst-go-build`，避免系统 Go 1.18 与只读 `/root/.cache/go-build`。
