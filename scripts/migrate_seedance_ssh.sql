-- Seedance 整合：直接 SSH 执行（无需 seed 脚本）
-- vps-94: docker exec newapi-postgres psql -U root -d new-api -f /path/to/this.sql

BEGIN;

-- 1. 插入/更新统一 profile
INSERT INTO model_ui_param_profiles (
    capability, profile_id, api_mode, requires_reference_media,
    poll, poll_status, reference_limits, params, option_rules, hints,
    created_time, updated_time
) VALUES (
    'video',
    'video-tpl-seedance-async',
    'videos-json-async',
    FALSE,
    '{}',
    NULL,
    '{"images":9,"videos":3,"audios":3}',
    '{"resolution":{"enabled":true,"options":[{"value":"480p","label":"480p"},{"value":"720p","label":"720p"},{"value":"1080p","label":"1080p"}]},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"},{"value":"1:1","label":"方形"},{"value":"21:9","label":"宽银幕"},{"value":"3:4","label":"3:4"},{"value":"4:3","label":"4:3"}]},"duration":{"enabled":true,"numericOptions":[4,5,6,7,8,9,10,11,12,13,14,15],"min":4,"max":15},"generateAudio":{"enabled":false},"watermark":{"enabled":false},"seed":{"enabled":false},"widthHeight":{"enabled":false},"frameInputs":{"enabled":true,"hint":"首尾帧与多参考图/视频/音频二选一；成对指定 first + last"}}',
    '[{"param":"resolution","value":"480p","disabledWhen":{"modelExcludes":"480p"}},{"param":"resolution","value":"1080p","disabledWhen":{"modelExcludes":"1080p"}},{"param":"resolution","value":"720p","disabledWhen":{"modelIncludes":"480p"}},{"param":"resolution","value":"720p","disabledWhen":{"modelIncludes":"1080p"}}]',
    '[{"text":"480P/720P 经济档，按秒计费；支持多参考图、参考视频与首尾帧；不支持参考音频。","when":{"modelExcludes":"1080p","modelIncludes":"480p"}},{"text":"Seedance 2.0 经济档（oairegbox-seedance-2.0-*）：按秒计费；933 全能参考、首尾帧；模式由素材字段自动判定。","when":{"modelIncludes":"oairegbox-seedance-2.0"}},{"text":"Seedance 经济档，按秒计费；支持多参考图、参考视频与首尾帧；不支持参考音频。","when":{"modelIncludes":"seedance-2.0","modelExcludes":"oairegbox-seedance-2.0"}},{"text":"满血 Pro/Fast，按秒计费；支持 @Image/@Video/@Audio 全参考（9图/3视频/3音频）。","when":{"modelIncludes":"1080p"}},{"text":"Pro/Fast 720P，按秒计费；支持 @Image/@Video/@Audio 全参考（9图/3视频/3音频）。","when":{"modelIncludes":"pro-720p"}},{"text":"Pro/Fast 720P，按秒计费；支持 @Image/@Video/@Audio 全参考（9图/3视频/3音频）。","when":{"modelIncludes":"fast-720p"}},{"text":"Seedance 视频，按秒计费；支持多参考图、参考视频与首尾帧。"}]',
    EXTRACT(EPOCH FROM NOW())::BIGINT,
    EXTRACT(EPOCH FROM NOW())::BIGINT
)
ON CONFLICT (capability, profile_id) DO UPDATE SET
    api_mode = EXCLUDED.api_mode,
    reference_limits = EXCLUDED.reference_limits,
    params = EXCLUDED.params,
    option_rules = EXCLUDED.option_rules,
    hints = EXCLUDED.hints,
    updated_time = EXCLUDED.updated_time;

-- 2. 旧 profile 绑定的模型 → 统一 profile
UPDATE models SET video_profile_id = 'video-tpl-seedance-async', updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE video_profile_id IN (
    'video-tpl-async-ratio-duration-frame-ref9v3',
    'video-tpl-async-ratio-duration-frame-ref9v3a3',
    'video-tpl-json-pro-audio-seed-ref4v1',
    'video-tpl-json-fast-audio-seed-ref4v1',
    'video-tpl-json-15s-pro-audio-ref4v1a3',
    'video-tpl-json-15s-fast-audio-ref4v1a1',
    'video-tpl-json-10s-seed-ref9',
    'video-tpl-json-art-plain-ref4',
    'video-tpl-json-720-duration-ref4'
);

-- 3. OAIREGBox / CTLove Seedance（此前 profile 为空或未绑定）
UPDATE models SET video_profile_id = 'video-tpl-seedance-async', updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE deleted_at IS NULL
  AND (
    model_name LIKE 'oairegbox-seedance-%'
    OR model_name LIKE 'ctlove-seedance-%'
  );

-- 4. 禁用 gz-seedance / gz-video 渠道能力
UPDATE abilities SET enabled = false
WHERE model LIKE 'gz-seedance-%' OR model LIKE 'gz-video-%';

-- 5. 清理 poll_defaults 中的 videos-json-gz
UPDATE model_ui_param_registries
SET poll_defaults = (poll_defaults::jsonb - 'videos-json-gz')::text
WHERE poll_defaults::jsonb ? 'videos-json-gz';

-- 6. 删除旧 profile 模板
DELETE FROM model_ui_param_profiles
WHERE profile_id LIKE 'video-tpl-json-%'
   OR api_mode = 'videos-json-gz'
   OR profile_id IN (
    'video-tpl-async-ratio-duration-frame-ref9v3',
    'video-tpl-async-ratio-duration-frame-ref9v3a3'
);

COMMIT;
