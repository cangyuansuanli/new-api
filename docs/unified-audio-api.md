# 统一音乐 / 音频生成 API 参考

面向外部 API 客户与无限画布共用的音频入站契约。本文定义唯一的统一端点、canonical 字段和任务生命周期；每个模型自己的 `models.api_doc` 只能从这套字段中声明支持子集、范围与完整示例。

## 端点

| 方法 | 路径 | 用途 |
|------|------|------|
| `POST` | `/v1/audio/generations` | 创建音频任务 |
| `GET` | `/v1/audio/generations/{task_id}` | 查询异步任务状态与结果 |

鉴权：`Authorization: Bearer sk-xxx`

## Canonical 请求字段

新客户端使用 `application/json`。

| 字段 | 类型 | 说明 |
|------|------|------|
| `model` | string | 必填，模型广场 public 名 |
| `prompt` | string | 必填，音频内容描述 |
| `response_format` | string | 返回格式 |
| `async` | boolean | 是否异步提交 |
| `stream` | boolean | 是否流式；仅模型明确支持时使用 |

上述是统一 schema，不表示每个模型全部支持。具体模型只能使用其独立文档列出的子集和范围，未列字段不要发送。

服务端可为历史客户保留入站归一化，出站 adaptor 再按各渠道协议转换。兼容路径、旧字段和渠道专属参数不属于新公共契约，不展示在客户文档中。

## 任务状态

异步模型提交后返回任务 ID，随后查询对应任务。常见状态为 `queued`、`in_progress`、`completed`、`failed`；完成后从响应中的结果 URL 下载。具体响应结构与建议轮询间隔以该模型文档为准。

## 单模型文档

每个启用模型必须独立提供 `dispatch_mode`、统一端点、完整 canonical 请求示例、支持字段、取值范围和响应示例。模型文档不得出现统一 schema 之外的字段。`audio_ui_params` 只驱动画布控件，不生成、合并或补全 API 文档；模型缺少有效 `api_doc` 时不展示文档。
