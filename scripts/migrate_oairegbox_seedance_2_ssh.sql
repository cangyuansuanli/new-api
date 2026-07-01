-- OAIREGBox Seedance 2.0 经济档渠道适配（源站 SSH 执行）
-- contabo: docker exec -i newapi-postgres psql -U root -d new-api < migrate_oairegbox_seedance_2_ssh.sql
-- 注册名：oairegbox-seedance-2.0-* → 上游 Seedance-2.0-*；ModelPrice 按文档原价 ¥/秒

BEGIN;

-- 1. 渠道 43
UPDATE channels SET
    models = 'oairegbox-seedance-2.0-mini-480p,oairegbox-seedance-2.0-fast-480p,oairegbox-seedance-2.0-480p,oairegbox-seedance-2.0-mini-720p,oairegbox-seedance-2.0-fast-720p,oairegbox-seedance-2.0-720p,oairegbox-seedance-2.0-1080p,oairegbox-seedance-2.0-4k',
    model_mapping = '{
  "oairegbox-seedance-2.0-mini-480p": "Seedance-2.0-mini-480p",
  "oairegbox-seedance-2.0-fast-480p": "Seedance-2.0-fast-480p",
  "oairegbox-seedance-2.0-480p": "Seedance-2.0-480p",
  "oairegbox-seedance-2.0-mini-720p": "Seedance-2.0-mini-720p",
  "oairegbox-seedance-2.0-fast-720p": "Seedance-2.0-fast-720p",
  "oairegbox-seedance-2.0-720p": "Seedance-2.0-720p",
  "oairegbox-seedance-2.0-1080p": "Seedance-2.0-1080p",
  "oairegbox-seedance-2.0-4k": "Seedance-2.0-4k"
}'::text,
    status = 1,
    name = '字节火山-Seedance-https://newapi-2.oairegbox.cc-Seedance2.0按秒'
WHERE id = 43;

-- 2. abilities
DELETE FROM abilities WHERE channel_id = 43;

INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT g.grp, m.model, 43, true, 0, 0
FROM (VALUES
    ('oairegbox-seedance-2.0-mini-480p'),
    ('oairegbox-seedance-2.0-fast-480p'),
    ('oairegbox-seedance-2.0-480p'),
    ('oairegbox-seedance-2.0-mini-720p'),
    ('oairegbox-seedance-2.0-fast-720p'),
    ('oairegbox-seedance-2.0-720p'),
    ('oairegbox-seedance-2.0-1080p'),
    ('oairegbox-seedance-2.0-4k')
) AS m(model)
CROSS JOIN (VALUES ('VIDEO'), ('全模型-无claude/gpt')) AS g(grp);

-- 3. profile：4K 档位
UPDATE model_ui_param_profiles SET
    params = jsonb_set(
        params::jsonb,
        '{resolution,options}',
        '[{"value":"480p","label":"480p"},{"value":"720p","label":"720p"},{"value":"1080p","label":"1080p"},{"value":"4k","label":"4K"}]'::jsonb
    )::text,
    option_rules = '[
  {"param":"resolution","value":"480p","disabledWhen":{"modelExcludes":"480p"}},
  {"param":"resolution","value":"1080p","disabledWhen":{"modelExcludes":"1080p"}},
  {"param":"resolution","value":"4k","disabledWhen":{"modelExcludes":"4k"}},
  {"param":"resolution","value":"720p","disabledWhen":{"modelIncludes":"480p"}},
  {"param":"resolution","value":"720p","disabledWhen":{"modelIncludes":"1080p"}},
  {"param":"resolution","value":"720p","disabledWhen":{"modelIncludes":"4k"}}
]'::text,
    hints = '[
  {"text":"480P 经济档，按秒计费；支持多参考图/视频/音频与首尾帧。","when":{"modelIncludes":"480p","modelExcludes":"720p"}},
  {"text":"720P 经济档，按秒计费；支持多参考图/视频/音频与首尾帧。","when":{"modelIncludes":"720p","modelExcludes":"480p"}},
  {"text":"1080P 超清，按秒计费；支持多参考图/视频/音频与首尾帧。","when":{"modelIncludes":"1080p"}},
  {"text":"4K 超高清，按秒计费；支持多参考图/视频/音频与首尾帧。","when":{"modelIncludes":"4k"}},
  {"text":"Seedance 2.0，按秒计费；支持 4–15 秒、多参考图/音视频与首尾帧。"}
]'::text,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video' AND profile_id = 'video-tpl-seedance-async';

-- 4. 清理旧注册名（无前缀的 Seedance-2.0-*）
UPDATE models SET deleted_at = NOW(), updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE model_name LIKE 'Seedance-2.0-%' AND deleted_at IS NULL;

-- 5. 注册 oairegbox-seedance-2.0-* 元数据
INSERT INTO models (model_name, description, tags, vendor_id, endpoints, status, sync_official, video_profile_id, created_time, updated_time)
SELECT v.model_name, v.description, v.tags, 12,
    '{"openai-video":{"path":"/v1/videos","method":"POST"}}',
    1, 0, 'video-tpl-seedance-async', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('oairegbox-seedance-2.0-mini-480p', 'OAIREGBox Seedance 2.0 mini 480p。按秒 ¥0.20/s，支持 9 图/3 视频/3 音频参考。', 'video,seedance,oairegbox,480p,mini'),
    ('oairegbox-seedance-2.0-fast-480p', 'OAIREGBox Seedance 2.0 fast 480p。按秒 ¥0.25/s，快速出片。', 'video,seedance,oairegbox,480p,fast'),
    ('oairegbox-seedance-2.0-480p', 'OAIREGBox Seedance 2.0 标准 480p。按秒 ¥0.45/s。', 'video,seedance,oairegbox,480p'),
    ('oairegbox-seedance-2.0-mini-720p', 'OAIREGBox Seedance 2.0 mini 720p。按秒 ¥0.35/s。', 'video,seedance,oairegbox,720p,mini'),
    ('oairegbox-seedance-2.0-fast-720p', 'OAIREGBox Seedance 2.0 fast 720p。按秒 ¥0.50/s。', 'video,seedance,oairegbox,720p,fast'),
    ('oairegbox-seedance-2.0-720p', 'OAIREGBox Seedance 2.0 标准 720p。按秒 ¥0.65/s。', 'video,seedance,oairegbox,720p'),
    ('oairegbox-seedance-2.0-1080p', 'OAIREGBox Seedance 2.0 1080p 超清。按秒 ¥1.50/s。', 'video,seedance,oairegbox,1080p'),
    ('oairegbox-seedance-2.0-4k', 'OAIREGBox Seedance 2.0 4K 超高清。按秒 ¥3.00/s。', 'video,seedance,oairegbox,4k')
) AS v(model_name, description, tags)
WHERE NOT EXISTS (
    SELECT 1 FROM models m WHERE m.model_name = v.model_name AND m.deleted_at IS NULL
);

UPDATE models AS m SET
    description = v.description,
    tags = v.tags,
    vendor_id = 12,
    endpoints = '{"openai-video":{"path":"/v1/videos","method":"POST"}}',
    video_profile_id = 'video-tpl-seedance-async',
    status = 1,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('oairegbox-seedance-2.0-mini-480p', 'OAIREGBox Seedance 2.0 mini 480p。按秒 ¥0.20/s，支持 9 图/3 视频/3 音频参考。', 'video,seedance,oairegbox,480p,mini'),
    ('oairegbox-seedance-2.0-fast-480p', 'OAIREGBox Seedance 2.0 fast 480p。按秒 ¥0.25/s，快速出片。', 'video,seedance,oairegbox,480p,fast'),
    ('oairegbox-seedance-2.0-480p', 'OAIREGBox Seedance 2.0 标准 480p。按秒 ¥0.45/s。', 'video,seedance,oairegbox,480p'),
    ('oairegbox-seedance-2.0-mini-720p', 'OAIREGBox Seedance 2.0 mini 720p。按秒 ¥0.35/s。', 'video,seedance,oairegbox,720p,mini'),
    ('oairegbox-seedance-2.0-fast-720p', 'OAIREGBox Seedance 2.0 fast 720p。按秒 ¥0.50/s。', 'video,seedance,oairegbox,720p,fast'),
    ('oairegbox-seedance-2.0-720p', 'OAIREGBox Seedance 2.0 标准 720p。按秒 ¥0.65/s。', 'video,seedance,oairegbox,720p'),
    ('oairegbox-seedance-2.0-1080p', 'OAIREGBox Seedance 2.0 1080p 超清。按秒 ¥1.50/s。', 'video,seedance,oairegbox,1080p'),
    ('oairegbox-seedance-2.0-4k', 'OAIREGBox Seedance 2.0 4K 超高清。按秒 ¥3.00/s。', 'video,seedance,oairegbox,4k')
) AS v(model_name, description, tags)
WHERE m.model_name = v.model_name AND m.deleted_at IS NULL;

COMMIT;

SELECT id, status, models FROM channels WHERE id = 43;
SELECT model, enabled FROM abilities WHERE channel_id = 43 ORDER BY model;
