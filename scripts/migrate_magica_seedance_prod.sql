-- Magica Seedance 2.0（cy-sd7）：双清晰度产品 + 渠道 + abilities + UI profile
-- 源站执行：docker exec -i newapi-postgres psql -U root -d new-api -v ON_ERROR_STOP=1 < scripts/migrate_magica_seedance_prod.sql
BEGIN;

INSERT INTO model_channel_prefixes (prefix, note, enabled, sort_order, created_time, updated_time)
VALUES ('cy-sd7-', 'Magica Seedance 2.0 双清晰度产品', TRUE, 131, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (prefix) DO UPDATE SET note = EXCLUDED.note, enabled = TRUE, updated_time = EXCLUDED.updated_time;

INSERT INTO model_public_aliases (internal_name, public_name, created_time, updated_time)
VALUES
    ('cy-sd7-seedance-2.0-720p', 'sd7-seedance-2.0-720p', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-sd7-seedance-2.0-1080p', 'sd7-seedance-2.0-1080p', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (internal_name) DO UPDATE SET
    public_name = EXCLUDED.public_name,
    updated_time = EXCLUDED.updated_time;

INSERT INTO model_ui_param_profiles (
    capability, profile_id, api_mode, payload_builder, requires_reference_media,
    poll, poll_status, reference_limits, params, option_rules, hints,
    created_time, updated_time
)
VALUES
(
    'video', 'video-tpl-magica-seedance-720p-async', 'videos-json-async', 'seedance-flat', FALSE,
    '{}', NULL,
    '{"images":5,"videos":3,"audios":3,"total":11,"fullReferenceMode":{"label":"多参参考","descriptionWithImages":"最多 5 图、3 视频、3 音频；prompt 可用 @Image1 引用"},"validationHint":"固定 720p；最多 5 张参考图、3 段参考视频、3 段参考音频。","showTempMediaHint":true,"prependReferenceGuide":true}',
    '{"resolution":{"enabled":false,"fixedLabel":"720p"},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"},{"value":"1:1","label":"方形"},{"value":"4:3","label":"4:3"},{"value":"3:4","label":"3:4"},{"value":"21:9","label":"21:9"}]},"duration":{"enabled":true,"numericOptions":[4,5,6,7,8,9,10,11,12,13,14,15],"min":4,"max":15},"generateAudio":{"enabled":true},"watermark":{"enabled":false},"seed":{"enabled":false},"widthHeight":{"enabled":false},"frameInputs":{"enabled":false}}',
    '[]',
    '[{"text":"固定 720p，按条计费；支持 4–15 秒与多种画幅比例。"},{"text":"支持文生、图生与多参参考（5 图 / 3 视频 / 3 音频）。"}]',
    EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
),
(
    'video', 'video-tpl-magica-seedance-1080p-async', 'videos-json-async', 'seedance-flat', FALSE,
    '{}', NULL,
    '{"images":5,"videos":3,"audios":3,"total":11,"fullReferenceMode":{"label":"多参参考","descriptionWithImages":"最多 5 图、3 视频、3 音频；prompt 可用 @Image1 引用"},"validationHint":"固定 1080p；最多 5 张参考图、3 段参考视频、3 段参考音频。","showTempMediaHint":true,"prependReferenceGuide":true}',
    '{"resolution":{"enabled":false,"fixedLabel":"1080p"},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"},{"value":"1:1","label":"方形"},{"value":"4:3","label":"4:3"},{"value":"3:4","label":"3:4"},{"value":"21:9","label":"21:9"}]},"duration":{"enabled":true,"numericOptions":[4,5,6,7,8,9,10,11,12,13,14,15],"min":4,"max":15},"generateAudio":{"enabled":true},"watermark":{"enabled":false},"seed":{"enabled":false},"widthHeight":{"enabled":false},"frameInputs":{"enabled":false}}',
    '[]',
    '[{"text":"固定 1080p，按秒计费；支持 4–15 秒与多种画幅比例。"},{"text":"支持文生、图生与多参参考（5 图 / 3 视频 / 3 音频）。"}]',
    EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
)
ON CONFLICT (capability, profile_id) DO UPDATE SET
    api_mode = EXCLUDED.api_mode,
    payload_builder = EXCLUDED.payload_builder,
    requires_reference_media = EXCLUDED.requires_reference_media,
    poll = EXCLUDED.poll,
    poll_status = EXCLUDED.poll_status,
    reference_limits = EXCLUDED.reference_limits,
    params = EXCLUDED.params,
    option_rules = EXCLUDED.option_rules,
    hints = EXCLUDED.hints,
    updated_time = EXCLUDED.updated_time;

INSERT INTO channels (
    type, key, status, name, weight, created_time, base_url, models, "group",
    model_mapping, priority, auto_ban, tag
)
SELECT
    55,
    k.gateway_key,
    1,
    'magica-web2api-1',
    90,
    EXTRACT(EPOCH FROM NOW())::BIGINT,
    'https://eu-ai.cangyuansuanli.cn/magica-api',
    'cy-sd7-seedance-2.0-720p,cy-sd7-seedance-2.0-1080p',
    'VIDEO,全模型-无claude/gpt',
    '{"cy-sd7-seedance-2.0-720p":"seedance-2.0","cy-sd7-seedance-2.0-1080p":"seedance-2.0"}',
    100,
    1,
    'magica-seedance'
FROM (VALUES ('57932bb4644cd04103361fc3c165cdb36af19aef22bc5fbe')) AS k(gateway_key)
WHERE NOT EXISTS (
    SELECT 1 FROM channels ch WHERE ch.name = 'magica-web2api-1'
);

INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT g.grp, m.model, ch.id, TRUE, 0, 90
FROM channels ch
CROSS JOIN (VALUES
    ('cy-sd7-seedance-2.0-720p'),
    ('cy-sd7-seedance-2.0-1080p')
) AS m(model)
CROSS JOIN (VALUES ('VIDEO'), ('全模型-无claude/gpt')) AS g(grp)
WHERE ch.name = 'magica-web2api-1'
  AND NOT EXISTS (
    SELECT 1 FROM abilities a
    WHERE a.channel_id = ch.id AND a.model = m.model AND a."group" = g.grp
  );

INSERT INTO models (
    model_name, description, tags, vendor_id, endpoints, status, sync_official,
    video_profile_id, created_time, updated_time
)
SELECT
    v.model_name, v.description, 'video,seedance,magica', 1,
    '{"openai-video":{"path":"/v1/videos","method":"POST"}}', 1, 0,
    v.profile_id,
    EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('cy-sd7-seedance-2.0-720p', 'Seedance 2.0 720p：固定 720p，按条计费；支持文生、图生与多参参考。', 'video-tpl-magica-seedance-720p-async'),
    ('cy-sd7-seedance-2.0-1080p', 'Seedance 2.0 1080p：固定 1080p，按秒计费；支持文生、图生与多参参考。', 'video-tpl-magica-seedance-1080p-async')
) AS v(model_name, description, profile_id)
WHERE NOT EXISTS (
    SELECT 1 FROM models m WHERE m.model_name = v.model_name AND m.deleted_at IS NULL
);

UPDATE models AS m SET
    description = v.description,
    tags = 'video,seedance,magica',
    endpoints = '{"openai-video":{"path":"/v1/videos","method":"POST"}}',
    status = 1,
    video_profile_id = v.profile_id,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('cy-sd7-seedance-2.0-720p', 'Seedance 2.0 720p：固定 720p，按条计费；支持文生、图生与多参参考。', 'video-tpl-magica-seedance-720p-async'),
    ('cy-sd7-seedance-2.0-1080p', 'Seedance 2.0 1080p：固定 1080p，按秒计费；支持文生、图生与多参参考。', 'video-tpl-magica-seedance-1080p-async')
) AS v(model_name, description, profile_id)
WHERE m.model_name = v.model_name AND m.deleted_at IS NULL;

COMMIT;

SELECT id, name, status, base_url, models, model_mapping FROM channels WHERE name = 'magica-web2api-1';
SELECT model_name, video_profile_id, status FROM models WHERE model_name LIKE 'cy-sd7%' AND deleted_at IS NULL;
SELECT a."group", a.model, a.channel_id, a.enabled FROM abilities a
JOIN channels ch ON ch.id = a.channel_id WHERE ch.name = 'magica-web2api-1';
