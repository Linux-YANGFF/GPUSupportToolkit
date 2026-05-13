# GST 测试与验证

## 测试日志

位于 `../exmple_log/`：

| 文件 | 格式 | 大小 | 测试场景 |
|------|------|------|---------|
| `error.txt` | rawtrace | 1.7MB | 空指针：glVertexAttribPointer 无 VBO 绑定 |
| `apiTrace.log` | rawtrace | 637MB | 性能：未批量 draw call + 过量 glGetError |
| `1frame_demo_api.txt` | rawtrace | 212KB | 基本功能验证 |
| `1frame_profile_demo.txt` | profile | 708B | profile 格式解析 |

## 验证标准

### error.txt 空指针诊断
- [x] 格式: rawtrace, 64 帧
- [x] Critical: `glVertexAttribPointer at line 27766` 空指针无 VBO
- [x] High: 434 次客户端指针无 VBO（context 0x55b32bc5e0）
- [x] 资源泄漏: 15 个报告
- [x] 反模式: <20 条（已降噪）

### apiTrace.log 性能诊断
- [x] 格式: rawtrace（不是 profile）, 231 帧
- [x] High: 21,600 次 glGetError
- [x] Draw Call: 1,852,626 个（平均 8,020/帧）
- [x] 瓶颈: CPU-bound, glDrawArrays 瓶颈
- [x] 驱动错误: 211 个

### API 端点
- [x] `/api/overview` — 返回摘要 + 诊断结果
- [x] `/api/analyze/workflow` — crash/performance 正常
- [x] `/api/log/analyze/bottleneck` — 正确分类
- [x] `/api/log/analyze/drawcalls` — 统计正确
- [x] `/api/log/analyze/textures` — 正常（已修 null）
- [x] `/api/log/trace/programs` — `apiTrace.log` 返回 76 个 program，响应保持轻量
- [x] `/api/log/frames/0/programs` — 帧 0 返回 10 个 program、107 个切换片段
- [x] `/api/log/frames/0/drawcalls?page=1&page_size=5` — 帧 0 返回 121 条总数和分页明细

## 最近验证命令

```bash
GOCACHE=/tmp/gst-go-build /usr/local/go/bin/go test ./...
npm --prefix gst/web run build
```

前端渲染验证使用本地服务 `http://127.0.0.1:18080/logs.html` 和 Playwright 临时脚本完成，Browser 插件当前不可用。
