-- Adobe2API channel 75: register Firefly Sora2/Veo video models.
-- Run after deploying the relay + video profile changes:
--   docker exec -i newapi-postgres psql -U root -d new-api < migrate_adobe2api_video_ssh.sql

BEGIN;

-- Keep public names stable after model public-name stripping.
INSERT INTO model_channel_prefixes (prefix, note, enabled, sort_order, created_time, updated_time)
VALUES ('adobe-', 'Adobe2API Firefly 图片/视频', TRUE, 130, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (prefix) DO UPDATE SET
    note = EXCLUDED.note,
    enabled = TRUE,
    updated_time = EXCLUDED.updated_time;

-- Public names are API-safe identifiers but retain the full model family/tier.
INSERT INTO model_public_aliases (internal_name, public_name, created_time, updated_time)
VALUES
    ('adobe-sora2', 'sora-2', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('adobe-sora2-pro', 'sora-2-pro', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('adobe-veo31', 'veo-3-1', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('adobe-veo31-ref', 'veo-3-1-ref', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('adobe-veo31-fast', 'veo-3-1-fast', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (internal_name) DO UPDATE SET
    public_name = EXCLUDED.public_name,
    updated_time = EXCLUDED.updated_time;

-- Keep the production DB self-contained; the JSON seed file is the repository source of truth.
INSERT INTO model_ui_param_profiles (
    capability, profile_id, api_mode, requires_reference_media,
    poll, poll_status, reference_limits, params, option_rules, hints,
    created_time, updated_time
)
VALUES
(
    'video', 'video-tpl-adobe-sora2-json-async', 'videos-json-async', FALSE,
    '{}', NULL,
    '{"images":0,"videos":0,"audios":0}',
    '{"resolution":{"enabled":false},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"}]},"duration":{"enabled":true,"min":4,"max":12,"numericOptions":[4,8,12]},"generateAudio":{"enabled":true},"watermark":{"enabled":false},"seed":{"enabled":false},"widthHeight":{"enabled":false},"frameInputs":{"enabled":false}}',
    '[]',
    '[{"text":"Adobe Firefly Sora2：POST /v1/videos 提交，GET /v1/videos/{id} 轮询；支持 4/8/12 秒与 16:9、9:16 画幅。"}]',
    EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
),
(
    'video', 'video-tpl-adobe-veo31-json-async', 'videos-json-async', FALSE,
    '{}', NULL,
    '{"images":3,"videos":0,"audios":0}',
    '{"resolution":{"enabled":true,"options":[{"value":"720p","label":"720p"},{"value":"1080p","label":"1080p"}]},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"}]},"duration":{"enabled":true,"min":4,"max":8,"numericOptions":[4,6,8]},"generateAudio":{"enabled":true},"watermark":{"enabled":false},"seed":{"enabled":false},"widthHeight":{"enabled":false},"frameInputs":{"enabled":false}}',
    '[]',
    '[{"text":"Adobe Firefly Veo 3.1：POST /v1/videos 提交，GET /v1/videos/{id} 轮询；支持 4/6/8 秒、16:9/9:16、720p/1080p；最多 3 张参考图。"}]',
    EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
)
ON CONFLICT (capability, profile_id) DO UPDATE SET
    api_mode = EXCLUDED.api_mode,
    requires_reference_media = EXCLUDED.requires_reference_media,
    poll = EXCLUDED.poll,
    poll_status = EXCLUDED.poll_status,
    reference_limits = EXCLUDED.reference_limits,
    params = EXCLUDED.params,
    option_rules = EXCLUDED.option_rules,
    hints = EXCLUDED.hints,
    updated_time = EXCLUDED.updated_time;

UPDATE models
SET video_profile_id = CASE video_profile_id
    WHEN 'video-tpl-adobe-sora2-chat' THEN 'video-tpl-adobe-sora2-json-async'
    WHEN 'video-tpl-adobe-veo31-chat' THEN 'video-tpl-adobe-veo31-json-async'
    ELSE video_profile_id
END
WHERE video_profile_id IN ('video-tpl-adobe-sora2-chat', 'video-tpl-adobe-veo31-chat');

DELETE FROM model_ui_param_profiles
WHERE capability = 'video'
  AND profile_id IN ('video-tpl-adobe-sora2-chat', 'video-tpl-adobe-veo31-chat');

-- Replace the old unregistered bare video names with names that have model metadata.
WITH existing_models AS (
    SELECT c.id, btrim(item.model) AS model, item.ord
    FROM channels c
    CROSS JOIN LATERAL unnest(string_to_array(COALESCE(c.models, ''), ',')) WITH ORDINALITY AS item(model, ord)
    WHERE c.id = 75
      AND btrim(item.model) <> ''
      AND btrim(item.model) NOT IN ('sora2', 'veo31', 'veo31-ref', 'veo31-fast')
),
wanted_models AS (
    SELECT * FROM (VALUES
        ('adobe-sora2', 10001),
        ('adobe-sora2-pro', 10002),
        ('adobe-veo31', 10003),
        ('adobe-veo31-ref', 10004),
        ('adobe-veo31-fast', 10005)
    ) AS v(model, ord)
),
merged_models AS (
    SELECT model, min(ord) AS ord
    FROM (
        SELECT model, ord FROM existing_models
        UNION ALL
        SELECT model, ord FROM wanted_models
    ) s
    GROUP BY model
),
existing_groups AS (
    SELECT c.id, btrim(item.grp) AS grp, item.ord
    FROM channels c
    CROSS JOIN LATERAL unnest(string_to_array(COALESCE(c."group", ''), ',')) WITH ORDINALITY AS item(grp, ord)
    WHERE c.id = 75 AND btrim(item.grp) <> ''
),
wanted_groups AS (
    SELECT * FROM (VALUES
        ('VIDEO', 10001)
    ) AS v(grp, ord)
),
merged_groups AS (
    SELECT grp, min(ord) AS ord
    FROM (
        SELECT grp, ord FROM existing_groups
        UNION ALL
        SELECT grp, ord FROM wanted_groups
    ) s
    GROUP BY grp
)
UPDATE channels
SET
    models = (SELECT string_agg(model, ',' ORDER BY ord) FROM merged_models),
    model_mapping = (
        (
            COALESCE(NULLIF(model_mapping, '')::jsonb, '{}'::jsonb)
            - 'sora2' - 'veo31' - 'veo31-ref' - 'veo31-fast'
        )
        || '{
          "adobe-sora2": "sora2",
          "adobe-sora2-pro": "sora2-pro",
          "adobe-veo31": "veo31",
          "adobe-veo31-ref": "veo31-ref",
          "adobe-veo31-fast": "veo31-fast"
        }'::jsonb
    )::text,
    "group" = (SELECT string_agg(grp, ',' ORDER BY ord) FROM merged_groups),
    status = 1
WHERE id = 75;

-- Video models must be selectable through VIDEO, never through IMAGE.
DELETE FROM abilities
WHERE channel_id = 75
  AND model IN ('sora2', 'veo31', 'veo31-ref', 'veo31-fast', 'adobe-sora2', 'adobe-sora2-pro', 'adobe-veo31', 'adobe-veo31-ref', 'adobe-veo31-fast');

INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT g.grp, m.model, 75, TRUE, 0, 90
FROM (VALUES
    ('adobe-sora2'),
    ('adobe-sora2-pro'),
    ('adobe-veo31'),
    ('adobe-veo31-ref'),
    ('adobe-veo31-fast')
) AS m(model)
CROSS JOIN (VALUES ('VIDEO'), ('全模型-无claude/gpt'), ('对接专用')) AS g(grp);

-- Model metadata drives the model marketplace and binds the per-model UI profile.
INSERT INTO models (model_name, description, tags, vendor_id, endpoints, status, sync_official, video_profile_id, created_time, updated_time)
SELECT v.model_name, v.description, v.tags, 2, v.endpoints, 1, 0, v.video_profile_id,
    EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('adobe-sora2', 'Adobe Firefly Sora2 视频生成。POST /v1/videos 创建任务，GET /v1/videos/{task_id} 轮询取片；支持 4/8/12 秒与 16:9、9:16。', 'video,sora,adobe,firefly', '{"openai-video":{"path":"/v1/videos","method":"POST"}}', 'video-tpl-adobe-sora2-json-async'),
    ('adobe-sora2-pro', 'Adobe Firefly Sora2 Pro 视频生成。POST /v1/videos 创建任务，GET /v1/videos/{task_id} 轮询取片；支持 4/8/12 秒与 16:9、9:16。', 'video,sora,adobe,firefly,pro', '{"openai-video":{"path":"/v1/videos","method":"POST"}}', 'video-tpl-adobe-sora2-json-async'),
    ('adobe-veo31', 'Adobe Firefly Veo 3.1 标准视频生成。POST /v1/videos 创建任务，GET /v1/videos/{task_id} 轮询取片；支持 4/6/8 秒、画幅与分辨率。', 'video,veo,adobe,firefly', '{"openai-video":{"path":"/v1/videos","method":"POST"}}', 'video-tpl-adobe-veo31-json-async'),
    ('adobe-veo31-ref', 'Adobe Firefly Veo 3.1 参考图视频生成。POST /v1/videos 创建任务，GET /v1/videos/{task_id} 轮询取片；支持最多 3 张参考图。', 'video,veo,adobe,firefly,reference', '{"openai-video":{"path":"/v1/videos","method":"POST"}}', 'video-tpl-adobe-veo31-json-async'),
    ('adobe-veo31-fast', 'Adobe Firefly Veo 3.1 Fast 视频生成。POST /v1/videos 创建任务，GET /v1/videos/{task_id} 轮询取片；支持 4/6/8 秒、画幅与分辨率。', 'video,veo,adobe,firefly,fast', '{"openai-video":{"path":"/v1/videos","method":"POST"}}', 'video-tpl-adobe-veo31-json-async')
) AS v(model_name, description, tags, endpoints, video_profile_id)
WHERE NOT EXISTS (
    SELECT 1 FROM models m WHERE m.model_name = v.model_name AND m.deleted_at IS NULL
);

UPDATE models AS m
SET
    description = v.description,
    tags = v.tags,
    vendor_id = 2,
    endpoints = v.endpoints,
    status = 1,
    sync_official = 0,
    video_profile_id = v.video_profile_id,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('adobe-sora2', 'Adobe Firefly Sora2 视频生成。POST /v1/videos 创建任务，GET /v1/videos/{task_id} 轮询取片；支持 4/8/12 秒与 16:9、9:16。', 'video,sora,adobe,firefly', '{"openai-video":{"path":"/v1/videos","method":"POST"}}', 'video-tpl-adobe-sora2-json-async'),
    ('adobe-sora2-pro', 'Adobe Firefly Sora2 Pro 视频生成。POST /v1/videos 创建任务，GET /v1/videos/{task_id} 轮询取片；支持 4/8/12 秒与 16:9、9:16。', 'video,sora,adobe,firefly,pro', '{"openai-video":{"path":"/v1/videos","method":"POST"}}', 'video-tpl-adobe-sora2-json-async'),
    ('adobe-veo31', 'Adobe Firefly Veo 3.1 标准视频生成。POST /v1/videos 创建任务，GET /v1/videos/{task_id} 轮询取片；支持 4/6/8 秒、画幅与分辨率。', 'video,veo,adobe,firefly', '{"openai-video":{"path":"/v1/videos","method":"POST"}}', 'video-tpl-adobe-veo31-json-async'),
    ('adobe-veo31-ref', 'Adobe Firefly Veo 3.1 参考图视频生成。POST /v1/videos 创建任务，GET /v1/videos/{task_id} 轮询取片；支持最多 3 张参考图。', 'video,veo,adobe,firefly,reference', '{"openai-video":{"path":"/v1/videos","method":"POST"}}', 'video-tpl-adobe-veo31-json-async'),
    ('adobe-veo31-fast', 'Adobe Firefly Veo 3.1 Fast 视频生成。POST /v1/videos 创建任务，GET /v1/videos/{task_id} 轮询取片；支持 4/6/8 秒、画幅与分辨率。', 'video,veo,adobe,firefly,fast', '{"openai-video":{"path":"/v1/videos","method":"POST"}}', 'video-tpl-adobe-veo31-json-async')
) AS v(model_name, description, tags, endpoints, video_profile_id)
WHERE m.model_name = v.model_name AND m.deleted_at IS NULL;

COMMIT;

SELECT id, name, "group", models, model_mapping, status
FROM channels
WHERE id = 75;

SELECT channel_id, "group", model, enabled, priority, weight
FROM abilities
WHERE channel_id = 75
  AND model LIKE 'adobe-%'
ORDER BY model, "group";

SELECT model_name, video_profile_id, endpoints, status
FROM models
WHERE model_name LIKE 'adobe-%' AND deleted_at IS NULL
ORDER BY model_name;
