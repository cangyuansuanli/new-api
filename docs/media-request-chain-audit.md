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
| 异步生图 | 同上，body `async=true` | image task worker 重放快照，再进入 image Helper | 本地 worker CAS | 已统一任务表；legacy chat 仅为内部兼容读取器，不进入客户文档 |
| 标准异步视频 | `POST /v1/videos` | task adaptor → `DoTaskApiRequest` | `service.TaskPollingLoop` → `FetchTask` | 通用轮询路径 |
| Grok generations | `POST /v1/videos` | `oaivideo/vendors/grok` → `/v1/video/generations` | 通用轮询按任务模型重选 Grok vendor | 已纳入统一任务族 |
| Adobe2API 视频 | `POST /v1/videos` | `oaivideo/vendors/adobe` → `/v1/videos/generations` | 通用视频轮询 CAS | 已纳入标准视频任务族 |

Adobe2API 只返回 Adobe 上游的短期 presigned URL，不再自行下载或转存媒体。生图和视频结果都必须由 NewAPI Worker 下载并转存 R2；任何 Adobe 上游 URL（包括历史 `eu-ai.cangyuansuanli.cn/generated/` URL）都不得绕过 R2 直接发布给客户。

因此，图像同步/异步仍然由图像能力决定，视频统一为标准异步 Task；Adobe 仅是上游 vendor 差异，不再拥有独立任务生命周期。

客户可见的图像 API 文档只展示 `POST /v1/images/generations`、`POST /v1/images/edits` 与异步任务轮询。仓库中的 legacy chat image 转换仅用于读取和迁移历史客户端请求，不是模型广场或新客户端的公开调用方式。

## 已发现问题

### 已修复

- 异步图像任务启动恢复使用保留完整 `private_data.request_snapshot` 的查询，重启后可重放请求。
- Adobe 视频现在通过标准 task adaptor 进入现有 channel HTTP 执行栈，复用渠道 proxy、Header/Param override、连接池和 HTTP/2。
- 异步图像和标准视频原先没有统一保存提交时选中的多 Key，重启后可能切换账号；现在所有 `Task` 提交路径都保存私有 Key 快照，worker/轮询优先复用该 Key。
- 图像和标准视频统一调用 `service.TransitionTaskStatus`，且差额结算后的最终额度会持久化回 `tasks.quota`。
- 新提交的异步生图任务统一写入版本化 `RequestSnapshot v1` envelope，以 `kind` 明确区分 generation JSON、edit multipart 与 legacy chat JSON；旧任务只在内存中兼容解码，不再写入旧形态。
- edit multipart Worker 重放会保留文件名与原始 MIME，避免 `image/png` 被退化为 `application/octet-stream`。
- New API 与 Infinite Canvas 使用相同媒体契约 fixture，分别覆盖客户端 payload builder、任务持久化和 Worker 重放。
- Redis 通知是实时唤醒主通道；PostgreSQL 仅作持久真相与每 15 秒补偿扫描，避免 Worker 节点增加时每节点每秒扫表。
- 119337 Grok 不再由 default vendor 错投 `/v1/videos`；提交与轮询均在 Grok vendor 映射 generations 路径，`image_urls`、envelope 状态、`result_url` 和 `fail_reason` 在边界归一化。
- 视频轮询请求携带任务保存的 internal / upstream 模型，Router 不再丢失 vendor 身份后固定走默认 `FetchTask`。
- 视频结果转存会识别顶层对象与 `{code,data}` envelope，在正确层级回写 CDN URL 和 `usage.seconds`，不再把 `data` 对象误改成数组。
- 画布 profile 已收口为单一视频 API mode；上游 endpoint 仅由 NewAPI vendor 决定。

### 仍需重构

1. `/v1/edits` 仍是旧的同步 relay 入口，不会进入 `/v1/images/generations` 的 async 判定；前端和文档如果把它当作统一图像入口，会产生行为差异。
2. 前端模型参数元数据仍声明多种视频 API mode；模型选择后的真实 endpoint 仍由 profile/endpoint type 决定，文案需要与后端路由表持续对齐。
3. 旧的 `controller/task_video.go` 只保留仓库内无调用的兼容转发函数，已删除；视频轮询入口统一保留在 `service/task_polling.go`。

## 重构顺序

1. [已完成] 抽出 `media task` 的统一状态推进器：图像、Adobe、标准视频共用 `service.TransitionTaskStatus`，差额结算后的最终额度写回 `tasks.quota`。
2. [已完成] 抽出版本化 `RequestSnapshot v1`：新任务统一持久化 method、path、content type、kind 与标准化 body/files；legacy chat 只作为明确 kind 和旧任务只读兼容存在。
3. [下一阶段] 将 Adobe vendor 的严格请求体规范化进一步抽出为共享 `TaskRequestBuilder`；生命周期与终态结算已经复用标准视频路径。
4. 最后处理兼容入口：确认客户端迁移后再移除 `/v1/edits` 和 legacy chat 的重复路径。

## 验收条件

- 进程重启后，未完成图像/视频任务可凭任务记录恢复，不依赖 HTTP 原始 body。
- 每个任务只发生一次成功结算或失败退款；并发 worker 只有一个 CAS 获胜者。
- public、internal、upstream 三个模型名在日志、任务查询和上游 body 中边界清晰。
- 渠道 proxy、Header override、Param override、多 Key 和上游模型映射在同步和异步执行中结果一致。
- `POST /v1/images/generations`、`POST /v1/images/edits` 的 JSON/multipart 行为由 New API 与 Infinite Canvas 的同一份契约 fixture 覆盖；视频契约由独立视频链路测试覆盖。
- 新任务只写 `RequestSnapshot v1`，旧任务的三种历史形态只读兼容，Worker 不再依赖 URL 字符串猜测新快照类型。
- edit 文件重放后的文件名、MIME 和字节数与客户端提交一致。
