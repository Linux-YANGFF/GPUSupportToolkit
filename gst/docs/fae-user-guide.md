# GST FAE 使用说明

版本：1.1.0

GST（GPU Support Toolkit）用于分析 apitrace/profile/rawtrace 日志，提供 Web 图形界面和命令行工具。FAE 日常优先使用 Web 界面；需要批处理或导出 JSON 时使用 CLI。

## 1. 安装

根据系统架构选择安装包：

| 系统 | x86_64/amd64 | ARM64/aarch64 |
|------|--------------|---------------|
| Ubuntu/Debian | `gst-1.1.0-amd64.deb` | `gst-1.1.0-arm64.deb` |
| RHEL/CentOS/Fedora | `gst-1.1.0-1.x86_64.rpm` | `gst-1.1.0-1.aarch64.rpm` |

Debian/Ubuntu：

```bash
sudo dpkg -i gst-1.1.0-amd64.deb
# 或 ARM64
sudo dpkg -i gst-1.1.0-arm64.deb
```

RHEL/CentOS/Fedora：

```bash
sudo rpm -Uvh gst-1.1.0-1.x86_64.rpm
# 或 ARM64
sudo rpm -Uvh gst-1.1.0-1.aarch64.rpm
```

安装后文件位置：

| 路径 | 用途 |
|------|------|
| `/usr/bin/gst-server` | Web 服务 |
| `/usr/bin/gst` | CLI 主命令 |
| `/usr/bin/gst-cli` | CLI 兼容命令，等价于 `gst` |
| `/usr/bin/gst-ui` | Web 图形界面启动器 |
| `/usr/bin/gst-create-desktop-shortcut` | 为当前用户创建桌面快捷方式 |
| `/usr/share/gst/web/` | Web 前端静态文件 |
| `/usr/share/applications/gst.desktop` | 桌面启动入口 |
| `/usr/share/icons/hicolor/scalable/apps/gst.svg` | 应用图标 |
| `/var/lib/gst/` | 数据目录 |

## 2. 启动 Web 工具

方式一：桌面环境中点击应用菜单里的 **GST**。

如果需要桌面图标，用当前登录用户执行：

```bash
gst-create-desktop-shortcut
```

方式二：终端启动：

```bash
gst-ui
```

`gst-ui` 会自动选择可用端口并打开浏览器。如果浏览器没有自动打开，可查看日志里的访问地址：

```bash
cat ~/.local/state/gst/gst-ui.log
```

如需固定端口启动：

```bash
gst-server --port 8080 --browser --web-dir /usr/share/gst/web
```

然后手动访问：

```text
http://localhost:8080/logs.html
```

健康检查：

```bash
curl http://localhost:8080/health
```

返回 `OK` 表示服务正常。

## 3. Web 界面常用流程

1. 打开 `http://localhost:8080/logs.html`。
2. 在日志路径输入框中填写日志绝对路径，或使用上传入口选择日志文件。
3. 点击“解析”。
4. 查看“帧列表”，选择异常帧进入详情。
5. 使用分析页签查看函数统计、Draw Call、Texture、Program/Shader、诊断结果。
6. 需要给研发定位时，下载完整帧日志或导出 JSON/CSV/TXT。

建议先看：

| 页面/功能 | 适用场景 |
|-----------|----------|
| 概览 | 快速判断帧数、耗时、API 数、诊断摘要 |
| 帧列表 | 找慢帧、API 密集帧、Draw Call 密集帧 |
| 帧详情 | 查看单帧 API、原始日志、Program/DrawCall 细节 |
| 诊断 | 查资源生命周期、Shader 问题、Driver error、性能反模式 |
| 导出 | 给研发或 AI 工具进一步分析 |

## 4. CLI 常用命令

显示帮助：

```bash
gst --help
gst-cli --help
```

解析日志：

```bash
gst -parse /path/to/trace.log
```

查看综合概览：

```bash
gst -overview -parse /path/to/trace.log
```

输出诊断 JSON：

```bash
gst -diagnose-json -parse /path/to/trace.log
```

输出 Markdown 诊断报告：

```bash
gst -diagnose -parse /path/to/trace.log > diagnosis.md
```

导出分析结果：

```bash
gst -export json -output result.json -parse /path/to/trace.log
gst -export csv -output result.csv -parse /path/to/trace.log
gst -export txt -output result.txt -parse /path/to/trace.log
```

导出单帧原始日志：

```bash
gst -frame-raw 137 -output frame-137.log -parse /path/to/trace.log
```

## 5. 支持的日志

| 格式 | 说明 |
|------|------|
| rawtrace | 每行一个 GL/API 调用，适合做帧级和调用级定位 |
| apitrace 聚合格式 | 包含 `count=`、`time=` 等统计字段 |
| profile 格式 | 轻量统计日志 |

大日志建议使用 Web 路径解析或 CLI，避免把完整日志复制到聊天工具中。

## 6. 结果解读

| 字段 | 含义 |
|------|------|
| `critical` | 高置信度严重问题，优先给研发处理 |
| `high` | 高风险问题，通常影响性能、稳定性或渲染正确性 |
| `medium` | 需要结合场景判断的问题或性能建议 |
| `low` | 低置信度或依赖日志完整性的提示 |
| `has_timing=false` | 当前 rawtrace 没有真实耗时，不能直接用耗时判断瓶颈 |

如果日志不包含 frame cost/profile 耗时，GST 仍可统计 API、Draw Call、资源和诊断项，但耗时类瓶颈结论会受限。

## 7. 故障排查

| 现象 | 处理 |
|------|------|
| 点击 GST 没反应 | 终端运行 `gst-server --port 8080 --browser=false --web-dir /usr/share/gst/web` 查看错误 |
| 端口被占用 | 改用 `gst-server --port 18080 --browser --web-dir /usr/share/gst/web` |
| 页面打不开 | 先确认 `curl http://localhost:8080/health` 是否返回 `OK` |
| 日志路径无法解析 | 使用绝对路径，并确认当前用户有读取权限 |
| 诊断结果很少 | 确认日志是否完整、是否包含原始 API 调用；聚合日志能提供的信息较少 |
| ARM 包无法安装 | 确认系统架构为 `aarch64/arm64`，不要在 x86_64 系统安装 ARM 包 |

卸载：

```bash
sudo dpkg -r gst
# 或
sudo rpm -e gst
```
