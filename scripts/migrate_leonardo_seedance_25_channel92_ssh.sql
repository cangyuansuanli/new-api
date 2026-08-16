-- Leonardo Seedance 2.5：仅上架到生产渠道 92。
-- cy-origin: docker exec -i newapi-postgres psql -U root -d new-api < migrate_leonardo_seedance_25_channel92_ssh.sql

BEGIN;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM channels WHERE id = 92) THEN
        RAISE EXCEPTION 'channel 92 does not exist';
    END IF;
END $$;

INSERT INTO model_ui_param_profiles (
    capability, profile_id, match, sort_order, api_mode, payload_builder,
    requires_reference_media, poll_status, poll, reference_limits,
    params, option_rules, hints, note, created_time, updated_time
) VALUES (
    'video',
    'video-tpl-seedance-2.5-subscription-async',
    '["cy-sd4-seedance-2.5-480p","cy-sd4-seedance-2.5-720p"]',
    0,
    'videos-json-async',
    'seedance-flat',
    FALSE,
    '',
    '{}',
    '{"images":30,"videos":10,"audios":10,"imageMaxBytes":26214400,"videoMaxBytes":209715200,"audioMaxBytes":15728640,"video":{"maxDurationMs":30200,"totalMaxDurationMs":30200},"audio":{"maxDurationMs":30200,"totalMaxDurationMs":30200},"fullReferenceMode":{"label":"多模态","descriptionWithImages":"多模态：图 + 可选视频/音频"},"validationHint":"参考图 png/jpg/webp ≤25MB（最多 30）；参考视频 mp4/mov ≤200MB，最多 10 条且单条/总时长 ≤30.2 秒，宽高各 720–2160px、24–60 FPS；参考音频 mp3/wav ≤15MB、最多 10 条且单条/总时长 ≤30.2 秒。","showTempMediaHint":true,"prependReferenceGuide":true}',
    '{"resolution":{"enabled":false},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"},{"value":"1:1","label":"方形"},{"value":"21:9","label":"宽银幕"},{"value":"3:4","label":"3:4"},{"value":"4:3","label":"4:3"}]},"duration":{"enabled":true,"numericOptions":[4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29],"min":4,"max":29},"generateAudio":{"enabled":true,"hint":"是否生成原生音频，默认开启"},"watermark":{"enabled":false},"seed":{"enabled":false},"widthHeight":{"enabled":false},"frameInputs":{"enabled":true,"hint":"首尾帧与多模态（参考图/视频/音频）二选一；成对指定 first + last"}}',
    '[]',
    '[{"text":"模型名决定固定清晰度，按秒计费；480p 最长 30 秒（API），720p 最长 29 秒。"},{"text":"最多 30 图 / 10 视频 / 10 音频；参考视频和音频单条及合计均不超过 30.2 秒。"}]',
    'Seedance 2.5 两档固定清晰度 SKU；exact model match。',
    EXTRACT(EPOCH FROM NOW())::BIGINT,
    EXTRACT(EPOCH FROM NOW())::BIGINT
)
ON CONFLICT (capability, profile_id) DO UPDATE SET
    match = EXCLUDED.match,
    sort_order = EXCLUDED.sort_order,
    api_mode = EXCLUDED.api_mode,
    payload_builder = EXCLUDED.payload_builder,
    requires_reference_media = EXCLUDED.requires_reference_media,
    poll_status = EXCLUDED.poll_status,
    poll = EXCLUDED.poll,
    reference_limits = EXCLUDED.reference_limits,
    params = EXCLUDED.params,
    option_rules = EXCLUDED.option_rules,
    hints = EXCLUDED.hints,
    note = EXCLUDED.note,
    deleted_at = NULL,
    updated_time = EXCLUDED.updated_time;

INSERT INTO models (
    model_name, description, tags, vendor_id, endpoints, status, sync_official,
    video_profile_id, created_time, updated_time
)
SELECT v.model_name, v.description, v.tags, 4,
       '{"openai-video":{"path":"/v1/videos","method":"POST"}}',
       1, 0, 'video-tpl-seedance-2.5-subscription-async',
       EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('cy-sd4-seedance-2.5-480p', 'Seedance 2.5 标准 480p。支持 4–30 秒、原生音频和多模态参考，按秒计费。', 'video,seedance,subscription,2.5,480p'),
    ('cy-sd4-seedance-2.5-720p', 'Seedance 2.5 HD 720p。无参考视频支持 4–29 秒，带参考视频支持 4–18 秒；支持原生音频和多模态参考，按秒计费。', 'video,seedance,subscription,2.5,720p')
) AS v(model_name, description, tags)
WHERE NOT EXISTS (
    SELECT 1 FROM models m WHERE m.model_name = v.model_name AND m.deleted_at IS NULL
);

UPDATE models AS m SET
    description = v.description,
    tags = v.tags,
    vendor_id = 4,
    endpoints = '{"openai-video":{"path":"/v1/videos","method":"POST"}}',
    status = 1,
    sync_official = 0,
    video_profile_id = 'video-tpl-seedance-2.5-subscription-async',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('cy-sd4-seedance-2.5-480p', 'Seedance 2.5 标准 480p。支持 4–30 秒、原生音频和多模态参考，按秒计费。', 'video,seedance,subscription,2.5,480p'),
    ('cy-sd4-seedance-2.5-720p', 'Seedance 2.5 HD 720p。无参考视频支持 4–29 秒，带参考视频支持 4–18 秒；支持原生音频和多模态参考，按秒计费。', 'video,seedance,subscription,2.5,720p')
) AS v(model_name, description, tags)
WHERE m.model_name = v.model_name AND m.deleted_at IS NULL;

UPDATE channels SET
    models = concat_ws(',', NULLIF(models, ''),
        CASE WHEN POSITION('cy-sd4-seedance-2.5-480p' IN COALESCE(models, '')) = 0 THEN 'cy-sd4-seedance-2.5-480p' END,
        CASE WHEN POSITION('cy-sd4-seedance-2.5-720p' IN COALESCE(models, '')) = 0 THEN 'cy-sd4-seedance-2.5-720p' END
    ),
    model_mapping = (
        COALESCE(NULLIF(model_mapping, '')::jsonb, '{}'::jsonb) ||
        '{"cy-sd4-seedance-2.5-480p":"seedance-2.5","cy-sd4-seedance-2.5-720p":"seedance-2.5"}'::jsonb
    )::text
WHERE id = 92;

DELETE FROM abilities
WHERE channel_id = 92
  AND model IN ('cy-sd4-seedance-2.5-480p', 'cy-sd4-seedance-2.5-720p');

INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT g.grp, m.model, 92, TRUE, 100, 100
FROM (VALUES ('VIDEO'), ('全模型-无claude/gpt'), ('downstream-canghai')) AS g(grp)
CROSS JOIN (VALUES
    ('cy-sd4-seedance-2.5-480p'),
    ('cy-sd4-seedance-2.5-720p')
) AS m(model);

-- SD4 品牌名用于模型广场；中性名保留为可切换的 API 入站路由。
INSERT INTO model_public_aliases (internal_name, public_name, created_time, updated_time)
VALUES
    ('cy-sd4-seedance-2.5-480p', 'sd4-seedance-2.5-480p', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-sd4-seedance-2.5-720p', 'sd4-seedance-2.5-720p', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (internal_name) DO UPDATE SET
    public_name = EXCLUDED.public_name,
    deleted_at = NULL,
    updated_time = EXCLUDED.updated_time;

INSERT INTO model_routing_aliases (public_name, internal_name, note, created_time, updated_time)
VALUES
    ('seedance-2.5-480p', 'cy-sd4-seedance-2.5-480p', 'Neutral Seedance 2.5 480p -> SD4 Leonardo', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('seedance-2.5-720p', 'cy-sd4-seedance-2.5-720p', 'Neutral Seedance 2.5 720p -> SD4 Leonardo', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (public_name) DO UPDATE SET
    internal_name = EXCLUDED.internal_name,
    note = EXCLUDED.note,
    deleted_at = NULL,
    updated_time = EXCLUDED.updated_time;

-- API 文档与售价和模型上架保持同一事务。
UPDATE models AS m SET
    api_doc = jsonb_build_object(
        'dispatch_mode', 'async',
        'intro', format(
            'Seedance 2.5 固定 %s 视频模型，$%s/秒，支持 4–%s 秒。通过统一 /v1/videos API 创建、轮询并下载成片；模型名决定清晰度。',
            v.resolution, v.price, v.max_seconds
        ),
        'endpoints', jsonb_build_array(
            jsonb_build_object('method', 'POST', 'path', '{{base}}/videos', 'description', '创建异步视频任务。'),
            jsonb_build_object('method', 'GET', 'path', '{{base}}/videos/{task_id}', 'description', '查询任务状态和成片地址。'),
            jsonb_build_object('method', 'GET', 'path', '{{base}}/videos/{task_id}/content', 'description', '下载已完成任务的成片。')
        ),
        'params', jsonb_build_array(
            jsonb_build_object('name', 'model', 'description', format('必填，当前固定 %s 的模型名称。', v.resolution)),
            jsonb_build_object('name', 'prompt', 'description', '必填，视频内容描述，最多 5000 个 Unicode 字符。'),
            jsonb_build_object('name', 'duration', 'description', format('整数 4–%s 秒，默认 8。', v.max_seconds)),
            jsonb_build_object('name', 'aspect_ratio', 'description', '21:9、16:9、4:3、1:1、3:4 或 9:16。'),
            jsonb_build_object('name', 'generate_audio', 'description', '是否生成原生音频，布尔值，默认 true。'),
            jsonb_build_object('name', 'reference_image_urls', 'description', '参考图 URL 数组，最多 30 张。'),
            jsonb_build_object('name', 'reference_videos', 'description', '参考视频 URL 数组，最多 10 条，单条和合计均不超过 30.2 秒。'),
            jsonb_build_object('name', 'reference_audios', 'description', '参考音频 URL 数组，最多 10 条，单条和合计均不超过 30.2 秒。'),
            jsonb_build_object('name', 'first_image_url', 'description', '首帧 URL；必须与 last_image_url 成对提供，并与多模态参考素材互斥。'),
            jsonb_build_object('name', 'last_image_url', 'description', '尾帧 URL；必须与 first_image_url 成对提供，并与多模态参考素材互斥。')
        ),
        'basic_request_json', jsonb_build_object(
            'model', '{{model}}',
            'prompt', 'A calm blue sphere floating in a white studio',
            'duration', 4,
            'aspect_ratio', '16:9',
            'generate_audio', false
        ),
        'request_json', jsonb_build_object(
            'model', '{{model}}',
            'prompt', 'Use the subject and visual style from the references',
            'duration', 8,
            'aspect_ratio', '9:16',
            'generate_audio', true,
            'reference_image_urls', jsonb_build_array('https://example.com/subject.png')
        ),
        'create_response_json', jsonb_build_object(
            'id', 'task_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',
            'object', 'video',
            'model', '{{model}}',
            'status', 'queued',
            'progress', 0,
            'seconds', '4',
            'size', v.size
        ),
        'query_response_json', jsonb_build_object(
            'id', 'task_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',
            'object', 'video',
            'model', '{{model}}',
            'status', 'completed',
            'progress', 100,
            'seconds', '4',
            'size', v.size,
            'metadata', jsonb_build_object('video_url', '{{base}}/videos/task_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx/content'),
            'usage', jsonb_build_object('seconds', 4)
        )
    )::text,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('cy-sd4-seedance-2.5-480p', '480p', '854x480', 30, 0.25::numeric),
    ('cy-sd4-seedance-2.5-720p', '720p', '1280x720', 29, 0.35::numeric)
) AS v(model_name, resolution, size, max_seconds, price)
WHERE m.model_name = v.model_name AND m.deleted_at IS NULL;

UPDATE options SET value = (
    COALESCE(NULLIF(value, '')::jsonb, '{}'::jsonb) ||
    '{"cy-sd4-seedance-2.5-480p":0.25,"cy-sd4-seedance-2.5-720p":0.35}'::jsonb
)::text
WHERE key = 'ModelPrice';

UPDATE options SET value = (
    COALESCE(NULLIF(value, '')::jsonb, '{}'::jsonb) ||
    '{"cy-sd4-seedance-2.5-480p":"per_second","cy-sd4-seedance-2.5-720p":"per_second"}'::jsonb
)::text
WHERE key = 'billing_setting.billing_mode';

UPDATE options SET value = (
    COALESCE(NULLIF(value, '')::jsonb, '{}'::jsonb)
      - 'cy-sd4-seedance-2.5-480p'
      - 'cy-sd4-seedance-2.5-720p'
)::text
WHERE key = 'billing_setting.request_unit';

COMMIT;
