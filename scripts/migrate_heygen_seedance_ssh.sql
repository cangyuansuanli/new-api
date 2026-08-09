-- 渠道 105：Seedance 2.0 拆分为固定 720p 按条与固定 1080p 按秒两个产品。
-- 代码和 canary 完成前保持禁用，避免旧的未拆分模型进入生产路由。
BEGIN;

INSERT INTO model_channel_prefixes (prefix, note, enabled, sort_order, created_time, updated_time)
VALUES ('cy-sd6-', '渠道 105 Seedance 2.0 双清晰度产品', TRUE, 130, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (prefix) DO UPDATE SET note = EXCLUDED.note, enabled = TRUE, updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT;

INSERT INTO model_public_aliases (internal_name, public_name, created_time, updated_time)
VALUES
    ('cy-sd6-seedance-2.0-720p', 'sd6-seedance-2.0-720p', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-sd6-seedance-2.0-1080p', 'sd6-seedance-2.0-1080p', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
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
    'video', 'video-tpl-heygen-seedance-720p-async', 'videos-json-async', 'seedance-flat', FALSE,
    '{}', NULL,
    '{"images":9,"videos":3,"audios":1,"total":12,"audio":{"maxDurationMs":15000},"fullReferenceMode":{"label":"多模态参考","descriptionWithImages":"图片、视频与音频参考素材合计最多 12 个"},"validationHint":"固定 720p；最多 9 张图片、3 段视频、1 段不超过 15 秒的音频，三类素材合计最多 12 个。音频不能单独使用；首尾帧必须成对提供，且与多模态参考互斥。","showTempMediaHint":true,"prependReferenceGuide":true}',
    '{"resolution":{"enabled":false,"fixedLabel":"720p"},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"},{"value":"1:1","label":"方形"},{"value":"4:3","label":"4:3"},{"value":"3:4","label":"3:4"}]},"duration":{"enabled":true,"numericOptions":[4,5,6,8,10,12,15],"min":4,"max":15},"generateAudio":{"enabled":false},"watermark":{"enabled":false},"seed":{"enabled":false},"widthHeight":{"enabled":false},"frameInputs":{"enabled":true,"hint":"首尾帧必须成对提供，并与多模态参考二选一"}}',
    '[]',
    '[{"text":"固定 720p，按条计费；支持 4/5/6/8/10/12/15 秒与五种画幅比例。"},{"text":"支持文生、图生、多模态参考和首尾帧；最多 9 图、3 视频、1 音频，合计最多 12 个。"}]',
    EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
),
(
    'video', 'video-tpl-heygen-seedance-1080p-async', 'videos-json-async', 'seedance-flat', FALSE,
    '{}', NULL,
    '{"images":9,"videos":3,"audios":1,"total":12,"audio":{"maxDurationMs":15000},"fullReferenceMode":{"label":"多模态参考","descriptionWithImages":"图片、视频与音频参考素材合计最多 12 个"},"validationHint":"固定 1080p；最多 9 张图片、3 段视频、1 段不超过 15 秒的音频，三类素材合计最多 12 个。音频不能单独使用；首尾帧必须成对提供，且与多模态参考互斥。","showTempMediaHint":true,"prependReferenceGuide":true}',
    '{"resolution":{"enabled":false,"fixedLabel":"1080p"},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"},{"value":"1:1","label":"方形"},{"value":"4:3","label":"4:3"},{"value":"3:4","label":"3:4"}]},"duration":{"enabled":true,"numericOptions":[4,5,6,8,10,12,15],"min":4,"max":15},"generateAudio":{"enabled":false},"watermark":{"enabled":false},"seed":{"enabled":false},"widthHeight":{"enabled":false},"frameInputs":{"enabled":true,"hint":"首尾帧必须成对提供，并与多模态参考二选一"}}',
    '[]',
    '[{"text":"固定 1080p，按秒计费；支持 4/5/6/8/10/12/15 秒与五种画幅比例。"},{"text":"支持文生、图生、多模态参考和首尾帧；最多 9 图、3 视频、1 音频，合计最多 12 个。"}]',
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

UPDATE channels SET
    models = 'cy-sd6-seedance-2.0-720p,cy-sd6-seedance-2.0-1080p',
    model_mapping = '{"cy-sd6-seedance-2.0-720p":"seedance-2.0","cy-sd6-seedance-2.0-1080p":"seedance-2.0"}',
    "group" = 'VIDEO,全模型-无claude/gpt',
    status = 2
WHERE id = 105;

DELETE FROM abilities
WHERE channel_id = 105
  AND model IN (
      'cy-sd6-seedance-2.0',
      'cy-sd6-seedance-2.0-720p',
      'cy-sd6-seedance-2.0-1080p'
  );

INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT g.grp, m.model, 105, TRUE, 0, 90
FROM (VALUES
    ('cy-sd6-seedance-2.0-720p'),
    ('cy-sd6-seedance-2.0-1080p')
) AS m(model)
CROSS JOIN (VALUES ('VIDEO'), ('全模型-无claude/gpt')) AS g(grp);

UPDATE models SET status = 0, updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE model_name = 'cy-sd6-seedance-2.0' AND deleted_at IS NULL;

INSERT INTO models (
    model_name, description, tags, vendor_id, endpoints, status, sync_official,
    video_profile_id, created_time, updated_time
)
SELECT
    v.model_name, v.description, 'video,seedance', 1,
    '{"openai-video":{"path":"/v1/videos","method":"POST"}}', 1, 0,
    v.profile_id,
    EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('cy-sd6-seedance-2.0-720p', 'Seedance 2.0 720p：固定 720p，按条计费，支持文生、图生、多模态参考与首尾帧。', 'video-tpl-heygen-seedance-720p-async'),
    ('cy-sd6-seedance-2.0-1080p', 'Seedance 2.0 1080p：固定 1080p，按秒计费，支持文生、图生、多模态参考与首尾帧。', 'video-tpl-heygen-seedance-1080p-async')
) AS v(model_name, description, profile_id)
WHERE NOT EXISTS (
    SELECT 1 FROM models m
    WHERE m.model_name = v.model_name AND m.deleted_at IS NULL
);

UPDATE models AS m SET
    description = v.description,
    tags = 'video,seedance',
    endpoints = '{"openai-video":{"path":"/v1/videos","method":"POST"}}',
    status = 1,
    video_profile_id = v.profile_id,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('cy-sd6-seedance-2.0-720p', 'Seedance 2.0 720p：固定 720p，按条计费，支持文生、图生、多模态参考与首尾帧。', 'video-tpl-heygen-seedance-720p-async'),
    ('cy-sd6-seedance-2.0-1080p', 'Seedance 2.0 1080p：固定 1080p，按秒计费，支持文生、图生、多模态参考与首尾帧。', 'video-tpl-heygen-seedance-1080p-async')
) AS v(model_name, description, profile_id)
WHERE m.model_name = v.model_name AND m.deleted_at IS NULL;

COMMIT;
