-- Happy House 1.0 / 1.1: production channel, model metadata, API docs and pricing.
-- Run only after the new-api relay change supporting cy-sd4-happyhouse-* is deployed.
-- Source channel 92 supplies the existing private gateway credential and connection settings.

BEGIN;

INSERT INTO model_ui_param_profiles (
    capability, profile_id, match, api_mode, payload_builder, validation_key,
    requires_reference_media, poll_status, poll, reference_limits,
    params, option_rules, hints, note, created_time, updated_time
)
SELECT
    'video', v.profile_id, jsonb_build_array(v.model_name)::text,
    'videos-json-async', 'seedance-flat', '', FALSE, '', '{}',
    v.reference_limits::jsonb::text,
    '{
      "resolution":{"enabled":true,"options":[{"value":"720p","label":"HD 720p"},{"value":"1080p","label":"Full HD 1080p"}]},
      "ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"},{"value":"1:1","label":"方形"},{"value":"3:4","label":"3:4"},{"value":"4:3","label":"4:3"}]},
      "duration":{"enabled":true,"numericOptions":[3,4,5,6,7,8,9,10,11,12,13,14,15],"min":3,"max":15},
      "generateAudio":{"enabled":true,"hint":"是否生成原生音频，默认开启"},
      "watermark":{"enabled":false},"seed":{"enabled":false},"widthHeight":{"enabled":false},"frameInputs":{"enabled":false}
    }',
    '[]', jsonb_build_array(jsonb_build_object('text', v.hint))::text,
    'Happy House 异步视频参数展示；业务参数由上游服务校验。',
    EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    (
        'video-tpl-happyhouse-1.1-async',
        'cy-sd4-happyhouse-1.1',
        '{"images":9,"videos":0,"audios":0,"imageMaxBytes":26214400,"fullReferenceMode":{"label":"多图参考","descriptionWithImages":"最多 9 张参考图"},"validationHint":"参考图支持 png/jpg/webp，单张不超过 25MB，最多 9 张。不支持参考视频、参考音频和首尾帧。","showTempMediaHint":true,"prependReferenceGuide":true}',
        '720p / 1080p，3–15 秒；支持最多 9 张参考图，不支持参考视频、参考音频和首尾帧。'
    ),
    (
        'video-tpl-happyhouse-1.0-async',
        'cy-sd4-happyhouse-1.0',
        '{"images":9,"videos":1,"audios":0,"imageMaxBytes":26214400,"videoMaxBytes":209715200,"video":{"minDurationMs":3000,"maxDurationMs":10000,"totalMaxDurationMs":10000},"fullReferenceMode":{"label":"多模态","descriptionWithImages":"参考图 + 可选 1 个参考视频"},"validationHint":"参考图支持 png/jpg/webp，单张不超过 25MB，最多 9 张；带参考视频时最多 5 张图。参考视频支持 mp4/mov，不超过 200MB，宽高各 720–2160px、24–60 FPS、3–10 秒，最多 1 个。不支持参考音频和首尾帧。","showTempMediaHint":true,"prependReferenceGuide":true}',
        '720p / 1080p，3–15 秒；支持最多 9 张参考图或 1 个参考视频，带视频时最多 5 张图。'
    )
) AS v(profile_id, model_name, reference_limits, hint)
ON CONFLICT (capability, profile_id) DO UPDATE SET
    match = EXCLUDED.match,
    api_mode = EXCLUDED.api_mode,
    payload_builder = EXCLUDED.payload_builder,
    validation_key = EXCLUDED.validation_key,
    requires_reference_media = EXCLUDED.requires_reference_media,
    poll_status = EXCLUDED.poll_status,
    poll = EXCLUDED.poll,
    reference_limits = EXCLUDED.reference_limits,
    params = EXCLUDED.params,
    option_rules = EXCLUDED.option_rules,
    hints = EXCLUDED.hints,
    note = EXCLUDED.note,
    updated_time = EXCLUDED.updated_time,
    deleted_at = NULL;

INSERT INTO models (
    model_name, description, tags, vendor_id, endpoints, status, sync_official,
    video_profile_id, api_doc, created_time, updated_time
)
SELECT
    v.model_name, v.description, v.tags, 4,
    '{"openai-video":{"path":"/v1/videos","method":"POST"}}',
    1, 0, v.profile_id,
    jsonb_build_object(
        'dispatch_mode', 'async',
        'intro', v.intro,
        'endpoints', jsonb_build_array(
            jsonb_build_object('method', 'POST', 'path', '{{base}}/videos', 'description', '创建视频任务。'),
            jsonb_build_object('method', 'GET', 'path', '{{base}}/videos/{task_id}', 'description', '查询任务状态与结果。'),
            jsonb_build_object('method', 'GET', 'path', '{{base}}/videos/{task_id}/content', 'description', '下载生成的视频。')
        ),
        'params', jsonb_build_array(
            jsonb_build_object('name', 'model', 'description', '必填，传模型广场显示的模型名。'),
            jsonb_build_object('name', 'prompt', 'description', '必填，视频描述。'),
            jsonb_build_object('name', 'duration', 'description', '成片时长，3–15 秒整数。'),
            jsonb_build_object('name', 'resolution', 'description', '720p 或 1080p。'),
            jsonb_build_object('name', 'aspect_ratio', 'description', '16:9、9:16、1:1、3:4 或 4:3。'),
            jsonb_build_object('name', 'generate_audio', 'description', '是否生成原生音频，默认 true。'),
            jsonb_build_object('name', 'reference_image_urls', 'description', v.image_help)
        ) || CASE WHEN v.supports_video THEN jsonb_build_array(
            jsonb_build_object('name', 'reference_videos', 'description', v.video_help)
        ) ELSE '[]'::jsonb END,
        'basic_request_json', jsonb_build_object(
            'model', v.public_name,
            'prompt', '雨夜城市街道，镜头平稳向前推进，电影感光影',
            'duration', 4,
            'resolution', '720p',
            'aspect_ratio', '16:9',
            'generate_audio', true
        ),
        'request_json', jsonb_build_object(
            'model', v.public_name,
            'prompt', '雨夜城市街道，镜头平稳向前推进，电影感光影',
            'duration', 4,
            'resolution', '720p',
            'aspect_ratio', '16:9',
            'generate_audio', true
        ),
        'create_response_json', jsonb_build_object('id', 'video_42', 'status', 'queued', 'progress', 0),
        'query_response_json', jsonb_build_object('id', 'video_42', 'status', 'completed', 'progress', 100, 'video_url', 'https://example.com/output.mp4')
    )::text,
    EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    (
        'cy-sd4-happyhouse-1.1', 'happyhouse-1.1', 'video-tpl-happyhouse-1.1-async',
        'Happy House 1.1 视频生成。720p / 1080p，3–15 秒，支持多图参考。',
        'video,audio,multi-reference',
        'Happy House 1.1 异步视频。按条计费；失败不计费。',
        '参考图 URL 数组，最多 9 张；支持 PNG/JPG/WEBP，单张不超过 25MB。',
        '不支持，请勿传此字段。', false
    ),
    (
        'cy-sd4-happyhouse-1.0', 'happyhouse-1.0', 'video-tpl-happyhouse-1.0-async',
        'Happy House 1.0 视频生成。720p / 1080p，3–15 秒，支持图片和视频参考。',
        'video,audio,multi-reference',
        'Happy House 1.0 异步视频。按条计费；失败不计费。',
        '参考图 URL 数组，最多 9 张；带参考视频时最多 5 张。支持 PNG/JPG/WEBP，单张不超过 25MB。',
        '可选 1 个 MP4/MOV 公网 HTTPS 直链，不超过 200MB，3–10 秒。', true
    )
) AS v(model_name, public_name, profile_id, description, tags, intro, image_help, video_help, supports_video)
WHERE NOT EXISTS (
    SELECT 1 FROM models m WHERE m.model_name = v.model_name AND m.deleted_at IS NULL
);

UPDATE models m SET
    description = v.description,
    tags = v.tags,
    vendor_id = 4,
    endpoints = '{"openai-video":{"path":"/v1/videos","method":"POST"}}',
    status = 1,
    sync_official = 0,
    video_profile_id = v.profile_id,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('cy-sd4-happyhouse-1.1', 'video-tpl-happyhouse-1.1-async', 'Happy House 1.1 视频生成。720p / 1080p，3–15 秒，支持多图参考。', 'video,audio,multi-reference'),
    ('cy-sd4-happyhouse-1.0', 'video-tpl-happyhouse-1.0-async', 'Happy House 1.0 视频生成。720p / 1080p，3–15 秒，支持图片和视频参考。', 'video,audio,multi-reference')
) AS v(model_name, profile_id, description, tags)
WHERE m.model_name = v.model_name AND m.deleted_at IS NULL;

INSERT INTO model_public_aliases (internal_name, public_name, created_time, updated_time)
VALUES
    ('cy-sd4-happyhouse-1.1', 'happyhouse-1.1', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-sd4-happyhouse-1.0', 'happyhouse-1.0', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (internal_name) DO UPDATE SET
    public_name = EXCLUDED.public_name,
    updated_time = EXCLUDED.updated_time,
    deleted_at = NULL;

-- Separate channel: copy the private credential/config in-database without exposing it.
INSERT INTO channels (
    type, key, status, name, weight, created_time, base_url, models, "group",
    model_mapping, priority, auto_ban, tag, setting, settings, remark
)
SELECT
    c.type, c.key, 1, 'cy-sd4-happyhouse', c.weight,
    EXTRACT(EPOCH FROM NOW())::BIGINT, c.base_url,
    'cy-sd4-happyhouse-1.1,cy-sd4-happyhouse-1.0',
    c."group",
    '{"cy-sd4-happyhouse-1.1":"happy-horse-1.1","cy-sd4-happyhouse-1.0":"happy-horse"}',
    c.priority, c.auto_ban, 'cy-sd4-happyhouse', c.setting, c.settings,
    'Happy House isolated video channel'
FROM channels c
WHERE c.id = 92
  AND NOT EXISTS (SELECT 1 FROM channels ch WHERE ch.name = 'cy-sd4-happyhouse');

UPDATE channels SET
    status = 1,
    models = 'cy-sd4-happyhouse-1.1,cy-sd4-happyhouse-1.0',
    model_mapping = '{"cy-sd4-happyhouse-1.1":"happy-horse-1.1","cy-sd4-happyhouse-1.0":"happy-horse"}',
    "group" = 'VIDEO,全模型-无claude/gpt,downstream-canghai',
    test_model = 'cy-sd4-happyhouse-1.1'
WHERE name = 'cy-sd4-happyhouse';

DELETE FROM abilities
WHERE channel_id IN (SELECT id FROM channels WHERE name = 'cy-sd4-happyhouse')
  AND model IN ('cy-sd4-happyhouse-1.0', 'cy-sd4-happyhouse-1.1');

INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT g.grp, m.model_name, ch.id, true, ch.priority, ch.weight
FROM channels ch
CROSS JOIN (VALUES ('VIDEO'), ('全模型-无claude/gpt'), ('downstream-canghai')) AS g(grp)
CROSS JOIN (VALUES ('cy-sd4-happyhouse-1.0'), ('cy-sd4-happyhouse-1.1')) AS m(model_name)
WHERE ch.name = 'cy-sd4-happyhouse';

INSERT INTO options (key, value)
VALUES
    ('ModelPrice', '{"cy-sd4-happyhouse-1.1":2.9,"cy-sd4-happyhouse-1.0":4.5}'),
    ('billing_setting.billing_mode', '{"cy-sd4-happyhouse-1.1":"per_request","cy-sd4-happyhouse-1.0":"per_request"}'),
    ('billing_setting.request_unit', '{"cy-sd4-happyhouse-1.1":"generation","cy-sd4-happyhouse-1.0":"generation"}')
ON CONFLICT (key) DO UPDATE SET
    value = options.value::jsonb || EXCLUDED.value::jsonb;

COMMIT;

SELECT model_name, video_profile_id, description, length(api_doc) AS api_doc_len
FROM models WHERE model_name IN ('cy-sd4-happyhouse-1.0', 'cy-sd4-happyhouse-1.1') AND deleted_at IS NULL
ORDER BY model_name;

SELECT internal_name, public_name
FROM model_public_aliases
WHERE internal_name IN ('cy-sd4-happyhouse-1.0', 'cy-sd4-happyhouse-1.1')
ORDER BY internal_name;

SELECT id, name, status, base_url, "group", models, model_mapping, test_model
FROM channels WHERE name = 'cy-sd4-happyhouse';

SELECT key,
       value::jsonb->'cy-sd4-happyhouse-1.1' AS price_1_1,
       value::jsonb->'cy-sd4-happyhouse-1.0' AS price_1_0
FROM options WHERE key IN ('ModelPrice', 'billing_setting.billing_mode', 'billing_setting.request_unit')
ORDER BY key;
