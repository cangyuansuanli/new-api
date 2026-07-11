# Relay、计费、异步任务与上游适配边界

本文记录统一 API 请求从前端文档到上游渠道的职责边界。新增渠道或异步能力时，先扩展既有边界，不在控制器中复制一套提交、计费和退款流程。

## 1. 请求链路

```text
web/default
  -> API 文档 / 模型 UI 参数（只描述公开接口与参数）
  -> router（鉴权、公开模型名转换、分发）
  -> controller（HTTP 编排与错误响应）
  -> relay / service（渠道选择、模型映射、计费会话、重试）
  -> channel.TaskAdaptor（请求转换、上游调用、响应解析）
  -> model.Task（异步任务持久化）
  -> service.TaskPolling / vendor worker（终态更新与结算）
```

### 边界规则

- Router 只声明路径和 middleware，不判断厂商模型名。
- Controller 不直接拼接上游 URL，不直接增减用户额度；它只编排 relay/service 并返回 DTO。
- `RelayInfo` 是一次请求的运行时上下文，包含原始模型名、公开模型名、映射后的上游模型名、渠道和 `BillingSession`。
- `TaskAdaptor` 是渠道差异的唯一入口：验证请求、构建请求、发起请求、解析响应、任务轮询和可选的终态计费调整。
- `model.Task` 只保存跨请求所需的事实：公开/内部模型名、渠道、计费快照、标准化请求快照和结果。
- Worker/轮询器通过 CAS 推进状态；只有状态转换成功的一方可以执行退款、差额结算和结果写入。

## 2. 计费生命周期

```text
估价 -> BillingSession.Reserve -> 上游提交
                         |                 |
                         | 失败            | 成功
                         v                 v
                    Refund           Settle(actual)
                                           |
                                  TaskBillingContext 快照
                                           |
                                  异步终态可选差额结算
```

- 预扣只由 `service.PreConsumeBilling` 创建 `BillingSession`。
- 重试复用同一个 session，不得按渠道重复预扣。
- 任务成功提交后调用 `service.LogTaskConsumption` 记录消费日志和渠道/用户统计。
- 任务终态只允许使用 `service.RecalculateTaskQuota` 或 `service.RefundTaskQuota`，禁止在 controller、vendor worker 中直接调用用户额度模型方法。
- 按次计费任务的成功终态不再次生成“零差额消费”；零差额不是新的消费事件。
- 失败是否退款由 `service.ShouldRefundTaskOnFailure` 决定，worker 只提供失败原因和响应证据。

## 3. 异步任务契约

- `TaskID` 是公开 ID；`PrivateData.UpstreamTaskID` 仅保存上游任务 ID。
- `Properties.OriginModelName` 是内部模型名；`Properties.ClientModelName` 是公开模型名。响应出口统一调用 `service.PatchClientFacingModelJSONFromTask`。
- `PrivateData.BillingContext` 保存提交时价格、倍率和按次/按量决策，轮询期间不得重新读取当前价格覆盖历史账单。
- `PrivateData.RequestSnapshot` 保存标准化请求，不保存 multipart 边界等传输细节；任务完成或失败后释放。
- `PrivateData.Key` 只在异步 worker 需要复用提交时的多 Key 选择时保存，且永远不会通过 DTO 返回。
- 终态写入必须使用 `Task.UpdateWithStatus(previousStatus)`，CAS 失败时不得执行任何账务副作用。

## 4. 前端契约来源

- 模型编辑页的 `api_mode`、参数 profile、参考媒体限制由 `model_ui_params` 提供。
- 模型广场和 API 文档统一由 `web/default/src/features/pricing/lib/model-api-doc.ts` 生成公开说明。
- 视频统一公开接口是 `POST /v1/videos`、`GET /v1/videos/{task_id}`，下载使用 `/content`；厂商差异只体现在参数 profile 和后端 adaptor。
- 新增模型接口时同时更新 profile、模型 API 文档和后端 adaptor 测试，不在组件中写厂商前缀判断。

## 5. 审查清单

1. 请求是否经过 `PublicModelName`，内部代码是否只使用 `OriginModelName`？
2. 渠道映射是否只在 `ModelMappedHelper` 完成？
3. 是否复用统一 `BillingSession`，且重试不会重复预扣？
4. 成功、失败和超时是否分别只有一个账务副作用入口？
5. 异步 worker 是否能仅凭 `Task` 重启恢复？
6. 状态更新是否 CAS 保护，是否在 CAS 之后才执行退款或结算？
7. 上游错误是否保留可审计原因，但不把密钥、multipart 原文或上游大响应泄露到公开 `Task.Data`？
8. 前端文档的请求格式是否与 router/controller 的实际 Content-Type 解析一致？
