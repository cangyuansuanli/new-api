# 统一音乐 / 音频生成 API 参考

面向外部 API 客户与无限画布共用的 **唯一** 音乐生成契约。各模型差异见模型广场「能力卡片」，本文只描述通用 endpoint 与 canonical 字段。

> 内部执行链路：入口 `POST /v1/audio/generations`（默认 async）→ **源站 `new-api-worker-1` 上 8 路 audio worker** 重放快照并出站 `POST /v1/chat/completions`（Gemini 音乐）→ `GET` 轮询取 `data[0].url`。

## 端点

| 方法 | 路径 | 用途 |
|------|------|------|
| `POST` | `/v1/audio/generations` | 创建音乐任务（默认异步） |
| `GET` | `/v1/audio/generations/{task_id}` | 查询任务状态与结果 |

鉴权：`Authorization: Bearer sk-xxx`

> 旧版 `POST /v1/chat/completions` + `model: gemini-music` 仍可用，但已标记 `Deprecation`，请迁移至本接口。

## Canonical 请求字段（JSON）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `model` | string | 是 | 模型广场 public 名，如 `gemini-music` |
| `prompt` | string | 是 | 音乐描述 / 风格 / 用途 |
| `async` | boolean | 否 | **默认 `true`**；仅调试或低延迟场景传 `async: false` 同步等待 |
| `response_format` | string | 否 | 默认 `url` |
| `stream` | boolean | 否 | 须省略或 `false` |

与图像/视频对齐：**客户端默认走异步**（图像画布侧 `asyncTask=true`，视频始终 task 制），结果以 **`url`** 返回。

## 异步提交（推荐，默认）

省略 `async` 或显式 `"async": true`：

```bash
curl -X POST "https://newapi.example.com/v1/audio/generations" \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-music",
    "prompt": "创作一首轻快的电子风格 BGM，适合科技产品广告"
  }'
```

立即返回 task：

```json
{
  "id": "task_xxx",
  "object": "audio.generation",
  "model": "gemini-music",
  "status": "queued",
  "progress": "20%",
  "created_at": 1715923200
}
```

## 轮询

```bash
curl "https://newapi.example.com/v1/audio/generations/{task_id}" \
  -H "Authorization: Bearer sk-xxx"
```

`status`：`queued` | `in_progress` | `completed` | `failed`

完成后：

```json
{
  "id": "task_xxx",
  "object": "audio.generation",
  "status": "completed",
  "progress": "100%",
  "data": [{ "url": "https://download.example.com/v1/audio/aud-xxxx/content" }]
}
```

`data[0].url` 为可直接 GET 下载的音频地址，**无需**再带 Authorization。建议轮询间隔 5–10 秒；上游生成约 30–60 秒。

## 同步模式（可选）

仅当 `"async": false` 时，一次 POST 阻塞等待（约 30–60 秒）：

```json
{
  "created": 1715923200,
  "data": [{ "url": "https://download.example.com/v1/audio/aud-xxxx/content" }]
}
```

## 与图像 / 视频默认行为对照

| 媒体 | 客户端默认 | 服务端默认 | 结果格式 |
|------|-----------|-----------|---------|
| 图像 | `async: true` + `response_format: url` | 须传 `async: true` 才异步 | `data[].url` |
| 视频 | 始终 task | 始终 task | `data[].url` |
| **音乐** | **`async: true`（可省略）** | **`async` 省略 = true** | **`data[].url`** |

## 模型与计费

| 模型 | 计费 | 令牌分组（OAIREGBox） |
|------|------|----------------------|
| `gemini-music` | 按次 ¥0.99/条（internal：`cy-au1-gemini-music`） | `gemini-高速` |

失败不计费；502/503/504 可指数退避重试。

## 兼容说明

| 路径 | 状态 |
|------|------|
| `POST /v1/audio/generations` | **推荐** 统一入口 |
| `POST /v1/chat/completions` + `gemini-music` | 兼容，响应仍为 Chat 格式 |

运营/避坑补充：[`infinite-canvas/docs/dev/models/`](../infinite-canvas/docs/dev/models/)
