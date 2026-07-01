-- 腾达 Seedance 2.0 特惠视频：profile + 模型绑定（源站 SSH 执行）
-- vps-94: docker exec -i newapi-postgres psql -U root -d new-api < migrate_tengda_seedance_2.0_ssh.sql

BEGIN;

INSERT INTO model_ui_param_profiles (
    capability, profile_id, api_mode, requires_reference_media,
    poll, poll_status, reference_limits, params, option_rules, hints,
    created_time, updated_time
) VALUES (
    'video',
    'video-tpl-tengda-seedance-2.0-async',
    'videos-json-async',
    FALSE,
    '{}',
    NULL,
    '{"images":9,"videos":0,"audios":1}',
    '{"resolution":{"enabled":true,"options":[{"value":"480p","label":"480P"},{"value":"720p","label":"720P"}]},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"},{"value":"1:1","label":"方形"},{"value":"21:9","label":"宽银幕"},{"value":"3:4","label":"3:4"},{"value":"4:3","label":"4:3"}]},"duration":{"enabled":true,"numericOptions":[4,5,6,7,8,9,10,11,12,13,14,15],"min":4,"max":15},"generateAudio":{"enabled":true,"hint":"参考音频场景建议开启；纯文生可关闭"},"watermark":{"enabled":false},"seed":{"enabled":false},"widthHeight":{"enabled":false},"frameInputs":{"enabled":true,"hint":"首帧/首尾帧与多参考图二选一；参考图请在提示词中用 @image1、@image2 引用"}}',
    '[]',
    '[{"text":"腾达 Seedance 2.0 特惠：文生/首帧/首尾帧/多参考图/参考音频；480P/720P，4–15 秒；图片须公网 URL，勿传 Base64。"}]',
    EXTRACT(EPOCH FROM NOW())::BIGINT,
    EXTRACT(EPOCH FROM NOW())::BIGINT
)
ON CONFLICT (capability, profile_id) DO UPDATE SET
    api_mode = EXCLUDED.api_mode,
    reference_limits = EXCLUDED.reference_limits,
    params = EXCLUDED.params,
    hints = EXCLUDED.hints,
    updated_time = EXCLUDED.updated_time;

UPDATE models SET
    video_profile_id = 'video-tpl-tengda-seedance-2.0-async',
    description = '腾达 Geeknow Seedance 2.0 特惠。文生/首帧/首尾帧/多参考图/参考音频，480P/720P，4–15 秒。',
    tags = 'video,seedance,tengda,geeknow,special-offer',
    vendor_id = 6,
    endpoints = '{"openai-video":{"path":"/v1/videos","method":"POST"}}',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE model_name = 'tengd-Seedance-2.0' AND deleted_at IS NULL;

COMMIT;

SELECT model_name, video_profile_id, tags
FROM models
WHERE model_name = 'tengd-Seedance-2.0' AND deleted_at IS NULL;
