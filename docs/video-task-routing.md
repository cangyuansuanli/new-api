# 视频任务路由与轮询

本文描述 OpenAI Video 渠道族（`oaivideo`）以及 Adobe2API 视频模型的任务生命周期、模型路由与轮询行为。

## 目录结构

```
relay/channel/task/oaivideo/
├── router/           # 门面适配器（OpenAI/Sora/Custom 视频渠道入口）
├── registry/         # 模型 → Vendor 注册表
├── shared/           # FetchVideoTask、响应解析、multipart 透传
└── vendors/
    ├── manju/
    ├── chatvideo/    # 聚合线路 chat 上游，对外仍是统一任务
    ├── grok/         # 119337 generations endpoint + envelope normalization
    ├── geeknowgrok/  # Geeknow grok-imagine-video* via /v1/videos JSON
    ├── seedanceoairegbox/ # cy-sd1 → OAIREGBox flat /v1/videos
    ├── seedancetengda/    # cy-sd2 → Tengda content[] JSON
    ├── seedanceleonardo/  # cy-sd4 → Leonardo flat /v1/videos
    ├── sd5/               # cy-sd5 → Adobe2API generations
    ├── adobe/        # Adobe typed video endpoint + strict JSON normalization
    └── defaultvideo/ # sora-2 等标准 OpenAI Video 兜底
```

## 统一轮询流水线

所有视频模型共用 [`service/task_polling.go`](../service/task_polling.go) 中的 `TaskPollingLoop`（每 15 秒）：

1. `FetchTask` — Router 使用任务保存的 internal / upstream 模型重新选择 vendor；标准线路与 Geeknow Grok 请求 `GET {baseUrl}/v1/videos/{upstream_task_id}`，119337 Grok 请求 `/v1/video/generations/{upstream_task_id}`
2. `ParseTaskResultForTask` / `ParseTaskResult` — 优先由任务对应 Vendor 归一化上游 JSON；仅当专用解析未识别状态时才回退通用 `{code,data}` 任务响应解析，避免包裹结构“可反序列化但丢失结果 URL”
3. 写 DB（CAS `UpdateWithStatus`）
4. `AdjustBillingOnComplete` — 按 Vendor 结算差额

历史任务若曾把自身 `/v1/videos/{id}/content` 错写为 `result_url`，内容代理会仅在检测到该自引用时从原始任务响应恢复真实上游 URL；正常 CDN/上游结果地址不会被覆盖。

按秒 OAIREGBox Seedance 模型必须显式传 `duration`（JSON 或 multipart），范围为 4–15 秒整数。multipart 归一化必须读取 `duration` 本身，不能只读取兼容字段 `seconds`。任务成功且视频已转存时，系统从 MP4 元数据提取实际秒数写入 `usage.seconds`，终态结算优先使用实际成片时长；缺失或越界时长在提交前拒绝，禁止静默按 4 秒兜底。

轮询循环与单任务处理均带 `recover`，避免一次 panic 永久停摆。

Leonardo `cy-sd4-*` 渠道在插件主轮询窗口结束后仍返回 `in_progress`：插件内部转为 `delayed` 并低频追踪原 `generation_id`，NewAPI 不得将其提前改为失败或重新提交。NewAPI 全局任务清理由 `TASK_TIMEOUT_MINUTES` 控制（默认 1440 分钟），应显著长于插件主轮询窗口。

路由恢复必须兼容历史任务缺少 `ChannelMeta`、`UpstreamModelName` 或计费快照的情况：缺失字段按空值处理并回退到 `OriginModelName` / 默认视频解析器，禁止通过嵌入的空指针字段直接取值。共享请求转换同样将 `nil` 视为空值，不能把它序列化为 `"<nil>"` 后误判为客户显式参数。

## 客户端统一 API 契约

公开视频客户端只使用以下三个端点：

| 方法 | 路径 | 用途 |
|------|------|------|
| `POST` | `/v1/videos` | 创建视频任务，支持 `application/json` 与 `multipart/form-data` |
| `GET` | `/v1/videos/{task_id}` | 查询任务状态与结果 |
| `GET` | `/v1/videos/{task_id}/content` | 下载已完成任务的成片 |

创建请求的公共字段如下；模型不支持的可选字段由对应 profile 隐藏，并由服务端 vendor 校验：

| 字段 | 类型 | 说明 |
|------|------|------|
| `model` | string | 必填，模型广场展示的 public 模型名 |
| `prompt` | string | 必填，视频提示词 |
| `duration` / `seconds` | integer 或整数字符串 | 可任选其一；同时传入时必须一致 |
| `aspect_ratio` | string | 如 `16:9`、`9:16`、`1:1` |
| `resolution` | string | 如 `480p`、`720p`、`1080p` |
| `seed` | integer | 可复现种子；当前仅 SD5 Seedance 支持，显式 `0` 也会透传 |
| `generate_audio` | boolean | 是否生成音频，取决于模型能力 |
| `video_url` | string | 参考视频公网 URL，仅支持已声明视频编辑能力的模型 |
| `image` / `images` / `image_urls` / `reference_image_urls` | string、string[]；`image` 兼容 `{url}` | JSON 参考图；支持 HTTPS URL，具体数量由模型 profile 决定 |
| `input_reference` | file 或 file[] | multipart 参考素材 |
| `metadata` | object | 已登记 vendor 的扩展参数；不得用于选择上游协议或路径 |

JSON 示例：

```json
{
  "model": "grok-video",
  "prompt": "一辆跑车穿过雨夜城市",
  "duration": 6,
  "aspect_ratio": "9:16",
  "resolution": "720p",
  "image_urls": ["https://example.com/reference.png"]
}
```

提交与轮询响应都使用统一视频对象：`id`、`object: "video"`、`model`、`status`、`progress`、`created_at`；成功时可通过响应结果 URL 或 `/content` 取片，失败时读取 `error.message`。`status` 只向客户暴露 `queued`、`in_progress`、`completed`、`failed`。

Leonardo `cy-sd4-*` 的失败消息按可操作原因归一化：号池并发占满、冷却中或模型过载显示服务暂时不可用；上传媒体返回 `DURATION_TOO_LONG` 时提示缩短参考音视频；上游只返回 `FAILED` 且没有失败字段/输出时，明确说明上游未提供具体原因，并建议简化提示词、减少或更换参考素材。如请求包含参考素材，静默失败表示素材已完成上传和基础格式校验，不代表生成阶段内容审核、提示词与素材组合或模型稳定性一定通过。Cookie、账号及内部上游错误不得透传给客户。

时间字段对外统一为整数 Unix 秒。上游若返回带小数的 Unix 秒，`oaivideo/shared` 会在协议边界截断为整数，不能因供应商时间精度差异导致任务提交或轮询失败。

`/v1/chat/completions`、`/v1/video/generations`、`/v1/videos/generations` 等均是部分供应商的内部上游协议，不属于公开视频 API，也不得出现在客户调用示例中。

## 统一时长参数契约

对外只提供上述统一视频任务协议。`duration` / `seconds` 都表示整数秒：

- 只传其中一个时，`relay/common` 在入口归一化为同一内部时长值。
- 同时传入两个且值不一致时，提交阶段返回 `invalid_duration`，禁止计费与生成使用不同时长。
- 内部不通过原始 body 透传别名；由 vendor 边界输出上游契约要求的单一字段。

| Vendor | 上游时长字段 | 说明 |
|--------|------------------|------|
| default（Sora 等标准 OpenAI Video） | `seconds` | 上游 `/v1/videos` 契约 |
| Grok generations (119337) | `seconds` (integer) | 上游 `/v1/video/generations`；参考图统一为 `image_urls` 字符串数组 |
| Geeknow Grok | `seconds` (string) | 上游 `/v1/videos`；单图 `image`，多图 `images`；`resolution` 为 `480P`/`720P` |
| Seedance | `duration` | OAIREGBox 按秒视频契约 |
| Adobe | `duration` | Adobe typed `/v1/videos/generations` 严格 schema |

JSON 的其他字段和 multipart 文件必须保留；仅模型名与时长别名在 vendor 边界发生转换。
SD5 Seedance vendor 还会将可选整数 `seed` 原样传给 Adobe2API，包括显式零值；其他
Adobe Sora/Veo 模型继续过滤不支持的 seed。

Seedance 2.0 的参考图统一使用 `reference_image_urls`（含单图）；参考视频和参考音频均为可选且可独立使用；仅传 `prompt` 即为文生视频。Relay 在 registry 层按线路拆为独立 vendor：`seedance-oairegbox`（cy-sd1）、`seedance-tengda`（cy-sd2）、`seedance-leonardo`（cy-sd4）、`sd5-seedance`（cy-sd5）。Seedance vendor 不得因 `reference_videos` / `reference_audios` 存在而强制要求参考图，仍需保留各 profile 的数量、大小、时长与首尾帧互斥校验。

Adobe2API 视频现在属于标准视频任务族：对外使用 `POST /v1/videos` + `GET /v1/videos/{id}`，Adobe vendor 内部将创建请求映射为上游 `POST /v1/videos/generations`，并使用上游 `GET /v1/videos/{id}` 轮询。Adobe 任务直接进入通用任务表和通用轮询，不再创建独立 worker，也不再包装成 chat。

模型广场与 API 文档中的 Adobe 视频元数据必须同步使用 `openai-video` endpoint、`videos-json-async` UI profile 和 `dispatch_mode=async`；旧 `*-chat` profile 与 `/v1/chat/completions` 示例属于迁移前遗留数据，不能继续对客户展示。仅修正文档/profile 而不调整售价时，运行 `python3 scripts/seed_adobe2api_video_api_doc.py --docs-only`。

## 模型 → Vendor 路由表

注册逻辑：[`relay/channel/task/oaivideo/registry/registry.go`](../relay/channel/task/oaivideo/registry/registry.go)

| internal 模型前缀 | Vendor | 提交差异 | 轮询解析 |
|-------------------|--------|----------|----------|
| `manju-openai-sora*` | Manju | chat/completions 转换 | Manju 响应形（`platform:sora2` 等） |
| `cy-vid2-*` / `cy-sd1-grok-video*` | Chat Video | 内部转 chat/completions，读 SSE/JSON 视频 URL | 提交时即归一化为已完成任务 |
| `cy-gv1-grok-video*` + upstream `grok-image-video*` | Grok generations | 严格 JSON → `/v1/video/generations` | generations envelope → OpenAI Video 形 |
| `cy-gv1-grok-video*` + upstream `grok-imagine-video*` | Geeknow Grok | 严格 JSON → `/v1/videos` | OpenAI Video 形 |
| `cy-sd1-seedance*` | seedance-oairegbox | cy-sd1 白名单 flat JSON → OAIREGBox `/v1/videos` | OpenAI Video 形 |
| `cy-sd4-seedance*` | seedance-leonardo | cy-sd4 白名单 flat JSON → Leonardo `/v1/videos` | OpenAI Video 形 |
| `cy-sd5-seedance*` | SD5 Seedance | 按模型名前缀独立路由，不依赖 Adobe 渠道 ID 或模型映射；seed、9 图 / 3 视频 / 3 音频（合计最多 12）严格 JSON → `/v1/videos/generations` | `video.generation` → OpenAI Video 形 |
| `cy-sd2-seedance*` / `tengd-seedance*` | seedance-tengda | Tengda flat → `content[]` JSON | OpenAI Video 形 |
| `adobe-*sora*` / `adobe-*veo*` | Adobe | 严格 JSON → `/v1/videos/generations` | `video.generation` → OpenAI Video 形 |
| 其他（Sora 等） | default | 标准 OpenAI Video | OpenAI Video 形 |

门面适配器：[`relay/channel/task/oaivideo/router/adaptor.go`](../relay/channel/task/oaivideo/router/adaptor.go)

## 任务初始状态

视频任务提交成功后不再长期停留在 `NOT_START`：

- 默认 `SUBMITTED` / `10%`
- 若上游响应 `status=queued` → `QUEUED` / `20%`
- 实现：[`model.ApplySubmittedStatusFromUpstreamData`](../model/task.go)，在 [`controller/relay.go`](../controller/relay.go) 插入任务前调用

提交接口只负责取得并返回任务 ID，不等待生成完成；`queued`、`in_progress` 以及 Leonardo 插件内部的 `delayed` 都是非终态。

## 新增视频模型 Checklist

1. 在 `registry.ResolveWithChannel` 注册匹配规则（按 Adobe channel/model 进入独立 vendor adaptor，不复制任务生命周期）
2. 确认提交阶段：走 Manju / Grok / Seedance / Adobe 转换，还是 default 透传
3. 确认轮询：`FetchTask` 是否仍为 `/v1/videos/{id}`；若路径不同，必须由任务模型重新选择 vendor
4. 确认计费：`AdjustBillingOnComplete` 按秒还是按次
5. 补充 `registry` / `router` 单测
6. 源站抽一条任务验收状态推进与 quota

Adobe 额外检查：确认严格 JSON 只发送上游允许字段；确认提交时选中的多 Key 已写入私有任务数据，并由通用轮询复用同一 Key。

## 相关文件

- `relay/channel/task/oaivideo/registry/` — 路由注册表
- `relay/channel/task/oaivideo/shared/` — 共享解析与 `FetchVideoTask`
- `relay/channel/task/oaivideo/vendors/{manju,seedance,adobe,defaultvideo}/` — 子适配器
- `relay/channel/task/README.md` — task 目录总览（L1/L2 分层）
- `service/task_polling.go` — 轮询主循环

图像与视频跨能力的完整审计见 [`media-request-chain-audit.md`](media-request-chain-audit.md)。
