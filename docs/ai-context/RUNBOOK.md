# GST Runbook

## 基础环境

```bash
cd /root/code/GPUSupportToolkit/GPUSupportToolkit/gst
export PATH=/usr/local/go/bin:$PATH
export GOPROXY=https://goproxy.cn,direct
```

## 后端构建与运行

```bash
go build -o bin/gst-server ./cmd/gst-server
go build -o bin/gst-cli ./cmd/cli
GST_LOG_DIR=/root/code/GPUSupportToolkit/GPUSupportToolkit/exmple_log ./bin/gst-server -port 8080 -browser=false
```

常用健康检查：

```bash
curl http://localhost:8080/health
```

## 前端构建

```bash
cd /root/code/GPUSupportToolkit/GPUSupportToolkit/gst/web
npm run build
```

## 测试命令

```bash
cd /root/code/GPUSupportToolkit/GPUSupportToolkit/gst
go test ./...
go test ./internal/core/... -v
go test ./cmd/gst-server/internal/handlers -v
npm --prefix web run build
```

## 示例日志

- `exmple_log/error.txt` — rawtrace，空指针和资源问题验证。
- `exmple_log/apiTrace.log` — 大 rawtrace，性能和大文件验证。
- `exmple_log/log.txt` — 常规日志样本。

## Trace Inspector 快速检查

```bash
curl http://localhost:8080/api/log/trace/programs
curl http://localhost:8080/api/log/frames/0/programs
curl 'http://localhost:8080/api/log/frames/0/drawcalls?page=1&page_size=20'
```

## 开发注意事项

- 实际 API 路由以 `gst/cmd/gst-server/main.go` 为准。
- Handler 持有当前 `ParsedLog`，并用锁保护共享状态。
- 修改 API 响应时同步更新 `gst/web/src/types.ts` 和相关 composable。
- 修改解析、诊断或导出逻辑时，优先补充 Go 单元测试。
