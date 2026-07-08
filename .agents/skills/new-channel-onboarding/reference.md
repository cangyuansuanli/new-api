# 新渠道入库 — 参考模板

## channels.group 与路由

`model.InitChannelCache()` 用 **`channels.group` + `channels.models`** 建索引，**不是** `abilities` 表。

```
用户 token.group = VIDEO
  → 查 group2model2channels["VIDEO"]["manju-openai-sora2"]
  → 需要 channels.group 含 VIDEO 且 models 列表含该 internal 名
```

生图+视频混用渠道示例：

```sql
UPDATE channels SET "group" = 'IMAGE,VIDEO,全模型-无claude/gpt' WHERE id = 70;
```

---

## migrate SQL 骨架

```sql
-- migrate_<vendor>_<feature>_ssh.sql
-- contabo: docker exec -i newapi-postgres psql -U root -d new-api < migrate_....sql

BEGIN;

-- 1. 前缀
INSERT INTO model_channel_prefixes (prefix, note, enabled, sort_order, created_time, updated_time)
VALUES ('<prefix>-', '<Vendor 说明>', TRUE, 125, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (prefix) DO UPDATE SET note = EXCLUDED.note, enabled = TRUE, updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT;

-- 2. 渠道（在管理台已建渠道时，主要补 models / mapping / group）
UPDATE channels SET
    models = '<internal-a>,<internal-b>',
    model_mapping = '{
  "<internal-a>": "<upstream-a>",
  "<internal-b>": "<upstream-b>"
}'::text,
    "group" = 'IMAGE,VIDEO,全模型-无claude/gpt',
    status = 1
WHERE id = <CHANNEL_ID>;

-- 3. abilities
DELETE FROM abilities WHERE channel_id = <CHANNEL_ID> AND model = '<internal-a>';

INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT g.grp, '<internal-a>', <CHANNEL_ID>, true, 0, 90
FROM (VALUES ('VIDEO'), ('全模型-无claude/gpt')) AS g(grp);

-- 4. models
INSERT INTO models (model_name, description, tags, vendor_id, endpoints, status, sync_official, video_profile_id, created_time, updated_time)
SELECT v.model_name, v.description, v.tags, 1, v.endpoints, 1, 0, v.video_profile_id,
    EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('<internal-a>', '描述', 'video,vendor', '{"openai-chat-video":{"path":"/v1/chat/completions","method":"POST"}}', 'video-tpl-chat-no-params')
) AS v(model_name, description, tags, endpoints, video_profile_id)
WHERE NOT EXISTS (SELECT 1 FROM models m WHERE m.model_name = v.model_name AND m.deleted_at IS NULL);

UPDATE models AS m SET
    description = v.description, tags = v.tags, endpoints = v.endpoints,
    video_profile_id = v.video_profile_id, status = 1,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES ...) AS v(...)
WHERE m.model_name = v.model_name AND m.deleted_at IS NULL;

COMMIT;
```

---

## seed Python 骨架

```python
#!/usr/bin/env python3
"""<internal>：api_doc + ModelPrice（源站 docker 内执行）。"""

import json, subprocess, time

MODEL = "<internal-name>"
PROFILE = "video-tpl-chat-no-params"  # 或 image-tpl-*
PRICE = 0.40  # USD/秒 或 USD/次

def psql(sql: str) -> str:
    return subprocess.run(
        ["docker", "exec", "newapi-postgres", "psql", "-U", "root", "-d", "new-api", "-t", "-A", "-c", sql],
        check=True, capture_output=True, text=True,
    ).stdout.strip()

def psql_exec(sql: str) -> None:
    subprocess.run(
        ["docker", "exec", "newapi-postgres", "psql", "-U", "root", "-d", "new-api", "-v", "ON_ERROR_STOP=1", "-c", sql],
        check=True,
    )

def merge_model_price(updates: dict[str, float]) -> None:
    raw = psql("SELECT value::text FROM options WHERE key='ModelPrice'")
    prices = json.loads(raw)
    prices.update(updates)
    payload = json.dumps(prices, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
    psql_exec(f"UPDATE options SET value='{payload}' WHERE key='ModelPrice'")

def main() -> None:
    doc = {"dispatch_mode": "async", "endpoints": [...], "params": [...], ...}
    esc = json.dumps(doc, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
    psql_exec(f"UPDATE models SET api_doc = '{esc}', video_profile_id = '{PROFILE}', updated_time = {int(time.time())} WHERE model_name = '{MODEL}' AND deleted_at IS NULL;")
    merge_model_price({MODEL: PRICE})

if __name__ == "__main__":
    main()
```

---

## video_profile_id 选型

| profile | apiMode | 适用 |
|---------|---------|------|
| `video-tpl-form-size-ref1` | videos-form | OpenAI Sora 风格 seconds+size+ref |
| `video-tpl-chat-no-params` | chat-completions | Sora/Kling 等 chat 取片 |
| `video-tpl-chat-i2v-ref1` | chat-completions | 单图 i2v |
| `video-tpl-seedance-async` | videos-json-async | Seedance JSON body |
| `video-tpl-gen-ratio-ref7` | video-generations | Grok generations |

定义真值：`scripts/seed_data/model_ui_params_video.json`

---

## image_profile_id 选型

定义真值：`scripts/seed_data/model_ui_params_image.json`

| profile | 适用 |
|---------|------|
| `image-tpl-banana-chat` | Manju/Gemini Banana sync+async |
| `image-tpl-banana-chat-flash-lite` | Flash Lite 仅 1K |

---

## api_doc 最小字段

```json
{
  "dispatch_mode": "async",
  "intro": "一句话说明创建/查询路由",
  "endpoints": [
    {"method": "POST", "path": "{{base}}/chat/completions", "description": "..."},
    {"method": "GET", "path": "{{base}}/videos/{task_id}", "description": "..."}
  ],
  "basic_request_json": { "model": "{{model}}", "...": "..." },
  "request_json": { "...": "完整示例" },
  "params": [{"name": "model", "description": "..."}],
  "create_response_json": { "...": "..." },
  "query_response_json": { "...": "..." }
}
```

---

## 适配代码检查表

### 生图

- [ ] `imagevendor.Is*OriginModel` 仅匹配 internal 前缀
- [ ] 文生图 / 图生图路由分离（generations vs chat+image_url）
- [ ] async poll_url / task_id 轮询
- [ ] R2 rehost 策略（`ResolveRehostPolicy`）
- [ ] 响应 `model` 字段 outbound patch

### 视频

- [ ] 上游创建 URL 与文档一致
- [ ] 请求字段映射（如 `seconds`→`sora2_duration`，`size`→`sora2_ratio`）
- [ ] 非标准 JSON（嵌套 `data` 对象）解析
- [ ] 状态映射：`running`/`succeeded`/…
- [ ] 成片 URL 多路径提取
- [ ] 按秒计费从 `properties.duration` 或 `usage.seconds` 读取

---

## 源站常用命令

```bash
# Postgres
ssh contabo "docker exec newapi-postgres psql -U root -d new-api -c '...'"

# 查看容器网络
ssh contabo "docker inspect cangyuan-stack-new-api-b-1 --format '{{json .NetworkSettings.Networks}}'"

# ModelPrice
ssh contabo "docker exec newapi-postgres psql -U root -d new-api -c \"SELECT value::json->'<model>' FROM options WHERE key='ModelPrice';\""
```

**生产环境**：禁止 Agent 自行 `docker restart` / `docker compose restart`。渠道缓存在 `SYNC_FREQUENCY`（默认 60s）内自动刷新。
