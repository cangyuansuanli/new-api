# 客户端错误归一化

所有面向用户/画布的 API 错误翻译在 **new-api 单点完成**，下游只做解析，不再按渠道自行翻译。

## 入口

| 路径 | 职责 |
|------|------|
| [`service/clienterror/normalize.go`](../service/clienterror/normalize.go) | **唯一翻译入口** + 规则注册顺序 |
| [`service/clienterror/common.go`](../service/clienterror/common.go) | 跨渠道：内容审查、超时、体积/提示词、参考素材 |
| [`service/clienterror/omni.go`](../service/clienterror/omni.go) | oairegbox Omni / Veo（cy-sd1-omni-*）内容审查拒绝 |
| [`service/clienterror/leonardo.go`](../service/clienterror/leonardo.go) | Leonardo 池 / cy-sd4 多模态（含号池 humanize） |
| [`service/clienterror/upstream_humanize.go`](../service/clienterror/upstream_humanize.go) | **跨渠道** HTTP/503/容量不可用（不含 vendor 名） |
| [`service/clienterror/adobe.go`](../service/clienterror/adobe.go) | Adobe2API / cy-sd5 |
| [`service/clienterror/grok.go`](../service/clienterror/grok.go) | Grok / Geeknow Grok 视频 |
| [`service/clienterror/manju.go`](../service/clienterror/manju.go) | Manju Sora2 |
| [`service/clienterror/chatvideo.go`](../service/clienterror/chatvideo.go) | Chat 线路视频 |
| [`service/clienterror/defaultvideo.go`](../service/clienterror/defaultvideo.go) | 标准 OpenAI Video 聚合 |
| [`service/clienterror/coverage.md`](../service/clienterror/coverage.md) | **各渠道覆盖表**（缺哪条规则看这里） |

`service/content_policy_message.go` 为兼容层，re-export 常量与入口。

调用点：

- `controller/relay.go` — 同步 relay 错误
- `controller/relay.go` `respondTaskError` — 视频/任务提交
- `relay/relay_task.go` — 任务查询 `fail_reason`
- `relay/image/fetch.go` — 异步生图 job 错误

## 新增 vendor 错误

1. 到上游源码（adobe2api / leonardo-web2api / vendor adaptor）确认 raw 字符串
2. 在 `service/clienterror/<vendor>.go` 增加 matcher
3. 在 `normalize.go` 的 `init()` 里 `Register(normalize<Vendor>)`（common 始终第一）
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
