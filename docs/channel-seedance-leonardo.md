# Leonardo 视频模型（cy-sd4）

> 契约：`leonardo-web2api/docs/api.md` · Seedance 2.0/2.5 规则：`leonardo-web2api/docs/models/seedance-2.0.md`、`seedance-2.5.md` · Hailuo：`leonardo-web2api/docs/models/hailuo-03.md` · 任务路由：[`video-task-routing.md`](video-task-routing.md)

## 分层

```text
客户端 → NewAPI(cy-sd4)     路由 / 计费 / multipart 形态 / 错误透传
       → leonardo-web2api   videocatalog + referencemedia + clienterror
画布 profile               UX 提示；与 worker 冲突时以 leonardo-web2api 为准
```

| 层 | 包 / 模块 | 职责 |
| --- | --- | --- |
| **NewAPI** | `seedanceleonardo/adaptor` | flat JSON → `POST /v1/videos`；**仅** `ValidateMultipartDirect`（multipart、字段别名），**不**下载参考 URL、**不**做数量/互斥/像素等业务校验 |
| **NewAPI** | `service/client_error.go`、`clienterror/normalize.go` | `cy-sd4-seedance*` / `cy-sd4-minimax-h3*`：`fail_reason` 与轮询 JSON **原样返回** upstream 已人话化文案（跳过 `leonardo.go`） |
| **leonardo-web2api** | `internal/videocatalog` | 对外 `model` slug、成片能力、`ReferenceMediaPolicyID` |
| **leonardo-web2api** | `internal/referencemedia` | 入队 `ValidateEnqueue`；worker 下载 → 上传 → **CreateVideoGeneration 前** policy 硬校验 |
| **leonardo-web2api** | `internal/clienterror` | HTTP 4xx 与任务 `failed` 文案；队列保留原始错误，`GET /v1/videos/{id}` 结合任务输入模式归一化一次 |

错误归一化遵循“渠道具体、通用兜底”：Leonardo worker / 上游的原始错误保存在其
`queue_jobs.error_message`，Leonardo 视频任务出口根据持久化的 `input_mode` 与素材数量
生成最终客户文案；NewAPI `cy-sd4` 不再改写。NewAPI 的通用错误层只处理未由渠道覆盖的
跨渠道错误，不得覆盖渠道已经生成的 `error.message` / `fail_reason`。

## 路由

vendor 只接受下表完整的 internal/upstream 配对，不使用 `cy-sd4-*` 前缀模糊匹配：

| public 模型名 | internal 模型名 | upstream `model` | 固定档位 |
| --- | --- | --- | --- |
| `sd4-seedance-2.0` | `cy-sd4-seedance-2.0` | `seedance-2.0` | 请求选择 480p/720p |
| `sd4-seedance-2.0-fast` | `cy-sd4-seedance-2.0-fast` | `seedance-2.0-fast` | 请求选择 480p/720p |
| `sd4-seedance-2.0-mini` | `cy-sd4-seedance-2.0-mini` | `seedance-2.0-mini` | 请求选择 480p/720p |
| `seedance-2.0-mini-8s` | `cy-sd4-seedance-2.0-mini-8s` | `seedance-2.0-mini` | 最长 8s |
| `sd4-seedance-2.5-480p` | `cy-sd4-seedance-2.5-480p` | `seedance-2.5` | 强制 480p |
| `sd4-seedance-2.5-720p` | `cy-sd4-seedance-2.5-720p` | `seedance-2.5` | 强制 720p |
| `minimax-h3-768p` | `cy-sd4-minimax-h3-768p` | `hailuo-03` | 强制 768p |
| `minimax-h3-2k` | `cy-sd4-minimax-h3-2k` | `hailuo-03` | 强制 2K |
| `minimax-h3-4k` | `cy-sd4-minimax-h3-4k` | `hailuo-03` | 强制 4K |
| `happyhouse-1.0` | `cy-sd4-happyhouse-1.0` | `happy-horse` | 请求选择 HD/FHD |
| `happyhouse-1.1` | `cy-sd4-happyhouse-1.1` | `happy-horse-1.1` | 请求选择 HD/FHD |

固定档位 SKU 会覆盖客户端传入的 `resolution`，避免低价 SKU 请求高成本清晰度。

实现：[`relay/channel/task/oaivideo/vendors/seedanceleonardo/`](../relay/channel/task/oaivideo/vendors/seedanceleonardo/)

## UI Profile

[`scripts/seed_data/model_ui_params_video.json`](../scripts/seed_data/model_ui_params_video.json)：

- Seedance 2.0：`video-tpl-seedance-subscription-async`（别名映射见 `service/client_facing_pricing.go`）
- Seedance 2.5：`video-tpl-seedance-2.5-subscription-async`（exact match；最多 30 图 / 10 视频 / 10 音频，视频和音频单条及合计时长上限 30.2 秒）
- MiniMax3 三档：`video-tpl-minimax-h3-2k-async`（历史 profile ID，exact match 三个 SKU；真实网页没有声音开关，因此 Profile 隐藏 `generateAudio`，客户请求不发送该字段）

`referenceLimits` 与 leonardo-web2api 模型文档常量表对齐，仅作表单提示。
MiniMax3 为 **5 图 / 3 视频 / 3 音频**；视频与音频单条 2–15 秒、各自合计 ≤15 秒。不带参考音频时参考视频可为 0；使用参考音频时，必须同时提供至少 1 条参考视频。
Happy House 的素材数量、格式、尺寸、FPS、时长组合和版本能力差异同样由
leonardo-web2api 在入队与 worker 阶段校验；NewAPI 只负责 flat 请求归一化、
模型映射、异步轮询和错误透传。

## 公开 API 契约

统一入口只有以下三个端点：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/v1/videos` | 创建异步视频任务 |
| `GET` | `/v1/videos/{id}` | 查询任务状态 |
| `GET` | `/v1/videos/{id}/content` | 下载已完成视频 |

公共请求字段为 `model`、`prompt`、`duration`、`aspect_ratio`、`resolution`、
`audio`/`generate_audio`、`reference_image_urls`、`reference_videos`、
`reference_audios`、`first_image_url`、`last_image_url`。`seconds`、`image`、
`image_url` 等仅作兼容别名，会在统一层归一化。首尾帧必须成对提供，渠道会将其
映射为 Leonardo 的首帧/尾帧 guidance；它不是普通参考图数组。

### 能力矩阵

| 对外模型 | 输出档位和时长 | 参考图 | 参考视频 | 参考音频 | 备注 |
| --- | --- | --- | --- | --- | --- |
| `cy-sd4-seedance-2.0` / `-fast` / `-mini` | 480p/720p，4–15s | 支持 | 支持 | 支持 | 支持首尾帧；参考视频不改变 Leonardo 费率 |
| `cy-sd4-seedance-2.5-480p` | 480p，4–30s | 最多 30 | 最多 10，单条/合计 ≤30.2s | 最多 10，单条/合计 ≤30.2s | 支持首尾帧；有参考视频费率变为 258/s |
| `cy-sd4-seedance-2.5-720p` | 720p：无参考视频 4–29s，带参考视频 4–18s | 最多 30 | 最多 10，单条/合计 ≤30.2s | 最多 10，单条/合计 ≤30.2s | 支持首尾帧；有参考视频费率变为 466/s |
| `cy-sd4-minimax-h3-768p` | 768p，5–15s | 最多 5 | 最多 3，单条/合计 2–15s、合计 ≤15s | 最多 3，单条/合计 2–15s、合计 ≤15s | 支持首尾帧；有音频参考时至少 1 条视频参考；客户不传 `generate_audio` |
| `cy-sd4-minimax-h3-2k` | 2K，5–15s | 最多 5 | 同上 | 同上 | 支持首尾帧；客户不传 `generate_audio`；SKU 强制 2K |
| `cy-sd4-minimax-h3-4k` | 4K，5–15s | 最多 5 | 同上 | 同上 | 支持首尾帧；客户不传 `generate_audio`；SKU 强制 4K |
| `cy-sd4-happyhouse-1.1` | HD/FHD，3–15s | 最多 9 | 不支持 | 不支持 | 支持首帧；不支持尾帧 |
| `cy-sd4-happyhouse-1.0` | HD/FHD，3–15s | 无视频时最多 9；有视频时最多 5 | 最多 1，3–10s | 不支持 | 支持首帧；不支持尾帧；有视频时网页隐藏输出档位 |

### 8500 积分账号约束

号池单账号初始额度为 **8500 Leonardo 积分**。以下是按当前观测费率计算的
“单号无其他消耗时”可提交上限，不是把上游能力扩大为新的模型能力：

- Seedance 2.5 720p 无参考视频：`floor(8500 / 292) = 29s`，因此可完整使用 29s。
- Seedance 2.5 720p 有参考视频：费率为 `466/s`，最多 `floor(8500 / 466) = 18s`；
  若账号已有消耗或同时存在其他任务，可用时长更短。超过余额必须换号或返回余额不足，不能继续尝试同一账号。
- Seedance 2.5 480p 有参考视频：`floor(8500 / 258) = 32s`，受模型自身 30s 上限约束，
  但仍需扣除账号已有消耗。
- MiniMax3 参考视频会额外计入参考视频秒数；满 15s 成片 + 15s 参考视频的成本为
  3420 / 4200 / 6060 积分（768p / 2K / 4K）。

Web2API 会在 HTTP 入站和 worker 执行前按上述组合各校验一次：720p 仅在存在
`reference_videos` 时收紧到 18 秒；480p 带参考视频仍可到 30 秒。NewAPI 不重复
这条业务校验，只按同一条件计算预扣倍率并透传 Web2API 的 400 错误。

## 计费规则

2026-08-16 在 Leonardo 登录态中**删除参考视频后重新添加**测得的积分费率如下：

| 模型 | 无参考视频 | 有参考视频 |
| --- | --- | --- |
| Seedance 2.0 | 480p 140.6/s；720p 302.4/s | 不变 |
| Seedance 2.0 Fast | 480p 112.48/s；720p 241.92/s | 不变 |
| Seedance 2.0 Mini | 480p 80/s；720p 160/s | 不变 |
| Seedance 2.5 | 480p 180/s；720p 292/s | 480p 258/s；720p 466/s |
| MiniMax3 | 768p 114/s；2K 140/s；4K 202/s | 费率 ×（成片秒数 + 参考视频总秒数） |
| Happy Horse 1.1 | HD 150/s；Full HD 190/s | 不支持参考视频 |
| Happy Horse 1.0 | HD 140/s；Full HD 280/s | 280 × 参考视频秒数 |

NewAPI 对 Seedance 2.5 按秒结算：无参考视频使用基础 `ModelPrice`，有参考视频
自动附加 480p `258/180` 或 720p `466/292` 的倍率；2.0、MiniMax3 和 Happy Horse
仍按各自渠道的按次/上游积分逻辑处理。`8500 积分 = 3 元`，积分成本为
`积分 × 3 / 8500`，不应把上游积分直接当作人民币售价。

参考视频、音频数量、格式、尺寸、FPS、时长合计和互斥条件由 leonardo-web2api
的模型专属 policy 校验；NewAPI 只负责统一字段归一化、SKU 分辨率锁定、计费倍率、
异步轮询和错误透传。

## 相关代码

| 仓库 | 路径 |
| --- | --- |
| new-api | [`adaptor.go`](../relay/channel/task/oaivideo/vendors/seedanceleonardo/adaptor.go)、[`client_error.go`](../service/client_error.go)、[`leonardo.go`](../service/clienterror/leonardo.go)（**非** cy-sd4） |
| leonardo-web2api | `internal/server/payload_map.go`、`internal/referencemedia/`、`internal/clienterror/` |
