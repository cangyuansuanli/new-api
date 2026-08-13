# 统一图像 API 参考

面向外部 API 客户与无限画布共用的 **唯一** 图像契约。各模型差异见模型广场「能力卡片」，本文只描述通用 endpoint 与 canonical 字段。

> 内部执行链路见 [`media-request-chain-audit.md`](media-request-chain-audit.md)。

## 端点

### 异步（推荐）

| 方法 | 路径 | 用途 |
|------|------|------|
| `POST` | `/v1/images/generations` | 文生图 / JSON 图生图（body 含 `async: true`） |
| `POST` | `/v1/images/edits` | multipart 图生图 / 蒙版编辑（`async: true`） |
| `GET` | `/v1/images/generations/{task_id}` | 查询 generations 任务 |
| `GET` | `/v1/images/edits/{task_id}` | 查询 edits 任务 |
| `GET` | `/v1/images/{task_id}/content` | 下载图片（部分模型） |

### 同步

| 方法 | 路径 | 用途 |
|------|------|------|
| `POST` | `/v1/images/generations` | 单次请求直接返回（勿传 `async` 或 `async: false`） |

鉴权：`Authorization: Bearer sk-xxx`

> 请使用 `/v1/images/edits` 作为统一 edits 入口；旧版 `/v1/edits` 仅为兼容转发，行为可能与异步路径不一致。

## Canonical 请求字段（JSON 文生 / 图生）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `model` | string | 是 | 模型广场 public 名 |
| `prompt` | string | 是 | 图像描述 |
| `async` | boolean | 异步必填 | 异步模式传 `true` |
| `aspect_ratio` | string | 否 | 画幅比例，如 `16:9`、`1:1` |
| `size` | string | 否 | 像素尺寸或比例 token（依模型 profile） |
| `output_resolution` | string | 否 | 档位：`1K` / `2K` / `4K`（Banana / Adobe 等） |
| `quality` | string | 否 | OpenAI 风格别名：`low` / `medium` / `high` |
| `n` | integer | 否 | 生成张数，默认 1 |
| `response_format` | string | 同步可选 | `url` 或 `b64_json` |

JSON 图生图参考字段：`image`、`images`、`reference_images`（URL 或 data URI）。

## JSON 示例（异步文生）

```bash
curl -X POST "https://newapi.example.com/v1/images/generations" \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-image-2",
    "prompt": "一只橘猫坐在窗台上，午后阳光",
    "size": "1024x1024",
    "n": 1,
    "async": true
  }'
```

## Multipart 图生 / 编辑

有本地参考图或蒙版时使用 `POST /v1/images/edits`：

| 字段 | 说明 |
|------|------|
| `model` | 模型名 |
| `prompt` | 描述 |
| `async` | `true` 启用异步 |
| `image` | 参考图文件（多图时重复字段名 `image`） |
| `mask` | 可选蒙版文件 |

## 轮询（异步）

```bash
curl "https://newapi.example.com/v1/images/generations/{task_id}" \
  -H "Authorization: Bearer sk-xxx"
```

`status`：`queued` | `in_progress` | `completed` | `failed`

## 同步响应

```json
{
  "created": 1715923200,
  "data": [{ "url": "https://example.com/image.png" }]
}
```

## 模型能力矩阵

各模型支持的参数、同步/异步模式、档位映射见模型广场 API 文档「能力卡片」，由 `image_ui_params` profile 驱动。

运营/避坑补充：[`infinite-canvas/docs/dev/models/`](../infinite-canvas/docs/dev/models/)
