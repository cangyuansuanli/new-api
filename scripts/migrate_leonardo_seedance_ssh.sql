-- Leonardo 订阅号池 Seedance 2.0（源站 SSH 执行）
-- contabo: docker exec -i newapi-postgres psql -U root -d new-api < migrate_leonardo_seedance_ssh.sql
--
-- 渠道需在管理后台新建（或更新已有 Sora 渠道）：
--   类型: Sora (55)
--   Base URL: http://leonardo-web2api:8000
--   Key: 与 LEONARDO_WEB2API_GATEWAY_TOKEN 一致
--   models: leonardo-seedance-2.0,leonardo-seedance-2.0-fast
--   model_mapping: {"leonardo-seedance-2.0":"seedance-2.0","leonardo-seedance-2.0-fast":"seedance-2.0-fast"}

BEGIN;

-- 1. 注册模型元数据
INSERT INTO models (model_name, description, tags, vendor_id, endpoints, status, sync_official, video_profile_id, created_time, updated_time)
SELECT v.model_name, v.description, v.tags, 6,
    '{"openai-video":{"path":"/v1/videos","method":"POST"}}',
    1, 0, 'video-tpl-seedance-async', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('leonardo-seedance-2.0', 'Leonardo 订阅号 Seedance 2.0。文生/图生/多模态/首尾帧，标准 480p / HD 720p，4–15 秒。', 'video,seedance,leonardo,subscription'),
    ('leonardo-seedance-2.0-fast', 'Leonardo 订阅号 Seedance 2.0 Fast。更快出片，参数同标准版。', 'video,seedance,leonardo,subscription,fast')
) AS v(model_name, description, tags)
WHERE NOT EXISTS (
    SELECT 1 FROM models m WHERE m.model_name = v.model_name AND m.deleted_at IS NULL
);

UPDATE models AS m SET
    description = v.description,
    tags = v.tags,
    vendor_id = 6,
    endpoints = '{"openai-video":{"path":"/v1/videos","method":"POST"}}',
    video_profile_id = 'video-tpl-seedance-async',
    status = 1,
    sync_official = 0,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('leonardo-seedance-2.0', 'Leonardo 订阅号 Seedance 2.0。文生/图生/多模态/首尾帧，标准 480p / HD 720p，4–15 秒。', 'video,seedance,leonardo,subscription'),
    ('leonardo-seedance-2.0-fast', 'Leonardo 订阅号 Seedance 2.0 Fast。更快出片，参数同标准版。', 'video,seedance,leonardo,subscription,fast')
) AS v(model_name, description, tags)
WHERE m.model_name = v.model_name AND m.deleted_at IS NULL;

-- 2. profile 提示：Leonardo 仅 480p/720p；多模态 4/3/1
UPDATE model_ui_param_profiles SET
    option_rules = (
        SELECT COALESCE(jsonb_agg(DISTINCT elem ORDER BY elem), '[]'::jsonb)::text
        FROM (
            SELECT jsonb_array_elements(COALESCE(option_rules::jsonb, '[]'::jsonb)) AS elem
            UNION ALL
            SELECT * FROM jsonb_array_elements('[
                {"param":"resolution","value":"1080p","disabledWhen":{"modelIncludes":"leonardo-seedance"}},
                {"param":"resolution","value":"4k","disabledWhen":{"modelIncludes":"leonardo-seedance"}}
            ]'::jsonb)
        ) merged(elem)
    ),
    hints = (
        SELECT COALESCE(jsonb_agg(DISTINCT elem ORDER BY elem), '[]'::jsonb)::text
        FROM (
            SELECT jsonb_array_elements(COALESCE(hints::jsonb, '[]'::jsonb)) AS elem
            UNION ALL
            SELECT * FROM jsonb_array_elements('[
                {"text":"Leonardo 订阅号：标准 480p（16:9=864×496）/ HD 720p（1280×720）；多模态 4图/3视频（总时长≤15s）/1音频；与首尾帧互斥。","when":{"modelIncludes":"leonardo-seedance"}}
            ]'::jsonb)
        ) merged(elem)
    ),
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video' AND profile_id = 'video-tpl-seedance-async';

-- 3. 模型命名前缀
INSERT INTO model_channel_prefixes (prefix, note, enabled, sort_order, created_time, updated_time)
VALUES ('leonardo-', 'Leonardo 订阅号池（leonardo-web2api）', TRUE, 95, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (prefix) DO NOTHING;

COMMIT;

SELECT model_name, video_profile_id, tags
FROM models
WHERE model_name LIKE 'leonardo-seedance-%' AND deleted_at IS NULL
ORDER BY model_name;
