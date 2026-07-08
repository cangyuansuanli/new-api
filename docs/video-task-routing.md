# 视频任务路由与轮询

本文描述 OpenAI Video 渠道族（`oaivideo`）的任务生命周期、模型路由与轮询行为。

## 目录结构

```
relay/channel/task/oaivideo/
├── router/           # 门面适配器（ChannelType 1/55 入口）
├── registry/         # 模型 → Vendor 注册表
├── shared/           # FetchVideoTask、响应解析、multipart 透传
└── vendors/
    ├── manju/
    ├── seedance/
    └── defaultvideo/ # sora-2、grok 等兜底
```

## 统一轮询流水线

所有视频模型共用 [`service/task_polling.go`](../service/task_polling.go) 中的 `TaskPollingLoop`（每 15 秒）：

1. `FetchTask` — `GET {baseUrl}/v1/videos/{upstream_task_id}`（OpenAI Video 族协议相同）
2. `ParseTaskResult` / `ParseTaskResultForTask` — 将上游 JSON 映射为内部状态
3. 写 DB（CAS `UpdateWithStatus`）
4. `AdjustBillingOnComplete` — 按 Vendor 结算差额

轮询循环与单任务处理均带 `recover`，避免一次 panic 永久停摆。

## 模型 → Vendor 路由表

注册逻辑：[`relay/channel/task/oaivideo/registry/registry.go`](../relay/channel/task/oaivideo/registry/registry.go)

| internal 模型前缀 | Vendor | 提交差异 | 轮询解析 |
|-------------------|--------|----------|----------|
| `manju-openai-sora*` | Manju | chat/completions 转换 | Manju 响应形（`platform:sora2` 等） |
| `cy-sd1-seedance*` | Seedance | multipart 透传 `/v1/videos` | OpenAI Video 形 |
| `cy-sd4-seedance*` | Seedance | Leonardo 渠道 | OpenAI Video 形 |
| `cy-sd2-seedance*` / `tengd-seedance*` | Seedance | Tengda body 转换 | OpenAI Video 形 |
| 其他（Grok、Sora 等） | default | 标准 OpenAI Video | OpenAI Video 形 |

门面适配器：[`relay/channel/task/oaivideo/router/adaptor.go`](../relay/channel/task/oaivideo/router/adaptor.go)

## 任务初始状态

视频任务提交成功后不再长期停留在 `NOT_START`：

- 默认 `SUBMITTED` / `10%`
- 若上游响应 `status=queued` → `QUEUED` / `20%`
- 实现：[`model.ApplySubmittedStatusFromUpstreamData`](../model/task.go)，在 [`controller/relay.go`](../controller/relay.go) 插入任务前调用

## 新增视频模型 Checklist

1. 在 `registry.Resolve` 注册匹配规则（或扩展 `vendors/seedance`、`vendors/manju` 的 `IsRelay`）
2. 确认提交阶段：走 Manju 转换 / Seedance 透传或 Tengda 转换 / default 透传
3. 确认轮询：`FetchTask` 是否仍为 `/v1/videos/{id}`；解析属于 Manju 族还是 OpenAI Video 形
4. 确认计费：`AdjustBillingOnComplete` 按秒还是按次
5. 补充 `registry` / `router` 单测
6. 源站抽一条任务验收状态推进与 quota

## 相关文件

- `relay/channel/task/oaivideo/registry/` — 路由注册表
- `relay/channel/task/oaivideo/shared/` — 共享解析与 `FetchVideoTask`
- `relay/channel/task/oaivideo/vendors/{manju,seedance,defaultvideo}/` — 子适配器
- `relay/channel/task/README.md` — task 目录总览（L1/L2 分层）
- `service/task_polling.go` — 轮询主循环
