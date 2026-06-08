# GST 1.1.0 发布测试报告

测试日期：2026-05-19

## 结论

GST 1.1.0 已完成源码门禁、Web 构建、amd64/arm64 交叉编译、DEB/RPM 打包、amd64 DEB clean-room 安装、amd64 Web E2E 冒烟、诊断回归验证和安装体验优化。

arm64 包已完成交叉编译、包结构和架构检查；运行时安装/启动仍需在真实 ARM64 机器或启用 arm64 容器/QEMU 的环境中做最终验收。

## 已生成产物

| 产物 | 状态 |
|------|------|
| `bin/gst-1.1.0-amd64.deb` | 已生成并通过 amd64 clean-room 安装冒烟 |
| `bin/gst-1.1.0-arm64.deb` | 已生成并通过包结构/架构检查 |
| `bin/gst-1.1.0-1.x86_64.rpm` | 已生成并通过 RPM 元数据/文件列表检查 |
| `bin/gst-1.1.0-1.aarch64.rpm` | 已生成并通过 RPM 元数据/文件列表检查 |
| `gst/docs/fae-user-guide.md` | 已生成 |
| `bin/release/fae-user-guide-1.1.0.pdf` | 已生成 |
| `bin/release/SHA256SUMS` | 已生成 |

## 验证结果

| 检查项 | 结果 |
|--------|------|
| `go test ./...` | PASS |
| `go vet ./...` | PASS |
| `npm --prefix web run build` | PASS，存在 Element Plus chunk size 警告，不阻断发布 |
| amd64/arm64 Go 交叉编译 | PASS |
| amd64/arm64 DEB 打包 | PASS |
| x86_64/aarch64 RPM 打包 | PASS；aarch64 在 x86_64 主机构建时有 strip 识别警告，但 RPM 成功生成 |
| DEB/RPM 包内容 | PASS，包含 `gst`、`gst-cli`、`gst-server`、`gst-ui`、桌面入口和图标 |
| `gst-server -port 0` | PASS，可绑定随机可用端口并输出真实访问 URL |
| amd64 DEB 容器安装 | PASS |
| amd64 Web `/health` | PASS |
| Playwright Web 冒烟 | PASS，2/2 |
| CLI `-diagnose-json error.txt` | PASS，返回 34 findings，不再是 0 |
| HTTP `/api/diagnose error.txt` | PASS，返回 34 findings，不再是 0 |

## 诊断回归说明

修复前 `NewDefaultRegistry()` 只注册 3 个诊断器，导致 CLI JSON、HTTP 诊断和 Overview 中的诊断摘要对 indexed rawtrace 返回 0 findings。修复后默认注册 9 个诊断器，并在诊断路径按需补齐 indexed rawtrace 的原始 `APICalls`。

当前 `error.txt` 实测结果：

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High | 6 |
| Medium | 13 |
| Low | 15 |
| Total | 34 |

旧测试文档中 `error.txt` 的 Critical 空指针预期已过期；空指针 Critical 由新增的小型回归测试覆盖。

## 安装体验优化说明

本轮补齐了 FAE 安装后的用户入口和卸载清理：

| 项目 | 结果 |
|------|------|
| CLI 主入口 | `/usr/bin/gst` |
| CLI 兼容入口 | `/usr/bin/gst-cli -> gst` |
| Web UI 启动器 | `/usr/bin/gst-ui`，默认使用随机可用端口并打开浏览器 |
| 桌面入口 | `/usr/share/applications/gst.desktop` |
| 应用图标 | `/usr/share/icons/hicolor/scalable/apps/gst.svg` |
| 可选桌面快捷方式 | 当前用户执行 `gst-create-desktop-shortcut` |
| 卸载清理 | DEB/RPM 脚本刷新桌面/icon 缓存，清理 pid/runtime 文件，空 `/var/lib/gst` 自动删除 |

## 遗留风险

| 风险 | 处理建议 |
|------|----------|
| arm64 未做真实运行时安装/启动 | FAE 发布前在 ARM64 Linux 机器执行安装、`gst --help`、`gst-server /health`、Web 解析样例日志 |
| RPM 未在 RHEL/Fedora 容器实际安装运行 | 在目标发行版执行 `rpm -Uvh`、`gst --help`、`gst-server /health` |
| `desktop-file-validate` 当前构建环境未安装 | 若发布流程要求桌面规范门禁，在打包机安装 `desktop-file-utils` 后补跑 |
| Vite chunk size 警告 | 不影响当前功能；后续可拆分 Element Plus chunk 优化加载体积 |
| Makefile 之前未跟踪 internal Go 依赖 | 已修复，后续 internal 代码变化会触发 server/cli 重编 |
