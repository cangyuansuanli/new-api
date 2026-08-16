# 统一图像 API 参考

面向外部 API 客户与无限画布共用的 **唯一** 图像契约。前端提交 canonical 参数；模型能力校验、精确尺寸映射、上游路径和 vendor 字段均由 NewAPI 负责。

> 内部执行链路见 [`media-request-chain-audit.md`](media-request-chain-audit.md)。

## 公共端点

| 方法 | 路径 | 请求格式 | 用途 |
|------|------|----------|------|
| `POST` | `/v1/images/generations` | JSON | 文生图 |
| `POST` | `/v1/images/edits` | JSON | 参考图编辑 / 蒙版编辑 |
| `GET` | `/v1/images/generations/{task_id}` | - | 查询生成任务 |
| `GET` | `/v1/images/edits/{task_id}` | - | 查询编辑任务 |
| `GET` | `/v1/images/{task_id}/content` | - | 获取已归档结果 |

鉴权：`Authorization: Bearer sk-xxx`

异步请求传 `async: true`；同步请求不传 `async` 或传 `false`。请使用 `/v1/images/edits` 作为统一编辑入口；旧版 `/v1/edits` 仅为兼容转发。

## Canonical 参数

生成与编辑共用：`model`、`prompt`、`n`、`size`、`quality`、`background`、`output_format`、`output_compression`、`moderation`、`response_format`、`async`、`stream`。编辑请求额外使用 `images` HTTPS URL 数组和可选的 `mask` HTTPS URL。

新客户端应先通过 Presigned URL 把本地图片直传对象存储，再提交 URL，不应把 Blob、data URI 或文件流量经过业务 API。为避免破坏已上线客户，服务端仍在入站层接受历史文件请求；请求体受限且大表单会写临时文件而非常驻内存。仅当上游渠道要求文件表单时，NewAPI 才在出站边界把 canonical URL 下载并转换为其私有协议。

历史文件请求只是入站迁移层，不是新公共契约。JSON URL 与历史请求都先归一为同一 `ImageRequest` 控制参数并进入相同的模型映射、能力校验和渠道分发链。

`size` 保留客户选择的语义：比例使用 `1:1`、`16:9`、`9:16`，自定义尺寸使用 `1024x1024`。`quality` 使用 `low`、`medium`、`high`。前端不得把比例和质量提前换算为某个供应商的像素尺寸。

以下字段不是新公共契约，不应由新客户端发送：`aspect_ratio`、`image_size`、`output_resolution`、Gemini `extra_body.google.image_config`、chat `messages`。已上线第三方客户携带的扩展字段继续按兼容逻辑解析，但新渠道不能依赖这些字段。

## 出站职责

固定处理顺序：public 模型名映射到 internal 名 → 渠道分发与 `model_mapping` → `imagevendor.ValidateRequest` → `imagevendor.ApplyRequestPatch` → channel adaptor 转换 → `param_override` → 上游。

- `relay/imagevendor/`：按 internal 模型和已选渠道声明能力校验、尺寸档位映射、字段裁剪与 R2 策略。
- `relay/channel/*`：上游协议、路径、私有请求体与响应转换。
- `service/image_r2_rehost.go`：只负责结果下载、识别与对象存储归档。
- 前端 profile：只决定可见参数、选项和数量限制，不选择上游协议或 vendor builder。

新增渠道时优先注册 `imagevendor.Descriptor`；只有上游 body 或 endpoint 不兼容标准 Image API 时才扩展 channel adaptor。禁止在 controller、前端或通用 R2 服务新增模型名前缀分支。

## 示例

`gpt-image-2` 是稳定的 API 入站路由，当前绑定 `cy-img1-gpt-image-2`；模型广场使用独立展示名 `img1-gpt-image-2`。后续切换渠道只更新 routing alias，不改客户请求模型名。

```bash
curl -X POST "https://newapi.example.com/v1/images/generations" \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-image-2",
    "prompt": "一只橘猫坐在窗台上，午后阳光",
    "size": "1:1",
    "quality": "high",
    "n": 1,
    "response_format": "url",
    "async": true,
    "stream": false
  }'
```

## 轮询与响应

异步任务查询 `GET /v1/images/generations/{task_id}` 或 `GET /v1/images/edits/{task_id}`。`status` 为 `queued`、`in_progress`、`completed`、`failed`；建议间隔 5–10 秒，总等待时间至少 30 分钟。

同步成功返回标准 `data` 数组：

```json
{
  "created": 1715923200,
  "data": [{ "url": "https://example.com/image.png" }]
}
```

各模型支持的参数、同步/异步模式和档位只由该模型自己的 `models.api_doc` 维护。`image_ui_params` 仅驱动画布控件，不生成、合并或补全 API 文档；缺少有效 `api_doc` 时不展示模型文档。运营补充见 `infinite-canvas/docs/dev/models/`。
