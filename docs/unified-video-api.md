# 统一视频 API 参考

面向外部 API 客户与无限画布共用的 **唯一** 视频契约。各模型差异见模型广场「能力卡片」（价格、参考素材上限、是否 multipart 等），本文只描述通用 endpoint 与 canonical 字段。

> 内部路由与 vendor 出站见 [`video-task-routing.md`](video-task-routing.md)。

## 端点

| 方法 | 路径 | 用途 |
|------|------|------|
| `POST` | `/v1/videos` | 创建视频任务（`application/json` 或 `multipart/form-data`） |
| `GET` | `/v1/videos/{task_id}` | 查询任务状态与结果 |
| `GET` | `/v1/videos/{task_id}/content` | 下载已完成任务的成片 |

鉴权：`Authorization: Bearer sk-xxx`

## Canonical 请求字段（JSON）

默认使用 JSON；仅当模型能力卡片标注「需 multipart」时改用 `multipart/form-data`（见下文）。

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `model` | string | 是 | 模型广场 public 名 |
| `prompt` | string | 是 | 视频描述 |
| `duration` | integer | 否 | 时长（秒）；与 `seconds` 二选一，同时传入须一致 |
| `seconds` | integer / string | 否 | `duration` 的兼容别名 |
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
| `video_url` | string | 否 | 单条参考视频（`reference_videos` 的兼容别名） |

### 字段别名（仍接受，服务端归一化）

参考图：`image`、`image_url`、`images`、`image_urls` 均等价于 `reference_image_urls`。

时长：`duration` 与 `seconds` 在入口合并为同一内部值。

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

## Multipart 例外

仅当模型 profile 声明 `wireFormat: multipart-form`（如画布本地上传参考图文件）时使用 multipart；**Omni V2V 对客户统一走 JSON `reference_videos`**，NewAPI 出站映射上游协议（参见 [docs.oairegbox.cc](https://docs.oairegbox.cc/)）。

| 字段 | 说明 |
|------|------|
| `model` | 模型名 |
| `prompt` | 描述 |
| `input_reference` / `input_reference[]` | 画布本地上传参考图文件 |

JSON 能表达的场景优先用 URL 引用（含 V2V 的 `reference_videos`）；multipart 不用于客户侧 Omni V2V。

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
