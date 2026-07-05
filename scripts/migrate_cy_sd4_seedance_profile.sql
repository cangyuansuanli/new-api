-- cy-sd4 Seedance 2.0：专属 profile（参数 UI 由 profile 驱动，非客户端硬编码）
-- docker exec -i newapi-postgres psql -U root -d new-api < migrate_cy_sd4_seedance_profile.sql

BEGIN;

INSERT INTO model_ui_param_profiles (
    capability, profile_id, api_mode, requires_reference_media, poll_status,
    params, option_rules, hints, poll, reference_limits, created_time, updated_time
) VALUES (
    'video',
    'video-tpl-cy-sd4-seedance-async',
    'videos-json-async',
    FALSE,
    '',
    '{"resolution":{"enabled":true,"options":[{"value":"480p","label":"标准 480p"},{"value":"720p","label":"HD 720p"}]},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"},{"value":"1:1","label":"方形"},{"value":"21:9","label":"宽银幕"},{"value":"3:4","label":"3:4"},{"value":"4:3","label":"4:3"}]},"duration":{"enabled":true,"numericOptions":[4,5,6,7,8,9,10,11,12,13,14,15],"min":4,"max":15},"generateAudio":{"enabled":true,"hint":"是否生成原生音频，默认开启"},"watermark":{"enabled":false},"seed":{"enabled":false},"widthHeight":{"enabled":false},"frameInputs":{"enabled":true,"hint":"首尾帧与多模态（参考图/视频/音频）二选一；成对指定 first + last"}}',
    '[]',
    '[{"text":"标准 480p（16:9=864×496）/ HD 720p（1280×720）；多模态 4图/3视频（总时长≤15s）/1音频（≤15s）；与首尾帧互斥。"}]',
    '{}',
    '{"images":4,"videos":3,"audios":1}',
    EXTRACT(EPOCH FROM NOW())::BIGINT,
    EXTRACT(EPOCH FROM NOW())::BIGINT
)
ON CONFLICT (capability, profile_id) DO UPDATE SET
    api_mode = EXCLUDED.api_mode,
    params = EXCLUDED.params,
    option_rules = EXCLUDED.option_rules,
    hints = EXCLUDED.hints,
    reference_limits = EXCLUDED.reference_limits,
    updated_time = EXCLUDED.updated_time;

UPDATE models SET
    video_profile_id = 'video-tpl-cy-sd4-seedance-async',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE model_name LIKE 'cy-sd4-seedance-%' AND deleted_at IS NULL;

COMMIT;

SELECT model_name, video_profile_id
FROM models
WHERE model_name LIKE 'cy-sd4-seedance-%' AND deleted_at IS NULL
ORDER BY model_name;
