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
    ├── seedance/
    ├── adobe/        # Adobe typed video endpoint + strict JSON normalization
    └── defaultvideo/ # sora-2、grok 等兜底
```

## 统一轮询流水线

所有视频模型共用 [`service/task_polling.go`](../service/task_polling.go) 中的 `TaskPollingLoop`（每 15 秒）：

1. `FetchTask` — `GET {baseUrl}/v1/videos/{upstream_task_id}`（OpenAI Video 族协议相同）
2. `ParseTaskResult` / `ParseTaskResultForTask` — 将上游 JSON 映射为内部状态
3. 写 DB（CAS `UpdateWithStatus`）
4. `AdjustBillingOnComplete` — 按 Vendor 结算差额

轮询循环与单任务处理均带 `recover`，避免一次 panic 永久停摆。

Leonardo `cy-sd4-*` 渠道在插件主轮询窗口结束后仍返回 `in_progress`：插件内部转为 `delayed` 并低频追踪原 `generation_id`，NewAPI 不得将其提前改为失败或重新提交。NewAPI 全局任务清理由 `TASK_TIMEOUT_MINUTES` 控制（默认 1440 分钟），应显著长于插件主轮询窗口。

路由恢复必须兼容历史任务缺少 `ChannelMeta`、`UpstreamModelName` 或计费快照的情况：缺失字段按空值处理并回退到 `OriginModelName` / 默认视频解析器，禁止通过嵌入的空指针字段直接取值。共享请求转换同样将 `nil` 视为空值，不能把它序列化为 `"<nil>"` 后误判为客户显式参数。

Adobe2API 视频现在属于标准视频任务族：对外使用 `POST /v1/videos` + `GET /v1/videos/{id}`，Adobe vendor 内部将创建请求映射为上游 `POST /v1/videos/generations`，并使用上游 `GET /v1/videos/{id}` 轮询。Adobe 任务直接进入通用任务表和通用轮询，不再创建独立 worker，也不再包装成 chat。

模型广场与 API 文档中的 Adobe 视频元数据必须同步使用 `openai-video` endpoint、`videos-json-async` UI profile 和 `dispatch_mode=async`；旧 `*-chat` profile 与 `/v1/chat/completions` 示例属于迁移前遗留数据，不能继续对客户展示。仅修正文档/profile 而不调整售价时，运行 `python3 scripts/seed_adobe2api_video_api_doc.py --docs-only`。

## 模型 → Vendor 路由表

注册逻辑：[`relay/channel/task/oaivideo/registry/registry.go`](../relay/channel/task/oaivideo/registry/registry.go)

| internal 模型前缀 | Vendor | 提交差异 | 轮询解析 |
|-------------------|--------|----------|----------|
| `manju-openai-sora*` | Manju | chat/completions 转换 | Manju 响应形（`platform:sora2` 等） |
| `cy-sd1-seedance*` | Seedance | multipart 透传 `/v1/videos` | OpenAI Video 形 |
| `cy-sd4-seedance*` | Seedance | Leonardo 渠道 | OpenAI Video 形 |
| `cy-sd2-seedance*` / `tengd-seedance*` | Seedance | Tengda body 转换 | OpenAI Video 形 |
| `adobe-*sora*` / `adobe-*veo*` | Adobe | 严格 JSON → `/v1/videos/generations` | `video.generation` → OpenAI Video 形 |
| 其他（Grok、Sora 等） | default | 标准 OpenAI Video | OpenAI Video 形 |

门面适配器：[`relay/channel/task/oaivideo/router/adaptor.go`](../relay/channel/task/oaivideo/router/adaptor.go)

## 任务初始状态

视频任务提交成功后不再长期停留在 `NOT_START`：

- 默认 `SUBMITTED` / `10%`
- 若上游响应 `status=queued` → `QUEUED` / `20%`
- 实现：[`model.ApplySubmittedStatusFromUpstreamData`](../model/task.go)，在 [`controller/relay.go`](../controller/relay.go) 插入任务前调用

提交接口只负责取得并返回任务 ID，不等待生成完成；`queued`、`in_progress` 以及 Leonardo 插件内部的 `delayed` 都是非终态。

## 新增视频模型 Checklist

1. 在 `registry.ResolveWithChannel` 注册匹配规则（按 Adobe channel/model 进入独立 vendor adaptor，不复制任务生命周期）
2. 确认提交阶段：走 Manju 转换 / Seedance 透传或 Tengda 转换 / default 透传
3. 确认轮询：`FetchTask` 是否仍为 `/v1/videos/{id}`；解析属于 Manju 族还是 OpenAI Video 形
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
