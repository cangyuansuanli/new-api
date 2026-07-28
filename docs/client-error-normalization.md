# 客户端错误归一化

所有上游/渠道错误的客户端翻译在 **new-api 单点完成**，下游只做解析，不再按渠道自行翻译。本地参数校验、认证和服务器内部错误仍使用各自所属 API 的 i18n/错误码机制，不伪装成上游错误。

## 入口

| 路径 | 职责 |
|------|------|
| [`service/clienterror/normalize.go`](../service/clienterror/normalize.go) | **唯一翻译入口** + 规则注册顺序（支持 raw 与结构化错误码） |
| [`service/clienterror/common.go`](../service/clienterror/common.go) | 跨渠道：内容审查、超时、体积/提示词、参考素材 |
| [`service/clienterror/transport.go`](../service/clienterror/transport.go) | 非渠道的 HTTP/传输状态（如重试耗尽后的 429） |
| [`service/clienterror/omni.go`](../service/clienterror/omni.go) | oairegbox Omni / Veo（cy-sd1-omni-*）内容审查拒绝 |
| [`service/clienterror/leonardo.go`](../service/clienterror/leonardo.go) | Leonardo 池 / cy-sd4 多模态（含号池 humanize） |
| [`service/clienterror/upstream_humanize.go`](../service/clienterror/upstream_humanize.go) | **跨渠道** HTTP/503/容量不可用（不含 vendor 名） |
| [`service/clienterror/adobe.go`](../service/clienterror/adobe.go) | Adobe2API / Adobe Direct |
| [`service/clienterror/sd5.go`](../service/clienterror/sd5.go) | SD5 / `cy-sd5-seedance-2.0*`，在渠道文件内解析 payload 的 `error_code` / `error_type` |
| [`service/clienterror/grok.go`](../service/clienterror/grok.go) | Grok / Geeknow Grok 视频 |
| [`service/clienterror/manju.go`](../service/clienterror/manju.go) | Manju Sora2 |
| [`service/clienterror/chatvideo.go`](../service/clienterror/chatvideo.go) | Chat 线路视频 |
| [`service/clienterror/defaultvideo.go`](../service/clienterror/defaultvideo.go) | 标准 OpenAI Video 聚合 |
| [`service/clienterror/coverage.md`](../service/clienterror/coverage.md) | **各渠道覆盖表**（缺哪条规则看这里） |

`service/client_error.go` 是 service 层唯一门面，re-export 常量与入口，并负责从持久化 Task 构造通用 `ErrorContext`。渠道 payload schema 不得在该门面解析。

调用点：

- `controller/relay.go` — 同步 relay 错误
- `controller/relay.go` `respondTaskError` — 视频/任务提交
- `relay/relay_task.go` — TaskDto 与 OpenAI Video 任务查询
- `relay/image/fetch.go` — 异步生图 job 查询
- `controller/image_sync_queue.go` — 同步等待的异步生图失败

## 边界

- vendor adaptor 负责上游协议解析，保留 raw reason / payload，不生成面向用户的多语言文案。
- `service/clienterror` 只在客户端输出边界翻译，不写回 Task，不参与扣费或退款判断。
- `service/task_billing.go` 的错误分类用于计费决策，与客户翻译独立；日志继续保留脱敏后原文。

## 新增 vendor 错误

1. 到上游源码（adobe2api / leonardo-web2api / vendor adaptor）确认 raw 字符串
2. 在 `service/clienterror/<vendor>.go` 增加 matcher；需要错误码/模型上下文时接收 `ErrorContext`，上游 payload schema 也必须在该 vendor 文件内解析
3. 在 `normalize.go` 的 `init()` 里注册：结构化渠道规则用 `Register`，原有 raw 规则用 `RegisterRaw`
4. 更新 `coverage.md` 对应行
5. **不要**在 infinite-canvas 增加翻译逻辑

## 画布解析

[`infinite-canvas/web/src/services/api/relay-error.ts`](../../infinite-canvas/web/src/services/api/relay-error.ts) 只解析 `message` / `detail` / `fail_reason`，**不做翻译**。

Relay 请求携带 `X-Cangyuan-Client: infinite-canvas` 时，new-api 返回已是中文。

## 号池额度类错误

额度相关错误分两类，均引导用户**先榨干剩余额度**（缩短秒数、降分辨率、换 480p/经济档），而非只提示联系管理员：

| 场景 | 上游 raw | 用户文案 |
|------|----------|----------|
| 号池整体耗尽 | `no active cookie`、`depleted (auto-disabled)` | `PoolDepletedMessage*` |
| 本次任务积分不够 | `insufficient credits (need X, have Y)` | `InsufficientCreditsForJobMessage*` |
| 多账号全失败且含积分不足 | `All cookies failed...` | 按类型汇总 + 缩短秒数/经济模型提示 |

**面向用户的文案不得出现 Leonardo、Adobe 等上游/vendor 名称**，也不回显 raw 英文错误；号池类失败按「积分不足 / 并发已满」等类型汇总，不暴露内部账号编号。

常量见 [`service/clienterror/messages.go`](../service/clienterror/messages.go)。

## 参考素材体积契约

见 [`video-task-routing.md`](video-task-routing.md)；常量源：`common/reference_media_limits.go`。
