---
name: new-channel-onboarding
description: >-
  new-api 新渠道/新模型入库完整流程：源站调研、Apifox 对齐、DB 迁移、relay 适配、
  api_doc/定价/UI profile、源站执行与验收。用户提到新渠道入库、渠道适配、源站 SSH、
  migrate_*_ssh.sql、seed_*_api_doc、abilities、model_mapping 时使用。
---

# 新渠道入库（new-api）

将上游渠道接入平台：**调研 → 路由分类 → DB → 代码适配 → seed → 源站执行 → 验收 → 提交**。

真值参考：`AGENTS.md` Rule 4c（生图）、`scripts/migrate_*_ssh.sql`、`scripts/seed_*_api_doc.py`。  
详细模板与 SQL 片段见 [reference.md](reference.md)。

---

## 进度清单

复制并逐项勾选：

```
- [ ] 1. 源站调研（渠道 ID、group、mapping、上游实测）
- [ ] 2. 读上游 Apifox/文档，确定官方路由与请求体
- [ ] 3. 路由分类（生图/视频/文本 + api_mode）
- [ ] 4. 编写 migrate_<vendor>_<model>_ssh.sql
- [ ] 5. 编写 seed_<vendor>_<model>_api_doc.py（api_doc + ModelPrice）
- [ ] 6. relay 适配代码 + 单测
- [ ] 7. （可选）UI profile / infinite-canvas 映射
- [ ] 8. 源站执行 SQL + seed，重启 new-api 刷新渠道缓存
- [ ] 9. 端到端验收（提交 + 轮询 + 取片/计费）
- [ ] 10. git-close-loop 提交合并
```

---

## 1. 源站调研（SSH contabo）

默认 SSH 别名：`contabo`（见 `pool-admin/scripts/ssh-config.snippet`）。

```bash
# 渠道配置
ssh contabo "docker exec newapi-postgres psql -U root -d new-api -c \"
  SELECT id, name, type, base_url, \\\"group\\\", models, model_mapping, status
  FROM channels WHERE id = <CHANNEL_ID>;
\""

# abilities / 模型元数据
ssh contabo "docker exec newapi-postgres psql -U root -d new-api -c \"
  SELECT \\\"group\\\", model, channel_id, enabled FROM abilities WHERE channel_id = <CHANNEL_ID>;
  SELECT model_name, video_profile_id, image_profile_id, tags FROM models
  WHERE model_name LIKE '<prefix>%' AND deleted_at IS NULL;
\""
```

**必查项：**

| 字段 | 常见坑 |
|------|--------|
| `channels.group` | 路由缓存按 **group + models** 匹配；视频模型须在 group 含 `VIDEO`，生图含 `IMAGE` |
| `channels.models` | 逗号分隔 internal 名，须与 `model_mapping` key 一致 |
| `model_mapping` | internal → upstream 名；客户端只传 internal |
| `abilities.group` | 须覆盖目标用户 token 的 group（如 `VIDEO`、`全模型-无claude/gpt`） |

**上游实测**（用渠道 key，勿写入 skill/日志）：

```bash
KEY=$(ssh contabo "docker exec newapi-postgres psql -U root -d new-api -t -A -c \"SELECT key FROM channels WHERE id=<ID>;\"" | tr -d ' \n')
# 按 Apifox 文档 curl 上游 base_url，记录：创建路由、响应 JSON 形状、轮询路由、成片 URL 字段
```

---

## 2. 读上游文档（Apifox）

**不要假设** OpenAI 标准路由就是上游官方路由。例：Manju sora2 官方创建走 `POST /v1/chat/completions`，不是 `/v1/videos`。

记录：

- 创建：method + path + 必填字段
- 查询：path + 状态枚举 + 结果 URL 字段路径
- 图生/参考图：额外字段（`input_reference`、`image_url` 等）
- 计费维度：按次 / 按秒 / 按分辨率档位

Manju 文档入口示例：https://ssnsuyettr.apifox.cn/

---

## 3. 路由分类

| 模态 | 客户端 api_mode | 典型 profile | 代码落点 |
|------|-----------------|--------------|----------|
| 生图 sync/async | `images-*` | `image-tpl-*` | `relay/imagevendor/` + `relay/channel/openai/adapt_*.go` |
| 视频 form 异步 | `videos-form` | `video-tpl-form-*` | `relay/channel/task/oaivideo/vendors/defaultvideo/` |
| 视频 chat 异步 | `chat-completions` | `video-tpl-chat-*` | `relay/channel/openai/adapt_*_chat.go` + task 轮询 |
| 视频 json 异步 | `videos-json-async` | `video-tpl-async-*` | `relay/channel/task/*` |
| 视频 generations | `video-generations` | `video-tpl-gen-*` | `relay/channel/task/oaivideo/vendors/defaultvideo/` 或专用 adaptor |

**命名约定：**

- internal 名：`<prefix>-<slug>`（如 `manju-openai-sora2`）
- 在 `model_channel_prefixes` 注册 prefix（如 `manju-`）
- 公开名由 `middleware.PublicModelName` + prefix 规则转换，vendor 代码只匹配 **internal**（`OriginModelName`）

---

## 4. DB 迁移（`scripts/migrate_*_ssh.sql`）

文件命名：`migrate_<vendor>_<feature>_ssh.sql`  
执行：`ssh contabo 'docker exec -i newapi-postgres psql -U root -d new-api < migrate_....sql'`

**标准块（按需提供）：**

1. `model_channel_prefixes` — prefix + note
2. `channels` — `models`、`model_mapping`、**`group`（含 IMAGE/VIDEO）**、`status`
3. `abilities` — 按模态 DELETE 旧项 + INSERT 正确 group
4. `models` — description、tags、vendor_id、`endpoints`、`image_profile_id` / `video_profile_id`

`api_doc` 与 `ModelPrice` **不要**写进 SQL，交给 seed 脚本。

---

## 5. Seed 脚本（`scripts/seed_*_api_doc.py`）

模板：`seed_manju_gemini_banana_api_doc.py` / `seed_manju_sora2_api_doc.py`

职责：

- 写入 `models.api_doc`（endpoints、params、request/response 示例、`dispatch_mode`）
- 更新 `options.ModelPrice`（USD；按次或按秒）
- 同步 `video_profile_id` / `image_profile_id`

源站执行：

```bash
scp scripts/seed_xxx_api_doc.py contabo:/tmp/
ssh contabo "python3 /tmp/seed_xxx_api_doc.py"
```

---

## 6. Relay 代码适配

### 生图（Manju Banana 模式）

1. `relay/imagevendor/vendor_<name>.go` — `Match` + `Rehost`
2. `relay/channel/openai/adapt_<name>.go` — 请求体/响应/poll
3. `adaptor.go` — `ConvertImageRequest` / `GetRequestURL` 分支
4. `adapt_*_test.go` — 请求转换 + 响应解析

### 视频（Manju Sora2 模式）

1. 确认上游**创建**路由（chat vs videos）
2. `relay/channel/task/oaivideo/vendors/<vendor>/` — `/v1/videos` 客户端兼容、上游 body 转换；Manju 嵌套 JSON 见 `vendors/manju/`
3. `relay/channel/openai/adapt_<name>_chat.go` — 客户端直调 chat 时请求/响应
4. `ParseTaskResult` / `DoResponse` — 非标准 `data` 形状用 gjson 兜底
5. `relay/common/relay_utils.go` — 校验 seconds/size（若适用）

### 计费

- 按秒：`ModelPrice` × `OtherRatios.seconds`；`TaskAdaptor.EstimateBilling` / `AdjustBillingOnComplete`
- 按次：`constant.TaskPricePatches` 或 `BillingModePerRequest`

---

## 7. 源站部署与缓存

```bash
# 执行迁移 + seed 后重启 new-api（刷新 InitChannelCache）
ssh contabo "docker restart cangyuan-stack-new-api-b-1"
# 等待 health 后再测
```

**常见失败：**

| 现象 | 原因 |
|------|------|
| `No available channel under group VIDEO` | `channels.group` 缺 `VIDEO` 或未重启 |
| `unmarshal_response_body_failed` | 上游 JSON 与 struct 不兼容，需 gjson 适配 |
| 任务成功无 URL | 成片字段路径未覆盖（如 `raw_data.video_url`） |

---

## 8. 端到端验收

```bash
# 经 docker 网络打 new-api（VIDEO group token）
TOKEN=...  # 从 DB 取，勿泄露
docker run --rm --network cangyuan-network curlimages/curl:8.5.0 \
  -X POST "http://new-api-b:3000/v1/videos" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model":"<internal>","prompt":"...","seconds":"8","size":"1280x720"}'
# 或按 api_mode 走 /v1/chat/completions
```

验收：提交 200 → 轮询 status 终态 → 成片 URL 可访问 → 额度扣费合理。

**Go 代码变更**须 `push main` 触发 CI（`.github/workflows/cangyuan-prod.yml`）后线上才生效；DB/seed 可先于代码上线。

---

## 9. 跨仓同步（可选）

若画布/前端用到该模型，同步 `infinite-canvas/`：

- `docs/dev/model-names.md`
- `docs/dev/newapi-video-model-mapping.json`
- `web/src/lib/video-parameter-profiles.ts` / payload builder

同一 feature 分支名 + commit header，body 写 `配合：infinite-canvas …`。

---

## 10. 提交

遵循 **`.agents/skills/git-close-loop/SKILL.md`**：`feat/new-channel-<name>` → verify → merge main。

提交 body 须含：

- 渠道 ID、internal 模型名、上游文档链接
- 新增/修改的 `migrate_*.sql`、`seed_*.py`、adapt 文件
- 源站是否已执行、验收结论
- `文档：` 路径或 `文档：无`

---

## 参考实例

| 案例 | 模态 | 关键文件 |
|------|------|----------|
| Manju Banana #70 | 生图 async/sync | `migrate_manju_gemini_banana_ssh.sql`, `adapt_manju_banana.go`, `vendor_manju_banana.go` |
| Manju Sora2 #70 | 视频 chat 创建 | `migrate_manju_sora2_ssh.sql`, `adapt_manju_sora2.go`, `adapt_manju_sora2_chat.go` |
| Leonardo Seedance | 视频 async | `migrate_leonardo_seedance_ssh.sql`, `seed_leonardo_seedance_api_doc.py` |

更多 SQL/Python 模板见 [reference.md](reference.md)。
