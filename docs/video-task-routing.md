# 视频任务路由与轮询

本文描述 OpenAI Video 渠道族（`oaivideo`）以及 Adobe2API 视频模型的任务生命周期、渠道协议路由与轮询行为。

## 目录结构

```
relay/channel/task/oaivideo/
├── router/           # 门面适配器（OpenAI/Sora/Custom 视频渠道入口）
├── registry/         # 已分发渠道 + 模型映射结果 → Vendor 注册表
├── shared/           # FetchVideoTask、响应解析、multipart 透传
└── vendors/
    ├── chatvideo/    # 聚合线路 chat 上游，对外仍是统一任务
    ├── grok/         # 119337 generations endpoint + envelope normalization
    ├── geeknowgrok/  # Geeknow grok-imagine-video* via /v1/videos JSON
    ├── seqnode/      # Seqnode Grok generations + protected content
    ├── seedanceoairegbox/ # cy-sd1 → OAIREGBox flat /v1/videos
    ├── seedancetengda/    # cy-sd2 → Tengda content[] JSON
    ├── seedanceleonardo/  # cy-sd4 → Leonardo flat /v1/videos
    ├── seedanceheygen/    # cy-sd6 双 SKU → 固定 720p/1080p flat /v1/videos
    ├── omnii2v/         # cy-sd1 omni-fast* flat → upstream images[]（JSON 多图，勿 multipart input_reference）
    ├── omniv2v/         # cy-sd1 omni-fast-v2v* flat → upstream videos/images
    ├── adobe/        # Adobe typed video endpoint + strict JSON normalization
    └── defaultvideo/ # sora-2 等标准 OpenAI Video 兜底
```

## 统一轮询流水线

所有视频模型共用 [`service/task_polling.go`](../service/task_polling.go) 中的 `TaskPollingLoop`（每 15 秒）：

1. `FetchTask` — 新任务使用提交时持久化的 `properties.task_vendor` 选择 vendor；历史任务缺少该字段时，才使用 channel ID + internal/upstream 模型恢复。标准线路、Geeknow Grok 与 Seqnode 请求 `GET {baseUrl}/v1/videos/{upstream_task_id}`，119337 Grok 请求 `/v1/video/generations/{upstream_task_id}`
2. `ParseTaskResultForTask` / `ParseTaskResult` — 优先由任务对应 Vendor 归一化上游 JSON；仅当专用解析未识别状态时才回退通用 `{code,data}` 任务响应解析，避免包裹结构“可反序列化但丢失结果 URL”
3. 成功态如需补齐受保护成片来源，由任务对应 vendor 返回 URL + 临时请求头，通用 R2 服务只负责下载和转存
4. 写 DB（CAS `UpdateWithStatus`）
5. `AdjustBillingOnComplete` — 按 Vendor 结算差额

历史任务若曾把自身 `/v1/videos/{id}/content` 错写为 `result_url`，内容代理会仅在检测到该自引用时从原始任务响应恢复真实上游 URL；正常 CDN/上游结果地址不会被覆盖。

按秒 OAIREGBox Seedance 模型必须显式传 `duration`（JSON 或 multipart），范围为 4–15 秒整数。multipart 归一化必须读取 `duration` 本身，不能只读取兼容字段 `seconds`。任务成功且视频已转存时，系统从 MP4 元数据提取实际秒数写入 `usage.seconds`，终态结算优先使用实际成片时长；缺失或越界时长在提交前拒绝，禁止静默按 4 秒兜底。

轮询循环与单任务处理均带 `recover`，避免一次 panic 永久停摆。

Leonardo `cy-sd4-*` 渠道在插件主轮询窗口结束后仍返回 `in_progress`：插件内部转为 `delayed` 并低频追踪原 `generation_id`，NewAPI 不得将其提前改为失败或重新提交。NewAPI 全局任务清理由 `TASK_TIMEOUT_MINUTES` 控制（默认 1440 分钟），应显著长于插件主轮询窗口。

路由恢复必须兼容历史任务缺少 `ChannelMeta`、`UpstreamModelName` 或计费快照的情况：缺失字段按空值处理并回退到 `OriginModelName` / 默认视频解析器，禁止通过嵌入的空指针字段直接取值。共享请求转换同样将 `nil` 视为空值，不能把它序列化为 `"<nil>"` 后误判为客户显式参数。

## 客户端统一 API 契约

客户契约文档：[`unified-video-api.md`](unified-video-api.md)（本文以下字段表与内部路由说明供开发维护）。

公开视频客户端只使用以下三个端点：

| 方法 | 路径 | 用途 |
|------|------|------|
| `POST` | `/v1/videos` | 创建视频任务；新调用仅使用 `application/json`，multipart 只保留历史兼容 |
| `GET` | `/v1/videos/{task_id}` | 查询任务状态与结果 |
| `GET` | `/v1/videos/{task_id}/content` | 下载已完成任务的成片 |

创建请求的公共字段如下；模型不支持的可选字段由对应 profile 隐藏，并由服务端 vendor 校验：

| 字段 | 类型 | 说明 |
|------|------|------|
| `model` | string | 必填，模型广场展示的 public 模型名 |
| `prompt` | string | 必填，视频提示词 |
| `duration` | integer | 视频时长（秒） |
| `aspect_ratio` | string | 如 `16:9`、`9:16`、`1:1` |
| `resolution` | string | 如 `480p`、`720p`、`1080p` |
| `seed` | integer | 可复现种子；Veo 3.1、Veo 3.1 Fast、Kling 3.0/Omni、SD5 Seedance 支持，Gemini Omni 不支持 |
| `generate_audio` | boolean | 是否生成音频，取决于模型能力 |
| `reference_videos` | string[] | 参考视频公网 URL 数组，仅支持已声明视频编辑能力的模型 |
| `reference_audios` | string[] | 参考音频公网 URL 数组，仅支持已声明音频参考能力的模型 |
| `reference_image_urls` | string[] | JSON 参考图 HTTPS URL；具体数量由模型 profile 决定 |
| `first_image_url` | string | 首帧参考图；只用于模型声明的首尾帧模式 |
| `last_image_url` | string | 末帧参考图；只用于模型声明的首尾帧模式 |
| `input_reference` | file 或 file[] | multipart 参考素材 |

`seconds`、`image`、`image_url`、`images`、`image_urls`、`reference_images`、JSON `input_reference` 与 `video_url` 仅为历史客户兼容字段。它们在 `TaskSubmitReq` 入站时归一，不能出现在新客户文档、前端请求构造或 vendor 业务逻辑中。

JSON 示例：

```json
{
  "model": "grok-video",
  "prompt": "一辆跑车穿过雨夜城市",
  "duration": 6,
  "aspect_ratio": "9:16",
  "resolution": "720p",
  "reference_image_urls": ["https://example.com/reference.png"]
}
```

提交与轮询响应都使用统一视频对象：`id`、`object: "video"`、`model`、`status`、`progress`、`created_at`；成功时可通过响应结果 URL 或 `/content` 取片，失败时读取 `error.message`。`status` 只向客户暴露 `queued`、`in_progress`、`completed`、`failed`。

Leonardo **`cy-sd4-seedance*`**：参考素材校验与失败文案在 **leonardo-web2api**（`referencemedia` + `clienterror`）；NewAPI **不重写** 任务 `fail_reason` 与轮询 JSON 中的 `error.message`（见 [`channel-seedance-leonardo.md`](channel-seedance-leonardo.md)、[`leonardo-web2api/docs/models/seedance-2.0.md`](../../leonardo-web2api/docs/models/seedance-2.0.md)）。其他渠道或非 cy-sd4 路径仍走 `service/clienterror` 单点翻译。

## 公共名与路由别名

模型命名分为两个公共边界：`model_routing_aliases` 提供客户首选的稳定 API 名并解析到实际 internal ability，`model_public_aliases` 提供可选的指定渠道名称。管理后台“模型命名与路由”区可维护两类映射；路由名写入时要求目标已存在于 `abilities`，且入站名不得占用渠道名称。渠道间的 `priority`、`weight` 和 retry 仍在渠道管理中配置。完整契约见 [`model-naming-routing.md`](model-naming-routing.md)。

`/v1/models` 与模型广场默认列出可用 routing alias，同时列出 `hidden_from_marketplace=false` 的渠道展示名。这样客户既可使用稳定中性路由，也可显式选择展示中的特定渠道。`cy-*` internal 名保留入站兼容，但不进入任何公共目录；隐藏渠道展示名只影响目录，已有客户仍可继续调用该别名。

`model_channel_prefixes` 仅用于识别必须配置显式展示名的 internal 命名空间，不再自动剥离或生成公开名。匹配这些前缀但缺少 `model_public_aliases` 的 enabled model 会从 `/pricing` 与 `/v1/models` 隐藏，并在 registry 状态和管理页列出。任何新的渠道 internal 模型必须在启用 ability 前写入明确展示名。

当前 Seedance 中性路由基线如下：

| 入站模型名 | internal ability | 上游线路 |
|------------|------------------|----------|
| `seedance-2.0` | `cy-sd7-seedance-2.0-720p` | Magica SD7 |
| `seedance-2.0-fast` | `cy-sd4-seedance-2.0-fast` | Leonardo SD4 |
| `seedance-2.0-mini` | `cy-sd4-seedance-2.0-mini` | Leonardo SD4 |
| `seedance-2.0-720p` | `cy-sd7-seedance-2.0-720p` | Magica SD7 |
| `seedance-2.0-1080p` | `cy-sd7-seedance-2.0-1080p` | Magica SD7 |
| `seedance-2.5-480p` | `cy-sd4-seedance-2.5-480p` | Leonardo SD4 |
| `seedance-2.5-720p` | `cy-sd4-seedance-2.5-720p` | Leonardo SD4 |

其他历史中性名也必须作为入站路由维护，而不是占用品牌展示名：`grok-video*` 当前绑定 GV2、`omni-*` / `veo-clean` 当前绑定 SD1、`gemini-music` 当前绑定 AU1。模型广场分别保留 `gv2-*`、`sd1-*`、`au1-*` 卡片；切换供应渠道时只更新 routing alias 的 `internal_name`。

修改路由别名后 controller 会同步刷新内存 registry；直接执行 SQL 时则等待定时刷新或滚动重启。第一版仅提供显式 CRUD，不根据上游错误自动 failover。

Seedance 2.5 的广场展示名固定为 `sd4-seedance-2.5-480p` / `sd4-seedance-2.5-720p`；对应中性名 `seedance-2.5-480p` / `seedance-2.5-720p` 仅作为入站路由。两类名称不得同名，创建和更新接口会双向拒绝冲突。

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
| Seedance HeyGen | `duration` | `cy-sd6` 两个产品分别强制 720p / 1080p，客户参数不能切换档位 |
| Adobe | `duration` | Adobe typed `/v1/videos/generations` 严格 schema |

JSON 业务字段不得从原始 body 透传。入站先生成 canonical `TaskSubmitReq`，vendor 再按上游白名单映射字段；统一视频接口不接受任意 vendor 扩展字段。历史 multipart 文件仍由共享 serializer 保留文件内容与元数据。
统一 Adobe vendor 会按模型契约处理 SD5 Seedance，并将可选整数 `seed` 原样传给 Adobe2API，包括显式零值；其他
SD5 出站契约同时保留 `duration`、`aspect_ratio`、`resolution`、`generate_audio`、
参考图、参考视频和参考音频。vendor 根据模型能力自动为普通参考素材选择并发送 `reference_mode=media`；仅显式 `first_image_url` / `last_image_url` 选择 `frame`。首尾帧与 media 全能参考互斥，验收必须
分别使用合法组合，不能把互斥字段拼成一个“全参数”请求。
Adobe Sora/Veo 模型继续过滤不支持的 seed。

Seedance 2.0 的参考图统一使用 `reference_image_urls`（单图 vendor 可映射为 `image_url`）；`image`、`images`、`image_urls` 和 `image_url` 是等价的公开别名，均会归一化为参考图。Relay 在 registry 层按线路拆分：`seedance-oairegbox`（cy-sd1）、`seedance-tengda`（cy-sd2）、`seedance-leonardo`（cy-sd4）、`adobe`（cy-sd5，与其他 Adobe 视频共用）、`seedance-heygen`（cy-sd6）、`seedance-magica`（cy-sd7）、`seedance-huabu`（cy-sd8）。`cy-sd6-seedance-2.0-720p` 与 `cy-sd6-seedance-2.0-1080p` 都映射上游 `seedance-2.0`，但由 vendor 分别强制 `resolution=720p` / `1080p`；不得让客户通过请求字段跨档。cy-sd6 支持最多 9 图、3 视频、1 音频且合计最多 12 个，音频不能单独使用，首尾帧必须成对并与多模态参考互斥。**cy-sd1 / cy-sd2 / cy-sd5** 仍在 NewAPI profile 或 adaptor 侧做数量/大小/互斥等出站校验；**cy-sd4 / cy-sd6** 业务规则分别由对应 web2api 上游校验。

公共图片别名必须在 `relay/common.TaskSubmitReq` 入口合并、去空并去重到 `Images`，vendor 只能消费该标准字段并渲染上游协议，不得再次从原始 JSON body 解析 `image` / `image_url` / `images` / `image_urls` / `reference_images` / `reference_image_urls`。`first_image_url` / `last_image_url` 是独立的首尾帧控制字段，不进入通用参考图归一化。

Seedance 各线路均声明成对 `first_image_url` + `last_image_url` 首尾帧能力，并在 vendor 出站层显式透传；首尾帧与普通参考图、参考视频、参考音频互斥。公共 `seedance-2.0` 路由继承当前目标线路的同一契约。文档同步脚本会校验 profile 已启用但文档漏字段、只声明单边帧字段，以及示例混用首尾帧和普通参考素材三类错误。

完整出站链路固定为：客户 JSON / multipart → `TaskSubmitReq.UnmarshalJSON` 兼容解析 → `Canonicalize` 清除别名 → `CanonicalVideoBody` 标准视图 → registry vendor 白名单映射 → 上游协议。当前 registry 的所有 vendor 都必须从标准 DTO 读取业务字段；新增 vendor 必须补充 canonical 输入到上游字段的测试。

Adobe2API 视频现在属于标准视频任务族：对外使用 `POST /v1/videos` + `GET /v1/videos/{id}`，Adobe vendor 内部将创建请求映射为上游 `POST /v1/videos/generations`，并使用上游 `GET /v1/videos/{id}` 轮询。Adobe 任务直接进入通用任务表和通用轮询，不再创建独立 worker，也不再包装成 chat。

模型广场与 API 文档中的 Adobe 视频元数据必须同步使用 `openai-video` endpoint、`videos-json-async` UI profile 和 `dispatch_mode=async`；旧 `*-chat` profile 与 `/v1/chat/completions` 示例属于迁移前遗留数据，不能继续对客户展示。仅修正文档/profile 而不调整售价时，运行 `python3 scripts/seed_adobe2api_video_api_doc.py --docs-only`。

## 渠道 → Vendor 路由

注册逻辑：[`relay/channel/task/oaivideo/registry/registry.go`](../relay/channel/task/oaivideo/registry/registry.go)

提交阶段在渠道分发和 `model_mapping` 完成后解析一次 vendor，并写入 `tasks.properties.task_vendor`。轮询、结果解析、成片获取、计费和响应转换都必须使用该持久化值，禁止按模型名重新猜测协议。模型匹配只用于提交时解析和历史任务兼容。

| 渠道 / internal 模型条件 | Vendor | 提交差异 | 轮询解析 |
|--------------------------|--------|----------|----------|
| `cy-vid2-*` / `cy-sd1-grok-video*` | Chat Video | 内部转 chat/completions，读 SSE/JSON 视频 URL | 提交时即归一化为已完成任务 |
| `cy-gv1-grok-video*` + upstream `grok-image-video*` | Grok generations | 严格 JSON → `/v1/video/generations` | generations envelope → OpenAI Video 形 |
| `cy-gv1-grok-video*` + upstream `grok-imagine-video*` | Geeknow Grok | 严格 JSON → `/v1/videos` | OpenAI Video 形 |
| `cy-gv2-grok-video` → `grok-imagine-video`；`cy-gv2-grok-video-1.5` → `grok-imagine-video-1.5`（当前分别在渠道 106/107） | Seqnode | 按完整 internal/upstream 模型精确配对，不绑定渠道 ID；JSON → `/v1/videos/generations` | `/v1/videos/{id}` 状态 + 鉴权 `/v1/videos/{id}/content` 成片转存 |
| `cy-sd1-seedance*` | seedance-oairegbox | cy-sd1 白名单 flat JSON → OAIREGBox `/v1/videos` | OpenAI Video 形 |
| `cy-sd1-omni-fast*` / upstream `omni-fast*` | omni-i2v | 公开 `reference_image_urls` / `images` / `image_url` → 上游 `images[]`（单图亦数组）；首尾帧 `first_image_url` / `last_image_url` 原样透传；成片转存时若同时返回 `data[].url`（`task_*`）与绝对 `video_url`（`vid-*`），vendor 优先 `vid-*` 以免 `task_*` 未就绪 409 | OpenAI Video 形 |
| `cy-sd1-omni-v2v*` / upstream `omni-fast-v2v*` | omni-v2v | 公开 JSON `reference_videos` / `reference_image_urls` → 上游 `videos` / `video_url` / `images`；legacy multipart 仅出站内部兼容 | OpenAI Video 形 |
| `cy-sd4-seedance*` | seedance-leonardo | flat JSON → Leonardo `/v1/videos` | OpenAI Video 形（校验/错误见 [`channel-seedance-leonardo.md`](channel-seedance-leonardo.md)） |
| `cy-sd6-seedance-2.0-720p` / `cy-sd6-seedance-2.0-1080p` + upstream `seedance-2.0` | seedance-heygen | 按完整 internal/upstream 模型精确配对，不绑定渠道 ID；严格 JSON 白名单，按 SKU 强制 720p/1080p | `/v1/videos/{id}` + 鉴权 `/content` 成片转存 |
| `cy-sd7-seedance-2.0-720p` / `cy-sd7-seedance-2.0-1080p` + upstream `seedance-2.0` | seedance-magica | 双 SKU 强制分辨率；有参考素材时 adaptor 自动改发 `seedance-2.0-reference` | `/v1/videos/{id}` + 鉴权 `/content` 成片转存（见 [`channel-seedance-magica.md`](channel-seedance-magica.md)） |
| `cy-sd8-seedance-2.0` → `sd2.0-933`；`cy-sd8-seedance-2.0-fast` → `sd-2.0-fast-v1` | seedance-huabu | 精确 internal/upstream 配对；卡脸版 9 图+3 视频+3 音频（参考图须遮眼）；快速版仅 9 图；`aspect_ratio`→`size`；单图→上游 `image` 字符串，多图→上游 `images` 数组 | `/v1/videos/{id}` + `result_url`；有 `result_url` 时 rehost 直链，否则回退鉴权 `/content` |
| `cy-sd5-seedance*` | Adobe2API Seedance | 与 Adobe 视频共用 `adobe` vendor；按内部模型契约输出 `media` 或 `frame`，支持 9 图 + 3 个视频/音频共享源位、全部素材最多 12；mapping 使用 `seedance-2.0` / `seedance-2.0-fast` | `video.generation` → OpenAI Video 形 |
| `cy-sd2-seedance*` / `tengd-seedance*` | seedance-tengda | Tengda flat → `content[]` JSON | OpenAI Video 形 |
| `cy-adobe-veo-3.1*` / `cy-adobe-kling-3.0*` / `cy-adobe-gemini-omni-flash` | Adobe2API | 渠道 86 统一使用 Adobe2API 视频出站；渠道 75 仅保留图片池；严格 JSON → `/v1/videos/generations` | `video.generation` → OpenAI Video 形 |
| 其他（Sora 等） | default | 标准 OpenAI Video | OpenAI Video 形 |

门面适配器：[`relay/channel/task/oaivideo/router/adaptor.go`](../relay/channel/task/oaivideo/router/adaptor.go)

## 任务初始状态

视频任务提交成功后不再长期停留在 `NOT_START`：

- 默认 `SUBMITTED` / `10%`
- 若上游响应 `status=queued` → `QUEUED` / `20%`
- 实现：[`model.ApplySubmittedStatusFromUpstreamData`](../model/task.go)，在 [`controller/relay.go`](../controller/relay.go) 插入任务前调用

提交接口只负责取得并返回任务 ID，不等待生成完成；`queued`、`in_progress` 以及 Leonardo 插件内部的 `delayed` 都是非终态。

## 新增视频模型 Checklist

1. 在 `registry.ResolveSubmission` 注册匹配规则；渠道专属协议必须排在通用模型族规则之前
2. 确认提交阶段：走 Grok / Seedance / Adobe 转换，还是 default 透传
3. 确认轮询：`FetchTask` 是否仍为 `/v1/videos/{id}`；若路径不同，由提交时持久化的 vendor 选择路径
4. 确认计费：`AdjustBillingOnComplete` 按秒还是按次
5. 补充 `registry` / `router` 单测
6. 源站抽一条任务验收状态推进与 quota

Adobe 额外检查：确认严格 JSON 只发送上游允许字段；确认提交时选中的多 Key 已写入私有任务数据，并由通用轮询复用同一 Key。

## 相关文件

- `service/clienterror/` — 客户端错误翻译（单入口 `NormalizeClientErrorMessage`）
- `docs/client-error-normalization.md` — 错误归一化架构与扩展方式
- `relay/channel/task/oaivideo/registry/` — 路由注册表
- `relay/channel/task/oaivideo/shared/` — 共享解析与 `FetchVideoTask`
- `relay/channel/task/oaivideo/vendors/` — 各上游子适配器
- `relay/channel/task/README.md` — task 目录总览（L1/L2 分层）
- `service/task_polling.go` — 轮询主循环

图像与视频跨能力的完整审计见 [`media-request-chain-audit.md`](media-request-chain-audit.md)。

## 参考素材体积契约

参考图/视频/音频的单文件上限以 [`common/reference_media_limits.go`](../common/reference_media_limits.go) 为 canonical 常量：

| 字段 | 默认值 |
|------|--------|
| `ReferenceImageMaxBytes` | 30MB |
| `ReferenceVideoMaxBytes` | 50MB |
| `ReferenceAudioMaxBytes` | 15MB |

同步要求：

1. `scripts/seed_data/model_ui_params_video.json` 的 `referenceLimits.*MaxBytes` 必须与上述常量一致（`common/reference_media_limits_test.go` 会校验）。
2. Adobe2API 部署的 `adobe2api/core/media_limits.py` 默认值须一致，可通过 `REFERENCE_*_MAX_BYTES` 环境变量覆盖。
3. 面向用户的中文报错统一由 `service.NormalizeClientErrorMessage` 生成；画布只做解析，提交前校验复用同一文案模板（`参考图超过 30MB，请压缩后再上传`）。
