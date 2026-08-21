# 模型命名与路由契约

NewAPI 对客户提供可调用公共名；内部渠道能力名不进入公共目录。同一公共名可配置多条 internal 映射（容灾/多渠道路由），**开关决定当前生效路由**。

| 名称层 | 数据来源 | 是否进入模型广场 | 是否进入 `/v1/models` | 是否可调用 |
|--------|----------|------------------|-----------------------|------------|
| 公共路由名 | `model_public_aliases`（开关 ON 的唯一项） | 是 | 是 | 是 |
| 默认回退路由 | `model_routing_aliases` | 是（无 public alias 时） | 是 | 是 |
| internal ability | `abilities.model` | 否 | 否 | 仅兼容调用 |

例如 `gpt-image-2-1k` 可同时映射 `cy-ac-gpt-image-2-1k` 与 `adobe-firefly-gpt-image-2-1k`；管理员在「模型命名与路由」打开其中一个开关即可切换默认渠道，无需改客户端模型名。`model_routing_aliases` 保留给尚无 public alias 的中性名（如 `seedance-2.0`），或在所有候选开关均关闭时作为回退目标。

## `/v1/models`

`/v1/models` 返回当前 Token 和用户组真正可调用且可发现的外部名称，不返回 `cy-*` internal 名。列表包含：

- 当前开关 ON 的公共路由名
- 尚无 public alias 的 routing alias 中性名

并通过 `supported_endpoint_types` 描述能力：

- 文本：`openai`、`openai-response`
- 图像：`image-generation`（渠道可配置为 `openai-image`）
- 视频：`openai-video`
- 音频：`openai-audio`

模型仍需同时满足 enabled ability、enabled channel、用户组和计费配置。

## 兼容规则

- 入站解析顺序：internal 直调 → **public alias 中开关 ON 的唯一项** → 全部关闭时回退 `model_routing_aliases` → 单映射兜底 → 歧义报错。
- 同一 `public_name` 允许多条 `model_public_aliases`；**开关 ON 时自动关闭同公共名的其它项**，禁止同时 ON 两个。
- `model_public_aliases` 与 `model_routing_aliases` 可共享同一 `public_name`；运行时以开关 ON 的 public alias 优先，routing alias 仅作回退。
- 隐藏 public alias 不参与路由选择，也不进入 `/pricing`、模型广场和 `/v1/models`。
- `cy-*` 只用于内部命名空间，不通过自动前缀剥离生成任何外部名称。
