# 统一视频 API 参考

面向外部 API 客户与无限画布共用的 **唯一** 视频契约。各模型差异见模型广场「能力卡片」（价格、参考素材上限、是否 multipart 等），本文只描述通用 endpoint 与 canonical 字段。

> 内部路由与 vendor 出站见 [`video-task-routing.md`](video-task-routing.md)。

## 端点

| 方法 | 路径 | 用途 |
|------|------|------|
| `POST` | `/v1/videos` | 创建视频任务（新调用使用 `application/json`） |
| `GET` | `/v1/videos/{task_id}` | 查询任务状态与结果 |
| `GET` | `/v1/videos/{task_id}/content` | 下载已完成任务的成片 |

鉴权：`Authorization: Bearer sk-xxx`

## Canonical 请求字段（JSON）

统一视频 API 使用 JSON。浏览器中的本地素材应先直传临时对象存储，再把 HTTPS URL 放入参考字段；不要使用 data URI 承载大文件。

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `model` | string | 是 | 模型广场 public 名 |
| `prompt` | string | 是 | 视频描述 |
| `duration` | integer | 否 | 时长（秒）；具体范围由模型能力决定 |
| `aspect_ratio` | string | 否 | 如 `16:9`、`9:16`、`1:1` |
| `resolution` | string | 否 | 如 `480p`、`720p`、`1080p` |
| `size` | string | 否 | 像素画幅，如 `1280x720`（部分 OpenAI Video 族模型） |
| `seed` | integer | 否 | 可复现种子（仅支持模型生效） |
| `generate_audio` | boolean | 否 | 是否生成音频（仅支持模型生效） |
| `reference_image_urls` | string[] | 否 | **推荐** 参考图 URL 或 data URI |
| `reference_videos` | string[] | 否 | 参考视频 URL |
| `reference_audios` | string[] | 否 | 参考音频 URL |
| `first_image_url` | string | 否 | 首帧参考图 |
| `last_image_url` | string | 否 | 末帧参考图 |

### 历史字段兼容

新接入只使用上表 canonical 字段。为保证已上线客户无需改代码，服务端仍接受历史参考图字段 `image`、`image_url`、`images`、`image_urls`、`reference_images` 和 JSON `input_reference`，并在入站统一归一到 `reference_image_urls`；`image` 还兼容 `{ "url": "..." }`。这些别名不会出现在模型请求字段或新示例中。

历史 `seconds` 仍作为 `duration` 的兼容别名；两者同时传入时必须一致。历史 `video_url` 仍作为单条 `reference_videos` 的兼容输入。兼容字段只在入站解析，vendor 出站只能读取归一化值并按各自协议适配。

历史 `reference_mode` 与 `metadata` 仍可被请求解析，但不属于统一视频契约，也不会进入 canonical 出站视图。参考模式由 vendor 根据模型能力和素材字段自动推导：只有 `first_image_url` / `last_image_url` 表示首尾帧，普通参考数组由模型选择全能参考或普通素材模式。

## JSON 示例

```bash
curl -X POST "https://newapi.example.com/v1/videos" \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "seedance-2.0-720p",
    "prompt": "雨夜城市街道，电影感镜头缓慢推进",
    "duration": 8,
    "aspect_ratio": "16:9",
    "resolution": "720p",
    "reference_image_urls": ["https://example.com/ref.png"]
  }'
```

## Multipart 兼容（已弃用）

仅为避免破坏已有客户，服务端暂时接受历史 multipart 文件请求；新接入不得使用。新前端必须先将参考素材直传临时对象存储，再通过 JSON canonical URL 字段提交。**Omni V2V 对客户统一走 JSON `reference_videos`**，NewAPI 出站映射上游协议（参见 [docs.oairegbox.cc](https://docs.oairegbox.cc/)）。

| 字段 | 说明 |
|------|------|
| `model` | 模型名 |
| `prompt` | 描述 |
| `input_reference` / `input_reference[]` | 画布本地上传参考图文件 |

multipart 兼容入口后续会在完成调用统计和迁移通知后下线；不得为新模型新增 multipart 能力。JSON URL 方案适用于包括 V2V `reference_videos` 在内的所有新调用。

## 轮询

```bash
curl "https://newapi.example.com/v1/videos/{task_id}" \
  -H "Authorization: Bearer sk-xxx"
```

`status`：`queued` | `in_progress` | `completed` | `failed`

完成后取片：`data[0].url` 或 `GET /v1/videos/{task_id}/content`

建议轮询间隔 5–10 秒，客户端总等待时间建议 **≥30 分钟**（排队或复杂任务可能更长，勿过早判定超时）。

## 响应对象

| 字段 | 说明 |
|------|------|
| `id` | 任务 ID |
| `object` | `"video"` |
| `model` | 客户端提交的 public 模型名 |
| `status` | 任务状态 |
| `progress` | 进度 |
| `created_at` | Unix 秒 |
| `data` | 成功时含结果 URL |
| `error.message` | 失败原因 |

## 模型能力矩阵

各模型支持哪些 optional 字段、参考素材数量上限、计费方式，见模型广场 API 文档页的「能力卡片」，由 `video_ui_params` profile 驱动，不在本文重复。

运营/避坑补充文档（非 API 契约）：[`infinite-canvas/docs/dev/models/`](../infinite-canvas/docs/dev/models/)
