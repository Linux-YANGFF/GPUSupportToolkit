# GST Runbook

## 基础环境

```bash
cd /root/code/GPUSupportToolkit/GPUSupportToolkit/gst
export PATH=/usr/local/go/bin:$PATH
export GOPROXY=https://goproxy.cn,direct
```

## 构建与运行

```bash
go build -o bin/gst-server ./cmd/gst-server
go build -o bin/gst-cli ./cmd/cli
GST_LOG_DIR=/root/code/GPUSupportToolkit/GPUSupportToolkit ./bin/gst-server -port 18080 -browser=false -web-dir web/dist
curl http://localhost:18080/health
```

```bash
cd /root/code/GPUSupportToolkit/GPUSupportToolkit/gst
make build-all
```

## 打包

```bash
cd /root/code/GPUSupportToolkit/GPUSupportToolkit/gst
make deb-amd64
make deb-arm64
make rpm-amd64
make rpm-arm64
```

deb 目标使用 `dpkg-deb`，当前环境可直接验证。rpm 目标依赖 `rpmbuild`，如果机器未安装 `rpm-build` 会明确失败；在 release 机或 packaging 容器中运行即可。

## 测试命令

```bash
cd /root/code/GPUSupportToolkit/GPUSupportToolkit/gst
go test ./...
go test ./internal/core/... -v
go test ./cmd/gst-server/internal/handlers -v
go test ./internal/core/glstats -v
npm --prefix web run build
npm --prefix web run test:e2e
```

Playwright 在当前环境使用系统 Chrome；如果普通沙箱下 Chrome crashpad/seccomp 失败，需在 Codex 中用已授权的 escalated Playwright 命令运行。

## tab22 大日志回归

```bash
curl -X POST -H 'Content-Type: application/json' \
  -d '{"path":"/root/code/GPUSupportToolkit/GPUSupportToolkit/exmple_log/tab22_api.log"}' \
  http://localhost:18080/api/log/parse

curl 'http://localhost:18080/api/log/frames?page=1&page_size=3'
curl 'http://localhost:18080/api/log/frames/1/apis?page=1&page_size=5'
curl 'http://localhost:18080/api/log/frames/1/raw-lines?page=1&page_size=20'
curl -OJ 'http://localhost:18080/api/log/frames/1/download'
curl 'http://localhost:18080/api/log/frames/659/programs'
curl 'http://localhost:18080/api/log/frames/659/drawcalls?page=1&page_size=3'
curl http://localhost:18080/api/overview
```

期望关键值：706 帧；第 1 帧 API 数 1975、draw 118；第 659 帧 draw 38358，Program 194 draw 38211。

## Trace Inspector 快速检查

```bash
curl http://localhost:18080/api/log/trace/programs
curl http://localhost:18080/api/log/frames/1/programs
curl 'http://localhost:18080/api/log/frames/1/drawcalls?page=1&page_size=20'
```

## v2 / AI 快速检查

```bash
curl http://localhost:18080/api/v2/cases/current/overview
curl http://localhost:18080/api/v2/cases/current/frames/423/stats
curl 'http://localhost:18080/api/v2/cases/current/frames/423/raw-lines?page=1&page_size=20'
curl 'http://localhost:18080/api/v2/cases/current/ai-summary?n=5'
./bin/gst-cli -parse ../exmple_log/1frame_profile_demo.txt -ai-summary
```

`/api/v2/cases/current/frames/:id/stats` 返回帧级 OpenGL 分类、重点 API、耗时来源和 evidence；AI 优先使用该类小 JSON，不要直接读取整份日志。

## 对外文档

- CLI：`gst/docs/cli.md`
- HTTP API：`gst/docs/http-api.md`
- 安装/打包：`gst/docs/install.md`

## 开发注意事项

- 实际 API 路由以 `gst/cmd/gst-server/main.go` 为准。
- Handler 持有当前 `ParsedLog`，并用锁保护共享状态。
- 修改 API 响应时同步更新 `gst/web/src/types.ts` 和相关 composable。
- OpenGL 函数分类统一维护在 `gst/internal/core/glstats`，不要在新 analyzer 中重复硬编码分类。
- 修改解析、诊断或导出逻辑时，优先补充 Go 单元测试。
- indexed 日志不能把每条 API 常驻内存；新增明细能力时优先基于 frame offset 分页扫描或流式下载。
