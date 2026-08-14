---
name: new-channel-onboarding
description: >-
  new-api 上游渠道完整入库流程：源站调研、Apifox/协议对齐、三层模型命名、DB 迁移、
  relay 适配（生图/视频/音频）、UI profile、api_doc/定价、源站执行与验收。
  用户提到新渠道入库、上游渠道适配、oaivideo vendor、model_mapping、
  migrate_*_ssh、seed_*_api_doc、video profile、sd*-seedance 时使用。
---

# 上游渠道入库（new-api）

将任意上游接入平台：**调研 → 模型身份 → DB → relay 适配 → profile → api_doc → 源站执行 → 验收 → 提交**。

真值参考：`AGENTS.md` Rule 4c（生图）、[reference.md](reference.md) 模板。  
UI profile 真值：`scripts/seed_data/model_ui_params_{video,image}.json`。

---

## 核心原则（强制）

1. **路由 = internal → upstream 精确配对**，来自 `channels.model_mapping`；vendor `IsRelay(origin, upstream)` 只认完整模型对。
   - **禁止**硬编码渠道 ID、base URL、vendor 优先级插队、`HasPrefix` 模糊匹配。
2. **三层命名分离**：
   - **internal**（`cy-<prefix>-*`）→ 客户端/API 路由、`channels.models`
   - **public**（`sd8-seedance-2.0`）→ `model_public_aliases`、api_doc、模型广场
   - **upstream**（`sd2.0-933`）→ `model_mapping` value、adaptor 出站 body
3. **出站字段以实测上游为准**，不得假设 OpenAI 标准字段名（例：上游 9 张图都用 `image`，不能发 `images`）。
4. **profile 管客户端参数契约**；vendor 管 upstream 转换；可复用 profile，**不能**复用路由或出站映射。
5. **api_doc 不暴露上游**：写 public 名、统一端点、能力说明；curl 示例由 `model-api-doc.ts` + unified API 生成。
6. **模型身份隔离**：共同词干（如 `seedance-2.0`）不代表同一产品；每个 internal 模型逐条核对渠道、mapping、价格、profile。

---

## 进度清单

复制并逐项勾选：

```
- [ ] 1. 源站调研（渠道 ID、group、mapping、上游实测）
- [ ] 2. 读上游文档/Apifox，确定官方路由与请求体字段
- [ ] 3. 路由分类（生图/视频/音频 + api_mode；逐模型确认）
- [ ] 4. 定义三层命名 + 编写 migrate_<vendor>_ssh.sql
- [ ] 5. relay 适配代码 + 单测（match 精确配对 + payload 字段映射）
- [ ] 6. UI profile（model_ui_params_*.json，match_mode: exact）
- [ ] 7. seed_<vendor>_api_doc.py + sync_enabled_video_api_docs.py
- [ ] 8. 源站执行 SQL + seed（勿重启；等渠道缓存同步 ≤60s）
- [ ] 9. 端到端验收（提交 + 轮询 + 取片/计费 + api_doc 无 upstream 泄露）
- [ ] 10. git-close-loop 提交合并
```

---

## 1. 源站调研（SSH contabo / cy-origin）

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
  WHERE model_name LIKE 'cy-<prefix>%' AND deleted_at IS NULL;
\""
```

**必查项：**

| 字段 | 常见坑 |
|------|--------|
| `channels.group` | 路由缓存按 **group + models** 匹配；视频须含 `VIDEO`，生图含 `IMAGE` |
| `channels.models` | 逗号分隔 **internal** 名，须与 `model_mapping` key 一致 |
| `model_mapping` | internal → upstream；客户端只传 internal |
| `abilities.group` | 须覆盖目标用户 token 的 group |

**上游实测**（用渠道 key，勿写入 skill/日志）：

记录：创建 method+path、轮询 path、成片 URL 字段（`result_url`/`video_url`/`/content`）、参考素材字段名与上限、时长枚举、错误码。

---

## 2. 读上游文档

**不要假设** OpenAI 标准路由就是上游官方路由。例：Manju sora2 官方创建走 `POST /v1/chat/completions`，不是 `/v1/videos`。

---

## 3. 路由分类

| 模态 | 客户端 api_mode | 典型 profile | 代码落点 |
|------|-----------------|--------------|----------|
| 生图 sync/async | `images-*` | `image-tpl-*` | `relay/imagevendor/` + `relay/channel/openai/adapt_*.go` |
| 视频 form 异步 | `videos-form` | `video-tpl-form-*` | `oaivideo/vendors/defaultvideo/` |
| 视频 chat 异步 | `chat-completions` | `video-tpl-chat-*` | `relay/channel/openai/adapt_*_chat.go` |
| 视频 json 异步 | `videos-json-async` | `video-tpl-async-*` | `relay/channel/task/oaivideo/vendors/<name>/` |
| 视频 generations | `video-generations` | `video-tpl-gen-*` | 专用 vendor 或 defaultvideo |
| 音频 async | `audio-generations` | `audio-tpl-*` | `relay/channel/task/*` |

**命名约定：**

- internal：`cy-<prefix>-<slug>`，在 `model_channel_prefixes` 注册
- public：`sd8-seedance-2.0` 等中性名，在 `model_public_aliases` 注册
- vendor 目录/文件名用**真实上游名**（如 `seedancehuabu/`），便于维护

---

## 4. DB 迁移

按 [reference.md](reference.md) 新建 `scripts/migrate_<vendor>_ssh.sql`。

**标准块：**

1. `model_channel_prefixes` — prefix + note
2. `model_public_aliases` — internal → public
3. `channels` — models、model_mapping、group、status
4. `abilities` — DELETE 旧项 + INSERT 正确 group
5. `models` — description、tags、endpoints、video_profile_id / image_profile_id

迁移不得用宽泛 `LIKE '%stem%'` 或全局 DELETE 影响同词干旧渠道。

```bash
ssh contabo 'docker exec -i newapi-postgres psql -U root -d new-api < scripts/migrate_<vendor>_ssh.sql'
```

---

## 5. Relay 代码适配

### 5a. 视频 json 异步（oaivideo vendor）

目录：`relay/channel/task/oaivideo/vendors/<upstream-name>/`

```
vendors/<name>/
├── match.go       # IsRelay(origin, upstream) — 精确配对
├── payload.go     # buildUpstreamBody — 出站字段映射
├── result_url.go  # 非标准 URL 字段提取（可选）
├── adaptor.go     # embed defaultvideo，override 生命周期
└── *_test.go
```

**match.go 模板：**

```go
func IsRelay(originModel, upstreamModel string) bool {
    origin := strings.ToLower(strings.TrimSpace(originModel))
    upstream := strings.ToLower(strings.TrimSpace(upstreamModel))
    switch origin {
    case "cy-sd8-seedance-2.0":
        return upstream == "sd2.0-933"
    case "cy-sd8-seedance-2.0-fast":
        return upstream == "sd-2.0-fast-v1"
    default:
        return false
    }
}
```

**payload.go**：从 `req.CanonicalVideoBody(upstreamModel)` 读归一化输入，映射到上游实测字段。  
常见 override：`ValidateRequestAndSetAction`、`BuildRequestBody`、`ParseTaskResult`、`ConvertToOpenAIVideo`、`ResolveTaskResultSource`。

**registry 注册**（与其他 vendor 同级，不插队）：

```go
if seedancehuabu.IsRelay(originModel, upstreamModel) {
    return VendorSeedanceHuabu
}
```

同步：`Vendor*` 常量、`ParseVendor`、`router/adaptor.go` delegate、`registry_test.go` 路由矩阵、`TestVendorDirectoriesAreRegistered`。

### 5b. 生图（Manju Banana 模式）

1. `relay/imagevendor/vendor_<上游名>.go` — Match + PatchRelayRequest
2. `relay/channel/openai/adapt_<name>.go` — 请求体/响应/poll
3. `adaptor.go` 分支 + `adapt_*_test.go`

### 5c. 视频 chat 创建（Manju Sora2 模式）

1. 确认上游**创建**路由（chat vs videos）
2. `relay/channel/openai/adapt_<name>_chat.go`
3. `ParseTaskResult` / `DoResponse` — 非标准 JSON 用 gjson

### 5d. 计费

- 按秒：`ModelPrice` × `OtherRatios.seconds`；`EstimateBilling` / `AdjustBillingOnComplete`
- 按次：`constant.TaskPricePatches` 或 `BillingModePerRequest`

---

## 6. UI profile

文件：`scripts/seed_data/model_ui_params_video.json`（或 `_image.json`）

```json
{
  "id": "video-tpl-sd8-seedance-async",
  "apiMode": "videos-json-async",
  "match_mode": "exact",
  "match": ["cy-sd8-seedance-2.0", "cy-sd8-seedance-2.0-fast"],
  "referenceLimits": { "images": 9, "videos": 3, "audios": 3 },
  "params": { "duration": { "numericOptions": [5, 10, 15] } }
}
```

- 新 profile 用 `match_mode: "exact"` + 完整 internal 名列表
- 全量 seed 前确认 JSON 包含所有既有 profile
- `models.video_profile_id` 与 migrate SQL 一致

---

## 7. Seed 脚本（api_doc + 定价）

模板见 [reference.md](reference.md)。

```bash
scp scripts/seed_<vendor>_api_doc.py contabo:/tmp/
ssh contabo "cd /path/to/new-api/scripts && python3 /tmp/seed_<vendor>_api_doc.py"
```

职责：

- 写入 `models.api_doc`（用 `seed_media_api_doc_common.capability_doc`）
- 更新 `options.ModelPrice`、`billing_setting.*`
- 同步 `scripts/sync_enabled_video_api_docs.py` 的 `Spec` 行

**api_doc 规范：**

- intro 用 **public 名**，不提 upstream 域名/品牌/真实字段名
- 参数说明写**客户端字段**（`reference_image_urls`），不写 upstream 字段（`image`）
- 不写重复 curl/endpoints（由 unified API 生成）

---

## 8. 源站部署与缓存

**生产环境禁止随意 `docker restart`**。DB/seed 变更通常 ≤60s 内通过 `InitChannelCache()` 生效。

Go relay 代码变更走 CI（`.github/workflows/cangyuan-prod.yml`）。

**常见失败：**

| 现象 | 原因 |
|------|------|
| `No available channel under group VIDEO` | `channels.group` 缺 `VIDEO`；或缓存未刷新 |
| `unmarshal_response_body_failed` | 上游 JSON 与 struct 不兼容，需 gjson |
| 任务成功无 URL | 成片字段路径未覆盖（如 `result_url`） |
| 上游 400 字段错误 | payload 映射与实测不一致 |

---

## 9. 端到端验收

```bash
TOKEN=...  # 从 DB 取，勿泄露
curl -X POST "https://<new-api>/v1/videos" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model":"cy-sd8-seedance-2.0","prompt":"...","duration":10,"aspect_ratio":"16:9"}'
```

**隔离回归（必须）：**

- 新 internal 在目标渠道命中，上游 body 字段名/值正确
- 同词干旧模型保持原渠道、原 profile、原出站 payload
- `ResolveSubmission` 对非目标模型对返回其他 vendor
- api_doc 显示 public 名，无 upstream 泄露
- Go 单测：`match`、`payload`、`registry_test`、`router/adaptor_test`

---

## 10. 跨仓同步（可选）

若画布/前端用到该模型，同步 `infinite-canvas/`：

- `docs/dev/model-names.md`
- `docs/dev/newapi-video-model-mapping.json`

**模型广场 public 前缀白名单（必做）：** 若 public 名形如 `sdN-seedance-*`（与 sd5/sd6/sd7 同类），必须在 `web/default/src/features/pricing/lib/model-display-name.ts` 的 `PUBLIC_MODEL_PREFIX_FIRST_SEGMENTS` 加入 `sdN`，否则会被误剥前缀并与其他 `seedance-2.0` 条目合并，导致模型广场与 API 文档不显示。

**源站 api_doc 同步（必做）：** migrate + seed 后，在源站执行 `scripts/sync_enabled_video_api_docs.py`（SPECS 须包含新 internal 名），否则 api_doc 格式不完整。

---

## 11. 提交

遵循 **`.agents/skills/git-close-loop/SKILL.md`**：`feat/new-channel-<name>` → verify → merge main。

提交 body 须含：渠道 ID、internal 模型名、migrate/seed/adapt 文件、源站执行与验收结论。

---

## 参考实例

| 案例 | 模态 | vendor / adapt | migrate | seed |
|------|------|----------------|---------|------|
| Huabu cy-sd8 | 视频 json | `vendors/seedancehuabu/` | `migrate_sd8_seedance_ssh.sql` | `seed_sd8_seedance_api_doc.py` |
| HeyGen cy-sd6 | 视频 json | `vendors/seedanceheygen/` | `migrate_heygen_seedance_ssh.sql` | `seed_heygen_seedance_api_doc.py` |
| Magica cy-sd7 | 视频 json | `vendors/seedancemagica/` | `migrate_magica_seedance_prod.sql` | `seed_magica_seedance_api_doc.py` |
| Manju Banana #70 | 生图 | `adapt_manju_banana.go` | — | `seed_manju_gemini_banana_api_doc.py` |
| Manju Sora2 #70 | 视频 chat | `adapt_manju_sora2_chat.go` | — | — |
| Leonardo cy-sd4 | 视频 json | `vendors/seedanceleonardo/` | `migrate_cy_sd4_seedance_profile.sql` | — |
