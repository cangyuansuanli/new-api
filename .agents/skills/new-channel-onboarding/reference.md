# 新渠道入库 — 参考模板

## channels.group 与路由

`model.InitChannelCache()` 用 **`channels.group` + `channels.models`** 建索引，**不是** `abilities` 表。

```
用户 token.group = VIDEO
  → 查 group2model2channels["VIDEO"]["cy-sd8-seedance-2.0"]
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
VALUES ('cy-sd8-', 'SD8 Seedance 2.0', TRUE, 132, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (prefix) DO UPDATE SET note = EXCLUDED.note, enabled = TRUE, updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT;

-- 2. public 别名
INSERT INTO model_public_aliases (internal_name, public_name, created_time, updated_time)
VALUES ('cy-sd8-seedance-2.0', 'sd8-seedance-2.0', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (internal_name) DO UPDATE SET public_name = EXCLUDED.public_name, updated_time = EXCLUDED.updated_time;

-- 3. 渠道
UPDATE channels SET
    models = 'cy-sd8-seedance-2.0,cy-sd8-seedance-2.0-fast',
    model_mapping = '{"cy-sd8-seedance-2.0":"sd2.0-933","cy-sd8-seedance-2.0-fast":"sd-2.0-fast-v1"}',
    "group" = 'VIDEO,全模型-无claude/gpt',
    status = 1
WHERE id = 114;

-- 4. abilities
DELETE FROM abilities WHERE channel_id = 114 AND model IN ('cy-sd8-seedance-2.0', 'cy-sd8-seedance-2.0-fast');
INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT g.grp, m.model, 114, TRUE, 0, 90
FROM (VALUES ('cy-sd8-seedance-2.0'), ('cy-sd8-seedance-2.0-fast')) AS m(model)
CROSS JOIN (VALUES ('VIDEO'), ('全模型-无claude/gpt')) AS g(grp);

-- 5. models
INSERT INTO models (model_name, description, tags, vendor_id, endpoints, status, sync_official, video_profile_id, created_time, updated_time)
SELECT v.model_name, v.description, 'video,seedance', 1,
    '{"openai-video":{"path":"/v1/videos","method":"POST"}}', 1, 0, v.profile_id,
    EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('cy-sd8-seedance-2.0', '描述', 'video-tpl-sd8-seedance-async')
) AS v(model_name, description, profile_id)
WHERE NOT EXISTS (SELECT 1 FROM models m WHERE m.model_name = v.model_name AND m.deleted_at IS NULL);

COMMIT;
```

`api_doc` 与 `ModelPrice` **不要**写进 SQL，交给 seed 脚本。

---

## seed Python 骨架

```python
#!/usr/bin/env python3
"""<internal>：api_doc + ModelPrice（源站 docker 内执行）。"""

import json, subprocess, time
from seed_media_api_doc_common import capability_doc, UNIFIED_VIDEO_ENDPOINTS

MODELS = {
    "cy-sd8-seedance-2.0": {
        "price": 1.5,
        "billing_mode": "per_second",
        "request_unit": "second",
        "public_name": "sd8-seedance-2.0",
    },
}

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

def merge_json_option(key: str, updates: dict) -> None:
    current = json.loads(psql(f"SELECT value::text FROM options WHERE key='{key}'") or "{}")
    current.update(updates)
    payload = json.dumps(current, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
    psql_exec(f"INSERT INTO options (key,value) VALUES ('{key}','{payload}') ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value;")

def main() -> None:
    now = int(time.time())
    for model_name, cfg in MODELS.items():
        doc = capability_doc(
            intro=f"Seedance 2.0（{cfg['public_name']}），通过统一 /v1/videos 调用。",
            params=[{"name": "model", "description": "必填，传模型广场展示名。"}],
            endpoints=UNIFIED_VIDEO_ENDPOINTS,
        )
        payload = json.dumps(doc, ensure_ascii=False, separators=(",", ":")).replace("'", "''")
        psql_exec(f"UPDATE models SET api_doc='{payload}', updated_time={now} WHERE model_name='{model_name}' AND deleted_at IS NULL;")
    merge_json_option("ModelPrice", {m: c["price"] for m, c in MODELS.items()})
    merge_json_option("billing_setting.billing_mode", {m: c["billing_mode"] for m, c in MODELS.items()})

if __name__ == "__main__":
    main()
```
