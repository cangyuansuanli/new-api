-- Leonardo Seedance 2.5：仅上架到生产渠道 92。
-- api_doc 与 ModelPrice 由 seed_leonardo_seedance_25_api_doc.py 写入。
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
    '{"images":10,"videos":3,"audios":1,"imageMaxBytes":26214400,"videoMaxBytes":209715200,"audioMaxBytes":15728640,"video":{"maxDurationMs":30200,"totalMaxDurationMs":30200},"audio":{"maxDurationMs":30200},"fullReferenceMode":{"label":"多模态","descriptionWithImages":"多模态：图 + 可选视频/音频"},"validationHint":"参考图 png/jpg/webp ≤25MB（最多 10）；参考视频 mp4/mov ≤200MB，最多 3 条且单条/总时长 ≤30.2 秒，宽高各 720–2160px、24–60 FPS；参考音频 mp3/wav ≤15MB、最多 1 条且 ≤30.2 秒。","showTempMediaHint":true,"prependReferenceGuide":true}',
    '{"resolution":{"enabled":false},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"},{"value":"1:1","label":"方形"},{"value":"21:9","label":"宽银幕"},{"value":"3:4","label":"3:4"},{"value":"4:3","label":"4:3"}]},"duration":{"enabled":true,"numericOptions":[4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29],"min":4,"max":29},"generateAudio":{"enabled":true,"hint":"是否生成原生音频，默认开启"},"watermark":{"enabled":false},"seed":{"enabled":false},"widthHeight":{"enabled":false},"frameInputs":{"enabled":true,"hint":"首尾帧与多模态（参考图/视频/音频）二选一；成对指定 first + last"}}',
    '[]',
    '[{"text":"模型名决定固定清晰度，按秒计费；480p 最长 30 秒（API），720p 最长 29 秒。"},{"text":"最多 10 图 / 3 视频 / 1 音频；参考视频和音频单条及合计均不超过 30.2 秒。"}]',
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
    ('cy-sd4-seedance-2.5-720p', 'Seedance 2.5 HD 720p。支持 4–29 秒、原生音频和多模态参考，按秒计费。', 'video,seedance,subscription,2.5,720p')
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
    ('cy-sd4-seedance-2.5-720p', 'Seedance 2.5 HD 720p。支持 4–29 秒、原生音频和多模态参考，按秒计费。', 'video,seedance,subscription,2.5,720p')
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

COMMIT;
