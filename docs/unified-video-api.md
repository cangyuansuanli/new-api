# 统一视频 API 参考

面向外部 API 客户与无限画布共用的视频入站契约。本文定义唯一的统一端点、canonical 字段和任务生命周期；每个模型自己的 `models.api_doc` 只能从这套字段中声明支持子集、范围与完整示例。

> 内部路由与 vendor 出站见 [`video-task-routing.md`](video-task-routing.md)。

## 端点

| 方法 | 路径 | 用途 |
|------|------|------|
| `POST` | `/v1/videos` | 创建视频任务（新调用使用 `application/json`） |
| `GET` | `/v1/videos/{task_id}` | 查询任务状态与结果 |
| `GET` | `/v1/videos/{task_id}/content` | 下载已完成任务的成片 |

鉴权：`Authorization: Bearer sk-xxx`

## Canonical 请求字段

统一入口的新客户端使用 `application/json`。本地素材先直传对象存储，再把 HTTPS URL 写入统一参考字段。

| 字段 | 类型 | 说明 |
|------|------|------|
| `model` | string | 必填，模型广场 public 名 |
| `prompt` | string | 必填，视频描述 |
| `duration` | integer | 时长秒数 |
| `aspect_ratio` | string | 画幅比例 |
| `resolution` | string | 清晰度档位 |
| `size` | string | 像素尺寸 |
| `seed` | integer | 可复现种子 |
| `generate_audio` | boolean | 是否生成音频 |
| `reference_image_urls` | string[] | 参考图 HTTPS URL |
| `reference_videos` | string[] | 参考视频 HTTPS URL |
| `reference_audios` | string[] | 参考音频 HTTPS URL |
| `first_image_url` | string | 首帧 HTTPS URL |
| `last_image_url` | string | 尾帧 HTTPS URL |

上述是统一 schema，不表示每个模型全部支持。具体模型只能使用其独立文档列出的子集和范围，未列字段不要发送。

服务端仍在入站层解析历史客户格式并归一化，出站 vendor 只读取标准 DTO 后按各渠道协议适配。兼容字段和传输格式不属于新公共契约，不展示在客户文档中。

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

## 单模型文档

每个启用模型必须独立提供 `dispatch_mode`、统一端点、完整 canonical 请求示例、支持字段、取值范围和响应示例。模型文档不得出现统一 schema 之外的字段。`video_ui_params` 只驱动画布控件，不生成、合并或补全 API 文档；模型缺少有效 `api_doc` 时不展示文档。

运营/避坑补充文档（非 API 契约）：[`infinite-canvas/docs/dev/models/`](../infinite-canvas/docs/dev/models/)
