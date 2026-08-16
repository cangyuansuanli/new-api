# 模型命名与路由契约

NewAPI 对客户提供两类可调用名称，内部渠道能力名不进入公共目录。

| 名称层 | 数据来源 | 是否进入模型广场 | 是否进入 `/v1/models` | 是否可调用 |
|--------|----------|------------------|-----------------------|------------|
| 稳定 API 路由名 | `model_routing_aliases.public_name` | 是 | 是 | 是 |
| 指定渠道展示名 | `model_public_aliases.public_name` | 由 `hidden_from_marketplace` 控制 | 与模型广场一致 | 是 |
| internal ability | `abilities.model` | 否 | 否 | 仅兼容调用 |

例如 `grok-video-1.5` 通过 `model_routing_aliases` 指向当前默认渠道；客户无需改模型名即可切换目标。`gv2-grok-video-1.5` 是指定 GV2 的渠道展示名，管理员可决定是否在目录中显示。`cy-gv2-grok-video-1.5` 仅用于内部注册、渠道映射和兼容调用。

## `/v1/models`

`/v1/models` 返回当前 Token 和用户组真正可调用且可发现的外部名称，不返回 `cy-*` internal 名。列表同时包含稳定 API 路由名和未隐藏的指定渠道展示名，并通过 `supported_endpoint_types` 描述能力：

- 文本：`openai`、`openai-response`
- 图像：`image-generation`（渠道可配置为 `openai-image`）
- 视频：`openai-video`
- 音频：`openai-audio`

模型仍需同时满足 enabled ability、enabled channel、用户组和计费配置。一个名称可支持多个 endpoint type；无需按模态拆分成多个 `/v1/models` 接口。

## 兼容规则

- 入站优先解析稳定 routing alias，再解析指定渠道 public alias。
- 已注册且具有显式 public alias 的 internal 名继续允许调用，但不对新客户展示。
- 隐藏 public alias 只影响 `/pricing`、模型广场和 `/v1/models`，不删除别名，也不影响已有客户调用。
- `cy-*` 只用于内部命名空间，不通过自动前缀剥离生成任何外部名称。
