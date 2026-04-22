# GST - GPU Support Toolkit

GPU 日志解析与性能分析工具，支持 apitrace/profile 日志。

## 功能特性

- **日志解析**：支持聚合格式和原始格式的 apitrace/profile 日志
- **帧分析**：按耗时排序分析 Top N 帧，定位性能瓶颈
- **API 统计**：按帧聚合 API 调用次数和耗时
- **Shader 检测**：检测日志中的 glShaderSource 调用
- **关键字搜索**：在日志中快速搜索关键字
- **导出功能**：支持 JSON/CSV/TXT 格式导出

## 快速开始

### 编译

```bash
# 下载依赖
go mod tidy

# 编译服务
go build -o bin/gst-server ./cmd/gst-server

# 编译 CLI（可选）
go build -o bin/gst-cli ./cmd/gst-cli
```

### 运行

```bash
# 启动服务（默认端口 8080）
./bin/gst-server

# 指定端口
./bin/gst-server -port 8080

# 禁止自动打开浏览器
./bin/gst-server -browser=false
```

然后在浏览器访问 http://localhost:8080

### 使用 CLI

```bash
# 解析日志
./bin/gst-cli -parse /path/to/log.trace

# 搜索关键字
./bin/gst-cli -search glDrawElements -parse /path/to/log.trace

# Top 20 帧分析
./bin/gst-cli -top 20 -parse /path/to/log.trace
```

## API 接口

### 解析日志

```bash
# 本地文件路径
curl -X POST http://localhost:8080/api/log/parse \
  -H "Content-Type: application/json" \
  -d '{"path":"/path/to/log.trace"}'

# 文件上传
curl -X POST http://localhost:8080/api/log/parse \
  -F "file=@/path/to/log.trace"
```

### 帧列表（分页）

```bash
curl "http://localhost:8080/api/log/frames?page=1&page_size=50"
```

### 帧详情

```bash
curl "http://localhost:8080/api/log/frames/90"
```

### 关键字搜索

```bash
curl "http://localhost:8080/api/log/search?q=glDrawElements"
```

### Top N 分析

```bash
curl "http://localhost:8080/api/log/analyze/top?n=20"
```

### Shader 列表

```bash
curl http://localhost:8080/api/log/analyze/shaders
```

### 导出

```bash
curl -X POST http://localhost:8080/api/log/export \
  -H "Content-Type: application/json" \
  -d '{"format":"json","type":"frames"}' > export.json
```

## 项目结构

```
gst/
├── cmd/
│   ├── gst-server/    # HTTP 服务
│   └── gst-cli/       # CLI 工具
├── internal/
│   └── core/
│       ├── parser/    # 日志解析器
│       ├── analyzer/  # 分析器
│       ├── search/    # 搜索
│       └── exporter/ # 导出
├── web/               # 前端文件
│   ├── index.html    # 主页
│   └── logs.html     # 日志分析页面
└── packaging/         # 打包配置
```

## 打包

```bash
# 编译所有
make build-all

# 生成 deb 包
make deb

# 生成 rpm 包
make rpm

# 生成所有包
make package
```

## 开发

```bash
# 运行测试
go test ./...

# 代码检查
go vet ./...
go fmt ./...
```

## 日志格式

工具支持两种 apitrace 日志格式：

### 聚合格式

```
[ 31085] swapBuffers: 64205 us
[ 31086] <<gc = 0xffff60638d80>>
[ 31087] glDeleteTextures: count=22, time=433 us
[ 31088] glFlush: count=5, time=203 us
...
[ 35645] 2 frame cost 8061ms
```

### 原始格式

```
[146982] (gc=0xfffe6985a840, tid=0x797f6fc0): glBindBuffer 0x8893 199
[146983] (gc=0xfffe6985a840, tid=0x797f6fc0): glEnableVertexAttribArray 2
...
[146981] 38 frame cost 110ms
```

### Shader 源码

```
[ 91086] (gc=0xffff63218c40, tid=0x797f6fc0): glShaderSource 35 1 0xffff797f59e8 (nil) 
[ 91087] ####
[ 91088] #version 400
[ 91089] out vec4 webgl_FragColor;
...
[ 91099] ####
```

## 需求对应

| 需求 | API 端点 |
|:---|:---|
| R1: 统计帧数 | `/api/log/frames` 返回 total |
| R2: 最长帧 | `/api/log/analyze/top?n=1` 第一条 |
| R3: 最长帧 API 统计 | `/api/log/frames/{id}` 返回 APISummary |
| R4: Shader 列表 | `/api/log/analyze/shaders` |
| R5: Top 20 帧详情 | `/api/log/analyze/top?n=20` |
