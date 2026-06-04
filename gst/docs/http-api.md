# GST HTTP API Reference

The HTTP API is served by `gst-server`. Start it with:

```bash
./bin/gst-server -port 8080 -browser=false -web-dir web/dist
```

The server binds to `127.0.0.1` by default. Use `-host 0.0.0.0` only when remote access is required. Local path parsing is restricted to the current directory unless `GST_LOG_DIR` is set.

All JSON endpoints return `application/json` unless noted.

## Health

| Method | Path | Description |
|:---|:---|:---|
| `GET` | `/health` | Server health check |

## Log Loading

| Method | Path | Description |
|:---|:---|:---|
| `POST` | `/api/log/parse` | Parse a log from JSON path or multipart upload |

JSON request:

```json
{"path":"/path/to/trace.log"}
```

Multipart upload uses form field `file` and optional `filename`.

## Frames

| Method | Path | Description |
|:---|:---|:---|
| `GET` | `/api/log/frames?page=1&page_size=50` | Paginated frame summaries |
| `GET` | `/api/log/frames/:id` | Full frame detail |
| `GET` | `/api/log/frames/:id/apis?page=1&page_size=200` | Aggregated API stats for one frame |
| `GET` | `/api/log/frames/:id/funcs` | Function stats for one frame |
| `GET` | `/api/log/frames/:id/programs` | Program usage for one frame |
| `GET` | `/api/log/frames/:id/drawcalls?page=1&page_size=50` | Draw calls for one frame |
| `GET` | `/api/log/frames/:id/raw-lines?page=1&page_size=200` | Original log lines for one frame |
| `GET` | `/api/log/frames/:id/download` | Download full original log slice for one frame as text |

## Search And Analysis

| Method | Path | Description |
|:---|:---|:---|
| `GET` | `/api/log/search?q=glDrawElements&page=1&page_size=50` | Search current log |
| `GET` | `/api/log/search/time?start_us=1000&end_us=50000&page=1&page_size=50` | Find frames by frame cost/time range; non-indexed logs include API calls |
| `GET` | `/api/log/analyze/top?n=10` | Top slow frames |
| `GET` | `/api/log/analyze/funcs` | Function statistics |
| `GET` | `/api/log/analyze/shaders` | Shader/program statistics |
| `GET` | `/api/log/analyze/drawcalls` | Draw call analysis |
| `GET` | `/api/log/analyze/textures` | Texture analysis |
| `GET` | `/api/log/analyze/bottleneck` | Bottleneck summary |
| `POST` | `/api/log/analyze/workflow` | Legacy workflow analysis |
| `GET` | `/api/overview` | Overall current-case overview |
| `POST` | `/api/diagnose` | Legacy bug diagnosis report |

Workflow request:

```json
{"workflow":"performance"}
```

`workflow` accepts `performance`, `crash`, `rendering`, or `memory`.

Diagnosis uses the current parsed log and does not require a request body.

## Trace Inspector

| Method | Path | Description |
|:---|:---|:---|
| `GET` | `/api/log/trace/programs` | Program/shader relationship summary |
| `GET` | `/api/log/trace/programs/:id` | Program detail |

## Export

| Method | Path | Description |
|:---|:---|:---|
| `POST` | `/api/log/export` | Export current parsed result as `json`, `csv`, or `txt` |

Export request:

```json
{"format":"json","type":"frames","query":""}
```

`format` accepts `json`, `csv`, or `txt`. `type` accepts `frames`, `funcs`, `shader`, `search`, `top`, or `longest`.

## v2 AI API

These endpoints are the stable preferred surface for AI consumers.

| Method | Path | Description |
|:---|:---|:---|
| `GET` | `/api/v2/cases/current/overview` | Current case overview plus OpenGL stats contract |
| `GET` | `/api/v2/cases/current/ai-summary?n=10` | Compact case-level AI summary |
| `GET` | `/api/v2/cases/current/frames?page=1&page_size=50` | v2 alias for frame list |
| `GET` | `/api/v2/cases/current/frames/:id` | v2 alias for frame detail |
| `GET` | `/api/v2/cases/current/frames/:id/stats` | Frame-level OpenGL category/key API stats |
| `GET` | `/api/v2/cases/current/frames/:id/apis` | v2 alias for frame API stats |
| `GET` | `/api/v2/cases/current/frames/:id/programs` | v2 alias for frame program usage |
| `GET` | `/api/v2/cases/current/frames/:id/drawcalls` | v2 alias for frame draw calls |
| `GET` | `/api/v2/cases/current/frames/:id/raw-lines` | v2 alias for raw line preview |
| `GET` | `/api/v2/cases/current/frames/:id/download` | v2 alias for frame log download |

## Server Control

| Method | Path | Description |
|:---|:---|:---|
| `POST` | `/api/shutdown` | Graceful shutdown for desktop/packaged runs |
