# Leonardo Seedance（cy-sd4）

> 契约：`leonardo-web2api/docs/api.md` · 参考素材规则：`leonardo-web2api/docs/models/seedance-2.0.md` · 任务路由：[`video-task-routing.md`](video-task-routing.md)

## 分层

```text
客户端 → NewAPI(cy-sd4)     路由 / 计费 / multipart 形态 / 错误透传
       → leonardo-web2api   videocatalog + referencemedia + clienterror
画布 profile               UX 提示；与 worker 冲突时以 leonardo-web2api 为准
```

| 层 | 包 / 模块 | 职责 |
| --- | --- | --- |
| **NewAPI** | `seedanceleonardo/adaptor` | flat JSON → `POST /v1/videos`；**仅** `ValidateMultipartDirect`（multipart、字段别名），**不**下载参考 URL、**不**做数量/互斥/像素等业务校验 |
| **NewAPI** | `service/client_error.go`、`clienterror/normalize.go` | `cy-sd4-seedance*`：`fail_reason` 与轮询 JSON **原样返回** upstream 已人话化文案（跳过 `leonardo.go`） |
| **leonardo-web2api** | `internal/videocatalog` | 对外 `model` slug、成片能力、`ReferenceMediaPolicyID` |
| **leonardo-web2api** | `internal/referencemedia` | 入队 `ValidateEnqueue`；worker 下载 → 上传 → **CreateVideoGeneration 前** policy 硬校验 |
| **leonardo-web2api** | `internal/clienterror` | HTTP 4xx 与任务 `failed` 文案（`service.ClientFacingMessage` 委托） |

## 路由

| internal 模型前缀 | Vendor | 上游 |
| --- | --- | --- |
| `cy-sd4-seedance*` | `seedance-leonardo` | leonardo-web2api `POST/GET /v1/videos` |

实现：[`relay/channel/task/oaivideo/vendors/seedanceleonardo/`](../relay/channel/task/oaivideo/vendors/seedanceleonardo/)

## UI Profile

[`scripts/seed_data/model_ui_params_video.json`](../scripts/seed_data/model_ui_params_video.json) → `video-tpl-seedance-subscription-async`（别名映射见 `service/client_facing_pricing.go`）。`referenceLimits` 与 [`seedance-2.0.md`](../../leonardo-web2api/docs/models/seedance-2.0.md) 常量表对齐，仅作表单提示。

## 相关代码

| 仓库 | 路径 |
| --- | --- |
| new-api | [`adaptor.go`](../relay/channel/task/oaivideo/vendors/seedanceleonardo/adaptor.go)、[`client_error.go`](../service/client_error.go)、[`leonardo.go`](../service/clienterror/leonardo.go)（**非** cy-sd4） |
| leonardo-web2api | `internal/server/payload_map.go`、`internal/referencemedia/`、`internal/clienterror/` |
