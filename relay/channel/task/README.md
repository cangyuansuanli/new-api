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
| **1 OpenAI, 55 Sora** | **`oaivideo/router/`** | OpenAI Video 族门面（见 L2） |

## L2：OpenAI Video 族（`oaivideo/`）

共用对外 API（`POST/GET /v1/videos`），同一渠道类型下按 **模型名** 二级路由：

```
oaivideo/
├── router/          # 门面：GetTaskAdaptor 返回 RouterAdaptor
├── registry/        # 模型 → Vendor（manju / seedance / default）
├── shared/          # 协议共享：FetchVideoTask、解析、multipart 透传
└── vendors/
    ├── manju/       # manju-openai-sora*（chat/completions 提交）
    ├── seedance/    # cy-sd1 / cy-sd2 / cy-sd4 / tengd-seedance*
    └── defaultvideo/ # 兜底：sora-2、grok-video 等标准 OpenAI Video
```

路由表与轮询行为详见 [`docs/video-task-routing.md`](../../../docs/video-task-routing.md)。

## 新增模型放哪

| 场景 | 改哪里 |
|------|--------|
| 新上游、新 channel.type | `task/<name>/` + `GetTaskAdaptor` 新 `case` |
| OpenAI Video 族新厂商 | `oaivideo/vendors/<vendor>/` + `registry.Resolve` |
| 仅改解析/计费 | 对应 `vendors/*` 或 `shared/` |

## 共享工具

- `taskcommon/` — 计费基类等，各 task 适配器复用
