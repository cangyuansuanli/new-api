-- 渠道 114：SD8 Seedance 2.0 双产品（标准 / fast）
-- 源站: ssh contabo 'docker exec -i newapi-postgres psql -U root -d new-api < migrate_sd8_seedance_ssh.sql'
BEGIN;

INSERT INTO model_channel_prefixes (prefix, note, enabled, sort_order, created_time, updated_time)
VALUES ('cy-sd8-', '渠道 114 SD8 Seedance 2.0', TRUE, 132, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (prefix) DO UPDATE SET note = EXCLUDED.note, enabled = TRUE, updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT;

INSERT INTO model_public_aliases (internal_name, public_name, created_time, updated_time)
VALUES
    ('cy-sd8-seedance-2.0', 'sd8-seedance-2.0', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-sd8-seedance-2.0-fast', 'sd8-seedance-2.0-fast', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (internal_name) DO UPDATE SET
    public_name = EXCLUDED.public_name,
    updated_time = EXCLUDED.updated_time;

UPDATE channels SET
    models = 'cy-sd8-seedance-2.0,cy-sd8-seedance-2.0-fast',
    model_mapping = '{"cy-sd8-seedance-2.0":"sd2.0-933","cy-sd8-seedance-2.0-fast":"sd-2.0-fast-v1"}',
    "group" = 'VIDEO,全模型-无claude/gpt',
    status = 1
WHERE id = 114;

DELETE FROM abilities
WHERE channel_id = 114
  AND model IN (
      'sd-2.0-fast-v1',
      'sd2.0-933',
      'cy-sd8-seedance-2.0',
      'cy-sd8-seedance-2.0-fast'
  );

INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT g.grp, m.model, 114, TRUE, 0, 90
FROM (VALUES
    ('cy-sd8-seedance-2.0'),
    ('cy-sd8-seedance-2.0-fast')
) AS m(model)
CROSS JOIN (VALUES ('VIDEO'), ('全模型-无claude/gpt')) AS g(grp);

INSERT INTO model_ui_param_profiles (
    capability, profile_id, api_mode, payload_builder, requires_reference_media,
    poll, poll_status, reference_limits, params, option_rules, hints,
    created_time, updated_time
)
VALUES
(
    'video', 'video-tpl-sd8-seedance-facepass-async', 'videos-json-async', 'seedance-flat', FALSE,
    '{}', NULL,
    '{"images":9,"videos":3,"audios":3,"total":15,"fullReferenceMode":{"label":"多模态参考","descriptionWithImages":"图片、视频与音频参考素材须为公网 HTTPS URL"},"validationHint":"最多 9 张参考图、3 段参考视频、3 段参考音频；含人物参考图须遮眼后再上传。","showTempMediaHint":true,"prependReferenceGuide":true}',
    '{"resolution":{"enabled":false},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"},{"value":"1:1","label":"方形"},{"value":"4:3","label":"4:3"},{"value":"3:4","label":"3:4"}]},"duration":{"enabled":true,"numericOptions":[5,10,15],"min":5,"max":15},"generateAudio":{"enabled":false},"watermark":{"enabled":false},"seed":{"enabled":false},"widthHeight":{"enabled":false},"frameInputs":{"enabled":false}}',
    '[]',
    '[{"text":"¥2.9/条；时长仅支持 5、10、15 秒。"},{"text":"卡脸线路：含人物参考图须遮眼（贴纸/马赛克遮挡眼部）后再上传。"},{"text":"支持最多 9 图、3 视频、3 音频；素材须为公网 HTTPS URL。"}]',
    EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
),
(
    'video', 'video-tpl-sd8-seedance-fast-async', 'videos-json-async', 'seedance-flat', FALSE,
    '{}', NULL,
    '{"images":9,"videos":0,"audios":0,"total":9,"validationHint":"仅支持参考图，最多 9 张；不支持参考视频与参考音频。","showTempMediaHint":true,"prependReferenceGuide":true}',
    '{"resolution":{"enabled":false},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"},{"value":"1:1","label":"方形"},{"value":"4:3","label":"4:3"},{"value":"3:4","label":"3:4"}]},"duration":{"enabled":true,"numericOptions":[5,10,15],"min":5,"max":15},"generateAudio":{"enabled":false},"watermark":{"enabled":false},"seed":{"enabled":false},"widthHeight":{"enabled":false},"frameInputs":{"enabled":false}}',
    '[]',
    '[{"text":"¥1.9/条；时长仅支持 5、10、15 秒。"},{"text":"仅支持参考图，最多 9 张；不支持参考视频与参考音频。"}]',
    EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
)
ON CONFLICT (capability, profile_id) DO UPDATE SET
    api_mode = EXCLUDED.api_mode,
    payload_builder = EXCLUDED.payload_builder,
    reference_limits = EXCLUDED.reference_limits,
    params = EXCLUDED.params,
    hints = EXCLUDED.hints,
    updated_time = EXCLUDED.updated_time;

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
    ('cy-sd8-seedance-2.0', 'Seedance 2.0 卡脸版：¥2.9/条；9 图 + 3 视频 + 3 音频；含人物参考图须遮眼。', 'video-tpl-sd8-seedance-facepass-async'),
    ('cy-sd8-seedance-2.0-fast', 'Seedance 2.0 快速版：¥1.9/条；仅支持最多 9 张参考图。', 'video-tpl-sd8-seedance-fast-async')
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
    ('cy-sd8-seedance-2.0', 'Seedance 2.0 卡脸版：¥2.9/条；9 图 + 3 视频 + 3 音频；含人物参考图须遮眼。', 'video-tpl-sd8-seedance-facepass-async'),
    ('cy-sd8-seedance-2.0-fast', 'Seedance 2.0 快速版：¥1.9/条；仅支持最多 9 张参考图。', 'video-tpl-sd8-seedance-fast-async')
) AS v(model_name, description, profile_id)
WHERE m.model_name = v.model_name AND m.deleted_at IS NULL;

COMMIT;

SELECT id, models, model_mapping, "group", status FROM channels WHERE id = 114;
SELECT model_name, video_profile_id, status FROM models WHERE model_name LIKE 'cy-sd8%' AND deleted_at IS NULL;
