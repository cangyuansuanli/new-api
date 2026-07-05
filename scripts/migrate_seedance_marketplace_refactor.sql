-- Seedance 模型广场重构：下线废弃线路，按分辨率档绑定专属 profile
-- docker exec -i newapi-postgres psql -U root -d new-api < migrate_seedance_marketplace_refactor.sql

BEGIN;

-- 1. 下线废弃模型（cy-sd0 历史、cy-sd2 腾达、ctlove 等）
UPDATE models SET status = 0, updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE deleted_at IS NULL AND (
    model_name LIKE 'cy-sd0-%'
    OR model_name LIKE 'cy-sd2-%'
    OR model_name LIKE 'ctlove-seedance-%'
    OR model_name LIKE 'oairegbox-seedance-%'
    OR model_name LIKE 'tengd-%'
    OR model_name LIKE 'gz-seedance-%'
    OR model_name LIKE 'gz-video-%'
    OR video_profile_id = 'video-tpl-seedance-async'
);

-- 2. 重新启用模型广场在用的 cy-sd1 / cy-sd4
UPDATE models SET status = 1, updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE deleted_at IS NULL AND (
    model_name LIKE 'cy-sd1-seedance-2.0-%'
    OR model_name LIKE 'cy-sd4-seedance-%'
);

-- 3. 删除废弃 profile 模板
DELETE FROM model_ui_param_profiles
WHERE capability = 'video' AND profile_id IN (
    'video-tpl-seedance-async',
    'video-tpl-tengda-seedance-2.0-async',
    'video-tpl-async-ratio-duration-frame-ref9v3',
    'video-tpl-async-ratio-duration-frame-ref9v3a3'
);

-- 4. 插入按分辨率档的 oairegbox profile（933 参考，reference_* 扁平字段）
INSERT INTO model_ui_param_profiles (
    capability, profile_id, api_mode, requires_reference_media, poll_status,
    params, option_rules, hints, poll, reference_limits, created_time, updated_time
) VALUES
(
    'video', 'video-tpl-seedance-480p-async', 'videos-json-async', FALSE, '',
    '{"resolution":{"enabled":true,"fixedLabel":"480p","options":[{"value":"480p","label":"480p"}]},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"},{"value":"1:1","label":"方形"},{"value":"21:9","label":"宽银幕"},{"value":"3:4","label":"3:4"},{"value":"4:3","label":"4:3"}]},"duration":{"enabled":true,"numericOptions":[4,5,6,7,8,9,10,11,12,13,14,15],"min":4,"max":15},"generateAudio":{"enabled":false},"watermark":{"enabled":false},"seed":{"enabled":false},"widthHeight":{"enabled":false},"frameInputs":{"enabled":true,"hint":"首尾帧与 933 参考互斥；成对指定 first + last"}}',
    '[]',
    '[{"text":"Seedance 2.0 · 480p：933 全能参考（9图/3视频/3音频）、首尾帧；按秒计费。"}]',
    '{}', '{"images":9,"videos":3,"audios":3}',
    EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
),
(
    'video', 'video-tpl-seedance-720p-async', 'videos-json-async', FALSE, '',
    '{"resolution":{"enabled":true,"fixedLabel":"720p","options":[{"value":"720p","label":"720p"}]},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"},{"value":"1:1","label":"方形"},{"value":"21:9","label":"宽银幕"},{"value":"3:4","label":"3:4"},{"value":"4:3","label":"4:3"}]},"duration":{"enabled":true,"numericOptions":[4,5,6,7,8,9,10,11,12,13,14,15],"min":4,"max":15},"generateAudio":{"enabled":false},"watermark":{"enabled":false},"seed":{"enabled":false},"widthHeight":{"enabled":false},"frameInputs":{"enabled":true,"hint":"首尾帧与 933 参考互斥；成对指定 first + last"}}',
    '[]',
    '[{"text":"Seedance 2.0 · 720p：933 全能参考（9图/3视频/3音频）、首尾帧；按秒计费。"}]',
    '{}', '{"images":9,"videos":3,"audios":3}',
    EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
),
(
    'video', 'video-tpl-seedance-1080p-async', 'videos-json-async', FALSE, '',
    '{"resolution":{"enabled":true,"fixedLabel":"1080p","options":[{"value":"1080p","label":"1080p"}]},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"},{"value":"1:1","label":"方形"},{"value":"21:9","label":"宽银幕"},{"value":"3:4","label":"3:4"},{"value":"4:3","label":"4:3"}]},"duration":{"enabled":true,"numericOptions":[4,5,6,7,8,9,10,11,12,13,14,15],"min":4,"max":15},"generateAudio":{"enabled":false},"watermark":{"enabled":false},"seed":{"enabled":false},"widthHeight":{"enabled":false},"frameInputs":{"enabled":true,"hint":"首尾帧与 933 参考互斥；成对指定 first + last"}}',
    '[]',
    '[{"text":"Seedance 2.0 · 1080p：933 全能参考（9图/3视频/3音频）、首尾帧；按秒计费。"}]',
    '{}', '{"images":9,"videos":3,"audios":3}',
    EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
),
(
    'video', 'video-tpl-seedance-4k-async', 'videos-json-async', FALSE, '',
    '{"resolution":{"enabled":true,"fixedLabel":"4k","options":[{"value":"4k","label":"4K"}]},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"},{"value":"1:1","label":"方形"},{"value":"21:9","label":"宽银幕"},{"value":"3:4","label":"3:4"},{"value":"4:3","label":"4:3"}]},"duration":{"enabled":true,"numericOptions":[4,5,6,7,8,9,10,11,12,13,14,15],"min":4,"max":15},"generateAudio":{"enabled":false},"watermark":{"enabled":false},"seed":{"enabled":false},"widthHeight":{"enabled":false},"frameInputs":{"enabled":true,"hint":"首尾帧与 933 参考互斥；成对指定 first + last"}}',
    '[]',
    '[{"text":"Seedance 2.0 · 4K：933 全能参考（9图/3视频/3音频）、首尾帧；按秒计费。"}]',
    '{}', '{"images":9,"videos":3,"audios":3}',
    EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
)
ON CONFLICT (capability, profile_id) DO UPDATE SET
    api_mode = EXCLUDED.api_mode,
    params = EXCLUDED.params,
    option_rules = EXCLUDED.option_rules,
    hints = EXCLUDED.hints,
    reference_limits = EXCLUDED.reference_limits,
    updated_time = EXCLUDED.updated_time;

-- 5. 绑定模型广场模型 → 分辨率档 profile
UPDATE models SET video_profile_id = 'video-tpl-seedance-480p-async', updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE deleted_at IS NULL AND model_name LIKE 'cy-sd1-seedance-2.0-%480p';

UPDATE models SET video_profile_id = 'video-tpl-seedance-720p-async', updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE deleted_at IS NULL AND model_name LIKE 'cy-sd1-seedance-2.0-%720p';

UPDATE models SET video_profile_id = 'video-tpl-seedance-1080p-async', updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE deleted_at IS NULL AND model_name = 'cy-sd1-seedance-2.0-1080p';

UPDATE models SET video_profile_id = 'video-tpl-seedance-4k-async', updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE deleted_at IS NULL AND model_name = 'cy-sd1-seedance-2.0-4k';

UPDATE models SET video_profile_id = 'video-tpl-cy-sd4-seedance-async', updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE deleted_at IS NULL AND model_name LIKE 'cy-sd4-seedance-%';

-- 6. 禁用废弃渠道前缀
UPDATE model_channel_prefixes SET enabled = FALSE, updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE prefix IN ('cy-sd0-', 'cy-sd2-', 'cy-sd3-');

COMMIT;

SELECT model_name, video_profile_id, status
FROM models
WHERE (model_name LIKE 'cy-sd1-seedance%' OR model_name LIKE 'cy-sd4-seedance%') AND deleted_at IS NULL
ORDER BY model_name;
