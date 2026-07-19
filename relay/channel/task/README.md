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

共用对外 API（`POST/GET /v1/videos`），同一渠道类型下按 **模型名** 二级路由：

```
oaivideo/
├── router/          # 门面：GetTaskAdaptor 返回 RouterAdaptor
    ├── registry/        # 模型/渠道 → Vendor（manju / grok / seedance / adobe / default）
├── shared/          # 协议共享：FetchVideoTask、解析、multipart 透传
└── vendors/
    ├── manju/       # manju-openai-sora*（chat/completions 提交）
    ├── chatvideo/   # 聚合视频线路：统一任务请求 → chat/completions
    ├── grok/        # cy-gv1 + 119337：/v1/video/generations 提交、轮询与响应归一化
    ├── geeknowgrok/ # Geeknow Grok：/v1/videos JSON（grok-imagine-video 系列）
    ├── seedance/    # cy-sd1 / cy-sd2 / cy-sd4 / tengd-seedance*
    ├── sd5/         # cy-sd5 Seedance：typed JSON、seed、9/3/3（合计 12）
    ├── adobe/       # Adobe2API typed video：/v1/videos/generations
    └── defaultvideo/ # 兜底：sora-2 等标准 OpenAI Video
```

Adobe2API 视频属于 `oaivideo` 的标准任务族：对外使用 `/v1/videos`，vendor 内部提交到 `/v1/videos/generations`，轮询复用 `/v1/videos/{id}`，不再使用独立 worker 或 chat 包装。

对外时长参数 `duration` / `seconds` 是同义字段，在 `relay/common` 归一化；上游字段由 vendor 选择：default / Grok 输出 `seconds`，Seedance / Adobe 输出 `duration`。禁止绕过 vendor 边界直接透传两个别名。

画布和外部客户只使用 `POST /v1/videos` + `GET /v1/videos/{id}`。`chatvideo` / `manju` 可以在 vendor 内部调用上游 `chat/completions`，但该路径、SSE 解析和视频 URL 提取不得再下放到前端。

路由表与轮询行为详见 [`docs/video-task-routing.md`](../../../docs/video-task-routing.md)。

`seedance` 适配器把上游 `queued` / `in_progress`（包括 Leonardo 插件内部的 `delayed`）统一保留为非终态；只有上游明确 `failed` 才结算失败。提交接口应立即返回任务 ID，生成耗时不占用提交请求。

Seedance 2.0 支持纯 prompt 文生，也支持参考图、参考视频或参考音频单独提交；vendor 转换不得将参考图作为视频/音频参考的前置条件。

## 新增模型放哪

| 场景 | 改哪里 |
|------|--------|
| 新上游、新 channel.type | `task/<name>/` + `GetTaskAdaptor` 新 `case` |
| OpenAI Video 族新厂商 | `oaivideo/vendors/<vendor>/` + `registry.ResolveWithChannel` |
| 仅改解析/计费 | 对应 `vendors/*` 或 `shared/` |

## 共享工具

- `taskcommon/` — 计费基类等，各 task 适配器复用
- `oaivideo/vendors/adobe/` — Adobe2API 请求规范化和 typed endpoint 路由；生命周期与计费复用标准视频
- `oaivideo/vendors/grok/` — 119337 Grok generations 路由；将公共 `image_urls` 映射到严格上游 JSON，并归一化 envelope 响应
- `oaivideo/vendors/geeknowgrok/` — Geeknow Grok 路由；`POST/GET /v1/videos`，`seconds` 字符串化，`image`/`images` 参考图
- `oaivideo/shared/` 的可选字段转换必须保持 `nil → 空值`；轮询路由读取历史任务时必须允许 `ChannelMeta` 缺失。
