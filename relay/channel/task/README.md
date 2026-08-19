# Task 适配器目录

异步任务（视频、音乐等）的 provider 适配器。分 **两层** 阅读，不要按「一个文件夹 = 一个模型」理解。

## L1：渠道类型 → `GetTaskAdaptor`

[`relay/relay_adaptor.go`](../../relay_adaptor.go) 按 `channel.type`（`TaskPlatform`）选一个适配器：

| channel.type | 文件夹 | 说明 |
|--------------|--------|------|
| 35 MiniMax | `hailuo/` | 海螺视频 API |
| 51 Jimeng | `jimeng/` | 火山即梦 CVSync |
| 48 Kling | `kling/` | 可灵 |
| … | `ali/`, `doubao/`, `gemini/`, `suno/`, `vidu/`, `vertex/` | 各独占一类 |
| **1 OpenAI, 55 Sora, 8 Custom** | **`oaivideo/router/`** | OpenAI Video 族门面（见 L2） |

## L2：OpenAI Video 族（`oaivideo/`）

共用对外 API（`POST/GET /v1/videos`）。渠道分发与模型映射完成后，registry 一次性确定出站 vendor 并持久化到任务；后续生命周期不再按模型名重新猜协议：

提交阶段的固定顺序是：模型映射 → vendor 解析/持久化 → 单次初始化 → vendor 校验 → 计费与出站。默认适配器不缓存 Base URL/API Key，所有继承方均从当前 `RelayInfo` 读取连接信息；不得恢复“映射前初始化默认 delegate、映射后切换 provider delegate”的两阶段行为。

```
oaivideo/
├── router/          # 门面：GetTaskAdaptor 返回 RouterAdaptor
├── registry/        # 已分发渠道 + internal/upstream 模型 → Vendor
├── shared/          # 协议共享：FetchVideoTask、解析、multipart 透传
└── vendors/
	    ├── chatvideo/   # 聚合视频线路：统一任务请求 → chat/completions
    ├── grok/        # cy-gv1 + 119337：/v1/video/generations 提交、轮询与响应归一化
    ├── geeknowgrok/ # Geeknow Grok：/v1/videos JSON（grok-imagine-video 系列）
    ├── seqnode/     # Seqnode Grok：/v1/videos/generations + 鉴权 content 成片
    ├── seedanceoairegbox/ # cy-sd1 → OAIREGBox flat /v1/videos
    ├── seedancetengda/    # cy-sd2 / tengd → Tengda content[] JSON
    ├── seedanceleonardo/  # cy-sd4 → Leonardo flat /v1/videos
    ├── seedanceheygen/    # cy-sd6 → 固定分辨率双产品 + 鉴权 content
    ├── seedancehuabu/   # cy-sd8 → Huabu flat /v1/videos；卡脸/快速双 SKU
    ├── omnii2v/     # cy-sd1 omni-fast*：flat reference_image_urls → upstream images[]；双 URL 时优先 vid-* 转存
    ├── omniv2v/     # cy-sd1 omni-fast-v2v*：flat reference_videos → 上游 videos/images
    ├── adobe/       # Adobe2API typed video（含 cy-sd5 Seedance）：/v1/videos/generations
    └── defaultvideo/ # 兜底：sora-2 等标准 OpenAI Video
```

Adobe2API 视频属于 `oaivideo` 的标准任务族：对外使用 `/v1/videos`，vendor 内部提交到 `/v1/videos/generations`，轮询复用 `/v1/videos/{id}`，不再使用独立 worker 或 chat 包装。Veo 标准版支持 3 张普通参考图或成对首尾帧；Veo Fast / Kling 3.0 / Kling 3.0 Omni 仅支持成对首尾帧；Gemini Omni 保留单首帧与风格参考能力。`cy-sd5` 也由 Adobe vendor 按内部模型契约处理：普通参考素材明确输出 `reference_mode=media`，支持 9 图 + 3 个视频/音频共享源位且全部素材最多 12；仅成对首尾帧输出 `reference_mode=frame`，两种模式互斥。

对外 canonical 时长字段是 `duration`；历史 `seconds` 仅在 `relay/common` 入站兼容并归一化。业务层统一读取 `TaskSubmitReq.Duration`，上游字段由 vendor 选择：default / Grok 输出 `seconds`，Seedance / Adobe 输出 `duration`。禁止绕过 vendor 边界透传别名。

JSON vendor 只能从 `TaskSubmitReq.CanonicalVideoBody` 或结构化字段读取业务参数，不得重新解析客户原始 body。公共字段集合由统一契约定义，模型支持集合由 profile / vendor contract 定义，两者取交集；`models.api_doc` 只能补充能力说明，不能扩张请求 schema。统一视频接口不提供任意 vendor 参数兜底。

画布和外部客户只使用 `POST /v1/videos` + `GET /v1/videos/{id}`。`chatvideo` 可以在 vendor 内部调用上游 `chat/completions`，但该路径、SSE 解析和视频 URL 提取不得再下放到前端。

路由表与轮询行为详见 [`docs/video-task-routing.md`](../../../docs/video-task-routing.md)。Leonardo cy-sd4 出站见 [`docs/channel-seedance-leonardo.md`](../../../docs/channel-seedance-leonardo.md)；worker 模型文档见 [`leonardo-web2api/docs/models/README.md`](../../../../leonardo-web2api/docs/models/README.md)。

`seedanceoairegbox` / `seedancetengda` / `seedanceleonardo` 适配器把上游 `queued` / `in_progress`（包括 Leonardo 插件内部的 `delayed`）统一保留为非终态；只有上游明确 `failed` 才结算失败。提交接口应立即返回任务 ID，生成耗时不占用提交请求。

Seedance 2.0 支持纯 prompt 文生与多种参考素材。`seedance-heygen` 只精确匹配 `cy-sd6` 的 720p/1080p 双产品，出站强制产品分辨率、仅发送上游白名单字段，并通过带渠道 Bearer 的 `/content` 来源交给通用 R2 转存。该线路的音频不能单独提交，首尾帧与多模态参考互斥，组合规则由上游做最终校验。

## 新增模型放哪

| 场景 | 改哪里 |
|------|--------|
| 新上游、新 channel.type | `task/<name>/` + `GetTaskAdaptor` 新 `case` |
| OpenAI Video 族新厂商 | `oaivideo/vendors/<vendor>/` + `registry.ResolveSubmission` |
| 仅改解析/计费 | 对应 `vendors/*` 或 `shared/` |

## 共享工具

- `taskcommon/` — 计费基类等，各 task 适配器复用
- `oaivideo/vendors/adobe/` — Adobe2API 请求规范化和 typed endpoint 路由；生命周期与计费复用标准视频
- `oaivideo/vendors/grok/` — 119337 Grok generations 路由；将公共 `reference_image_urls` 映射到上游 `image_urls`，并归一化 envelope 响应
- `oaivideo/vendors/geeknowgrok/` — Geeknow Grok 路由；`POST/GET /v1/videos`，`seconds` 字符串化，`image`/`images` 参考图
- `oaivideo/vendors/seqnode/` — Seqnode Grok 出站；提交、轮询和受保护成片来源均封装在 vendor 内
- `oaivideo/vendors/seedanceheygen/` — cy-sd6 双 SKU 出站；固定分辨率、素材白名单和受保护成片来源均封装在 vendor 内
- `oaivideo/vendors/seedancehuabu/` — cy-sd8 双 SKU（卡脸 9/3/3、快速 9 图）；单图→上游 `image`，多图→上游 `images[]`；`reference_videos`/`reference_audios`→`videos`/`audios`
- `oaivideo/shared/` 的可选字段转换必须保持 `nil → 空值`；轮询路由读取历史任务时必须允许 `ChannelMeta` 缺失。
