-- MiniMax3 / Hailuo 03 三档分辨率与 5/3/3 参考素材升级。
-- cy-origin: docker exec -i newapi-postgres psql -U root -d new-api < migrate_cy_sd4_minimax3_upgrade_ssh.sql

BEGIN;

UPDATE model_ui_param_profiles
SET reference_limits = '{"images":5,"videos":3,"audios":3,"imageMaxBytes":26214400,"videoMaxBytes":209715200,"audioMaxBytes":15728640,"video":{"minDurationMs":2000,"maxDurationMs":15000,"totalMaxDurationMs":15000},"audio":{"minDurationMs":2000,"maxDurationMs":15000,"totalMaxDurationMs":15000},"fullReferenceMode":{"label":"多模态","descriptionWithImages":"多模态：参考图/视频 + 可选参考音频"},"validationHint":"参考图 png/jpg/webp ≤25MB（最多 5）；参考视频 mp4/mov ≤200MB，最多 3 条，单条 2–15 秒、合计 ≤15 秒，宽高各 432–2160px、24–60 FPS；参考音频 mp3/wav ≤15MB，最多 3 条，单条 2–15 秒、合计 ≤15 秒。使用参考音频时，必须同时提供至少 1 条参考视频。","showTempMediaHint":true,"prependReferenceGuide":true}',
    params = jsonb_set(params::jsonb, '{resolution}', '{"enabled":false}'::jsonb, true)::text,
    hints = '[{"text":"模型名决定固定清晰度：768p / 2K / 4K；多模态最多 5 图 / 3 视频 / 3 音频。"},{"text":"视频与音频单条均 2–15 秒、各自合计 ≤15 秒；使用参考音频时必须同时提供至少 1 条参考视频。"}]',
    note = 'MiniMax3 三个固定清晰度 SKU；按条计费。',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video'
  AND profile_id = 'video-tpl-minimax-h3-2k-async'
  AND deleted_at IS NULL;

INSERT INTO models (
    model_name, description, tags, vendor_id, endpoints, status, sync_official,
    video_profile_id, created_time, updated_time
)
SELECT v.model_name, v.description, v.tags, 4,
       '{"openai-video":{"path":"/v1/videos","method":"POST"}}',
       1, 0, 'video-tpl-minimax-h3-2k-async',
       EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('cy-sd4-minimax-h3-768p', 'MiniMax3 768p。文生/图生/多模态/首尾帧，5–15 秒。', 'video,minimax,h3,768p'),
    ('cy-sd4-minimax-h3-2k',   'MiniMax3 2K。文生/图生/多模态/首尾帧，5–15 秒。', 'video,minimax,h3,2k'),
    ('cy-sd4-minimax-h3-4k',   'MiniMax3 4K。文生/图生/多模态/首尾帧，5–15 秒。', 'video,minimax,h3,4k')
) AS v(model_name, description, tags)
WHERE NOT EXISTS (
    SELECT 1 FROM models m WHERE m.model_name = v.model_name AND m.deleted_at IS NULL
);

UPDATE models AS m
SET description = v.description,
    tags = v.tags,
    vendor_id = 4,
    endpoints = '{"openai-video":{"path":"/v1/videos","method":"POST"}}',
    status = 1,
    sync_official = 0,
    video_profile_id = 'video-tpl-minimax-h3-2k-async',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('cy-sd4-minimax-h3-768p', 'MiniMax3 768p。文生/图生/多模态/首尾帧，5–15 秒。', 'video,minimax,h3,768p'),
    ('cy-sd4-minimax-h3-2k',   'MiniMax3 2K。文生/图生/多模态/首尾帧，5–15 秒。', 'video,minimax,h3,2k'),
    ('cy-sd4-minimax-h3-4k',   'MiniMax3 4K。文生/图生/多模态/首尾帧，5–15 秒。', 'video,minimax,h3,4k')
) AS v(model_name, description, tags)
WHERE m.model_name = v.model_name AND m.deleted_at IS NULL;

INSERT INTO model_public_aliases (internal_name, public_name, created_time, updated_time)
SELECT v.internal_name, v.public_name,
       EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('cy-sd4-minimax-h3-768p', 'minimax-h3-768p'),
    ('cy-sd4-minimax-h3-2k',   'minimax-h3-2k'),
    ('cy-sd4-minimax-h3-4k',   'minimax-h3-4k')
) AS v(internal_name, public_name)
ON CONFLICT (internal_name) DO UPDATE SET
    public_name = EXCLUDED.public_name,
    updated_time = EXCLUDED.updated_time,
    deleted_at = NULL;

UPDATE channels
SET models = 'cy-sd4-minimax-h3-768p,cy-sd4-minimax-h3-2k,cy-sd4-minimax-h3-4k',
    model_mapping = '{"cy-sd4-minimax-h3-768p":"hailuo-03","cy-sd4-minimax-h3-2k":"hailuo-03","cy-sd4-minimax-h3-4k":"hailuo-03"}'
WHERE tag = 'leonardo-minimax-h3'
   OR name = 'leonardo-minimax-h3-2k';

INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT g.grp, m.model_name, ch.id, true, 100, 100
FROM channels ch
CROSS JOIN (VALUES ('VIDEO'), ('全模型-无claude/gpt')) AS g(grp)
CROSS JOIN (VALUES
    ('cy-sd4-minimax-h3-768p'),
    ('cy-sd4-minimax-h3-2k'),
    ('cy-sd4-minimax-h3-4k')
) AS m(model_name)
WHERE ch.tag = 'leonardo-minimax-h3' OR ch.name = 'leonardo-minimax-h3-2k'
ON CONFLICT ("group", model, channel_id) DO UPDATE SET
    enabled = true,
    priority = EXCLUDED.priority,
    weight = EXCLUDED.weight;

UPDATE options
SET value = (value::jsonb || '{"cy-sd4-minimax-h3-768p":2.9,"cy-sd4-minimax-h3-2k":3.9,"cy-sd4-minimax-h3-4k":5.6}'::jsonb)::text
WHERE key = 'ModelPrice';

UPDATE options
SET value = (value::jsonb || '{"cy-sd4-minimax-h3-768p":"per_request","cy-sd4-minimax-h3-2k":"per_request","cy-sd4-minimax-h3-4k":"per_request"}'::jsonb)::text
WHERE key = 'billing_setting.billing_mode';

UPDATE options
SET value = (value::jsonb || '{"cy-sd4-minimax-h3-768p":"generation","cy-sd4-minimax-h3-2k":"generation","cy-sd4-minimax-h3-4k":"generation"}'::jsonb)::text
WHERE key = 'billing_setting.request_unit';

UPDATE models AS m
SET api_doc = jsonb_build_object(
        'dispatch_mode', 'async',
        'intro', format('MiniMax3 %s，5–15 秒。模型名锁定清晰度，请求中的 resolution 不会改变计价档。多模态最多 5 图 / 3 视频 / 3 音频；使用参考音频时，必须同时提供至少 1 条参考视频。', v.resolution),
        'params', jsonb_build_array(
            jsonb_build_object('name', 'model', 'description', '必填，传模型广场 public 名。'),
            jsonb_build_object('name', 'prompt', 'description', '必填，视频描述，最多 2000 字符。'),
            jsonb_build_object('name', 'aspect_ratio', 'description', '21:9、16:9、4:3、1:1、3:4 或 9:16。'),
            jsonb_build_object('name', 'duration', 'description', '整数 5–15 秒。'),
            jsonb_build_object('name', 'resolution', 'description', format('由当前 SKU 固定为 %s，请求值会被渠道覆盖。', v.resolution)),
            jsonb_build_object('name', 'generate_audio', 'description', '是否生成原生音频；首尾帧模式不可用。'),
            jsonb_build_object('name', 'reference_image_urls', 'description', '参考图 URL 数组，最多 5 张，PNG/JPG/WEBP，单张最大 25MB。'),
            jsonb_build_object('name', 'reference_videos', 'description', '参考视频 URL 数组，最多 3 条；MP4/MOV，单条 2–15 秒、合计最多 15 秒，432–2160px，24–60 FPS。'),
            jsonb_build_object('name', 'reference_audios', 'description', '参考音频 URL 数组，最多 3 条；MP3/WAV，单条 2–15 秒、合计最多 15 秒。使用时必须同时提供至少 1 条参考视频。'),
            jsonb_build_object('name', 'first_image_url', 'description', '首帧 URL；必须与 last_image_url 成对提供，并与多模态参考素材互斥。'),
            jsonb_build_object('name', 'last_image_url', 'description', '尾帧 URL；必须与 first_image_url 成对提供，并与多模态参考素材互斥。')
        ),
        'endpoints', jsonb_build_array(
            jsonb_build_object('method', 'POST', 'path', '{{base}}/videos', 'description', '创建视频任务。'),
            jsonb_build_object('method', 'GET', 'path', '{{base}}/videos/{task_id}', 'description', '查询任务状态与结果。'),
            jsonb_build_object('method', 'GET', 'path', '{{base}}/videos/{task_id}/content', 'description', '下载成片。')
        ),
        'basic_request_json', jsonb_build_object(
            'model', v.public_name, 'prompt', '雨夜霓虹街道，镜头缓慢推进',
            'duration', 8, 'aspect_ratio', '16:9'
        )
    )::text,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('cy-sd4-minimax-h3-768p', 'minimax-h3-768p', '768p'),
    ('cy-sd4-minimax-h3-2k',   'minimax-h3-2k',   '2K'),
    ('cy-sd4-minimax-h3-4k',   'minimax-h3-4k',   '4K')
) AS v(model_name, public_name, resolution)
WHERE m.model_name = v.model_name AND m.deleted_at IS NULL;

COMMIT;

SELECT model_name, video_profile_id, description
FROM models
WHERE model_name IN ('cy-sd4-minimax-h3-768p', 'cy-sd4-minimax-h3-2k', 'cy-sd4-minimax-h3-4k')
ORDER BY model_name;

SELECT id, name, models, model_mapping
FROM channels
WHERE tag = 'leonardo-minimax-h3' OR name = 'leonardo-minimax-h3-2k';

SELECT value::jsonb -> 'cy-sd4-minimax-h3-768p' AS price_768p,
       value::jsonb -> 'cy-sd4-minimax-h3-2k' AS price_2k,
       value::jsonb -> 'cy-sd4-minimax-h3-4k' AS price_4k
FROM options WHERE key = 'ModelPrice';
