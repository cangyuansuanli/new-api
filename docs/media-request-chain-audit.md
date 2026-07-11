# 图像与视频请求链路审计

本文记录用户选择模型后，从公开模型名进入 New API、完成渠道选择与模型映射、调用上游并返回同步或异步结果的当前结构。审计范围包括图像同步、图像异步、标准视频异步和 Adobe2API 视频异步。

## 结论

模型名入口边界是成立的：

1. `middleware.PublicModelName` 将客户端 public model 翻译为 internal model，并把客户端名称存入 context。
2. `service.CacheGetRandomSatisfiedChannel` 选择渠道；`helper.ModelMappedHelper` 将 internal model 映射为 upstream model。
3. `model.InitTask` 保存 `OriginModelName`、`UpstreamModelName` 和客户端名称，异步查询时由 service 层恢复客户端名称。

真正没有统一的是“媒体执行器”：

| 能力 | 对外入口 | 提交执行 | 结果推进 | 当前状态 |
| --- | --- | --- | --- | --- |
| 同步生图 | `POST /v1/images/generations`、`/v1/images/edits` | `relay/image.Helper` → channel adaptor | 当前请求内完成 | 主路径 |
| 异步生图 | 同上，body `async=true`；另有 legacy `/v1/chat/completions` | image task worker 重放快照，再进入 image Helper | 本地 worker CAS | 已统一任务表，仍保留 legacy chat 分支 |
| 标准异步视频 | `POST /v1/videos` | task adaptor → `DoTaskApiRequest` | `service.TaskPollingLoop` → `FetchTask` | 通用轮询路径 |
| Adobe2API 视频 | `POST /v1/videos` | `oaivideo/vendors/adobe` → `/v1/videos/generations` | 通用视频轮询 CAS | 已纳入标准视频任务族 |

因此，图像同步/异步仍然由图像能力决定，视频统一为标准异步 Task；Adobe 仅是上游 vendor 差异，不再拥有独立任务生命周期。

## 已发现问题

### 已修复

- 异步图像任务启动恢复使用保留完整 `private_data.request_snapshot` 的查询，重启后可重放请求。
- Adobe 视频现在通过标准 task adaptor 进入现有 channel HTTP 执行栈，复用渠道 proxy、Header/Param override、连接池和 HTTP/2。
- 异步图像和标准视频原先没有统一保存提交时选中的多 Key，重启后可能切换账号；现在所有 `Task` 提交路径都保存私有 Key 快照，worker/轮询优先复用该 Key。
- 图像和标准视频统一调用 `service.TransitionTaskStatus`，且差额结算后的最终额度会持久化回 `tasks.quota`。

### 仍需重构

1. 图像异步 worker 仍有 generation JSON、edit multipart 标准化快照、legacy chat JSON 三种输入形态，后续可继续抽象为统一 `RequestKind` 快照接口。
2. `/v1/edits` 仍是旧的同步 relay 入口，不会进入 `/v1/images/generations` 的 async 判定；前端和文档如果把它当作统一图像入口，会产生行为差异。
3. 前端模型参数元数据仍声明多种视频 API mode；模型选择后的真实 endpoint 仍由 profile/endpoint type 决定，文案需要与后端路由表持续对齐。
4. 旧的 `controller/task_video.go` 只保留仓库内无调用的兼容转发函数，已删除；视频轮询入口统一保留在 `service/task_polling.go`。

## 重构顺序

1. [已完成] 抽出 `media task` 的统一状态推进器：图像、Adobe、标准视频共用 `service.TransitionTaskStatus`，差额结算后的最终额度写回 `tasks.quota`。
2. [下一阶段] 抽出统一 `RequestSnapshot`：请求方法、公开/内部/上游模型、路径、标准化 body/files、渠道 Key 引用和执行策略一起持久化。
3. 将 Adobe vendor 的严格请求体规范化进一步抽出为共享 `TaskRequestBuilder`；生命周期与终态结算已经复用标准视频路径。
4. 最后处理兼容入口：保留 legacy chat image 读取器，但把它转换为同一个 image request kind；确认客户端迁移后再移除 `/v1/edits` 和 legacy chat 的重复路径。

## 验收条件

- 进程重启后，未完成图像/视频任务可凭任务记录恢复，不依赖 HTTP 原始 body。
- 每个任务只发生一次成功结算或失败退款；并发 worker 只有一个 CAS 获胜者。
- public、internal、upstream 三个模型名在日志、任务查询和上游 body 中边界清晰。
- 渠道 proxy、Header override、Param override、多 Key 和上游模型映射在同步和异步执行中结果一致。
- `POST /v1/images/generations`、`POST /v1/images/edits`、`POST /v1/videos` 的 JSON/multipart 行为由同一套请求快照测试覆盖。
