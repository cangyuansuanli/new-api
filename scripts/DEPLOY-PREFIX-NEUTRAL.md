# 渠道前缀中性化（历史源站流程）

> 当前系统不再通过 `model_channel_prefixes` 自动剥离公开名。所有匹配渠道 internal 前缀的模型必须写入显式 `model_public_aliases`；现网遗留结果已由 `migrate_materialize_public_names_ssh.sql` 固化。

将 internal 注册名从 `oairegbox-` / `119337-` 等改为 `cy-*` 路由码；**public 名与 upstream `model_mapping` 值不变**，下游继续用 `seedance-2.0-480p`、`grok-video` 等即可。

## 映射表

| 旧前缀 | 新前缀 | public 示例（不变） |
|--------|--------|----------------------|
| `oairegbox-` | `cy-sd1-` | `seedance-2.0-480p` |
| `119337-` | `cy-gv1-` | `grok-video` |
| `tengda-` | `cy-veo1-` | `veo-3-1-fast` |
| `tengd-` | `cy-sd2-` | `Seedance-2.0` |
| `ctlove-` | `cy-sd3-` | `seedance-2.0` 等 |
| `leonardo-` | `cy-sd4-` | 按 alias |
| `yunwu-` | `cy-vid2-` | `sora-2` 等 |
| `happyhorse-` | `cy-vid1-` | — |
| `gulie-` | `cy-img1-` | `gpt-image-2` |
| `geek2-` | `cy-img2-` | `gpt-image-2-4k` |
| `gz-` | `cy-sd0-` | 已下线 |

## 执行顺序

1. **先** merge 并部署 new-api + infinite-canvas（含 `cy-sd2-` / `cy-img1-` 代码判断）
2. **再** 源站执行 SQL：

```bash
docker exec -i newapi-postgres psql -U root -d new-api \
  < new-api/scripts/migrate_prefix_neutral_prod.sql
```

3. **滚动重启** new-api 蓝绿实例（刷新 `model_channel_prefixes` 内存注册表）
4. 看 SQL 末尾验收：`models_still_old_prefix` / `tags_still_vendor` / `description_still_vendor` / `api_doc_still_vendor` 应为 0 行

## 元信息（必须改）

`/api/pricing` 会把 `model_name` 剥成 public 名，但以下字段**原样下发**，模型广场与「查看文档」直接展示：

| 字段 | 处理 |
|------|------|
| `description` | 去掉 OAIREGBox / 119337 / Geeknow 等商家词 |
| `tags` | 删掉 `oairegbox,geeknow,119337…`，只留 `video,seedance,480p` 等能力标签 |
| `api_doc` | JSON 内 `intro` / 示例 `model` 字段：前缀替换 + 文案清洗 |
| `vendor_id` | **不改**（关联 Google/OpenAI/xAI 等官方厂商，不是转售渠道名） |
| `model_ui_param_profiles.hints` | 同步去旧前缀（SQL §6） |

## 不影响下游的原因

- 请求侧 public 名由 `model_channel_prefixes` 剥前缀生成，剥的是新 `cy-sd1-` 等，结果与旧 `oairegbox-` 相同
- `channels.model_mapping` **只改 JSON key（internal）**，value（upstream 模型名）不动
- `upstream_model_name` 日志字段保留

## 兼容性说明

- 已硬编码 **旧 internal 名**（如 `oairegbox-seedance-2.0-720p`）的 API 调用会 404，应改用 public 名
- 历史 `logs.model_name` 不批量改（计费审计保留原样）

## 供应商 ↔ cy 对照（仅运维，勿提交公开文档）

维护在私有笔记；SQL 内 note 字段仅写能力，不写域名。
