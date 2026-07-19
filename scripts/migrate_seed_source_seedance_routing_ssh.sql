-- Channel 75 Seedance 2.0: independent internal/public names and 9/3/3 profile.
-- Leonardo cy-sd4 models and routing are intentionally untouched.

BEGIN;

INSERT INTO model_channel_prefixes (
    prefix, note, enabled, sort_order, created_time, updated_time
)
VALUES (
    'cy-sd5-', '视频线路 SD5', TRUE, 135,
    EXTRACT(EPOCH FROM NOW())::BIGINT,
    EXTRACT(EPOCH FROM NOW())::BIGINT
)
ON CONFLICT (prefix) DO UPDATE SET
    note = EXCLUDED.note,
    enabled = TRUE,
    deleted_at = NULL,
    updated_time = EXCLUDED.updated_time;

UPDATE channels
SET models = array_to_string(
        ARRAY(
            SELECT item
            FROM unnest(string_to_array(models, ',')) AS item
            WHERE btrim(item) NOT IN (
                'cy-sd4-seedance-2.0',
                'cy-sd4-seedance-2.0-fast',
                'cy-sd5-seedance-2.0',
                'cy-sd5-seedance-2.0-fast'
            )
        ) || ARRAY['cy-sd5-seedance-2.0', 'cy-sd5-seedance-2.0-fast'],
        ','
    ),
    model_mapping = (
        (COALESCE(NULLIF(model_mapping, ''), '{}')::jsonb
            - 'cy-sd4-seedance-2.0'
            - 'cy-sd4-seedance-2.0-fast'
            - 'cy-sd5-seedance-2.0'
            - 'cy-sd5-seedance-2.0-fast')
        || jsonb_build_object(
            'cy-sd5-seedance-2.0', 'cy-sd5-seedance-2.0',
            'cy-sd5-seedance-2.0-fast', 'cy-sd5-seedance-2.0-fast'
        )
    )::text
WHERE id = 75;

DELETE FROM abilities
WHERE channel_id = 75
  AND model IN (
      'cy-sd4-seedance-2.0', 'cy-sd4-seedance-2.0-fast',
      'cy-sd5-seedance-2.0', 'cy-sd5-seedance-2.0-fast'
  );

INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT groups.name, models.name, 75, TRUE, 100, 100
FROM (VALUES ('IMAGE'), ('VIDEO'), ('全模型-无claude/gpt'), ('对接专用')) AS groups(name)
CROSS JOIN (VALUES ('cy-sd5-seedance-2.0'), ('cy-sd5-seedance-2.0-fast')) AS models(name);

INSERT INTO model_public_aliases (internal_name, public_name, created_time, updated_time)
VALUES
    ('cy-sd5-seedance-2.0', 'sd5-seedance-2.0', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-sd5-seedance-2.0-fast', 'sd5-seedance-2.0-fast', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (internal_name) DO UPDATE SET
    public_name = EXCLUDED.public_name,
    deleted_at = NULL,
    updated_time = EXCLUDED.updated_time;

INSERT INTO model_ui_param_profiles (
    capability, profile_id, match, sort_order, api_mode, payload_builder,
    requires_reference_media, poll, poll_status, reference_limits,
    params, option_rules, hints, created_time, updated_time
) VALUES (
    'video', 'video-tpl-cy-sd5-seedance-933-async', '["cy-sd5-seedance"]', 93,
    'videos-json-async', 'seedance-flat', FALSE, '{}', NULL,
    '{"images":9,"videos":3,"audios":3,"total":12,"imageMaxBytes":10485760,"videoMaxBytes":52428800,"audioMaxBytes":15728640,"video":{"minDurationMs":1000,"maxDurationMs":15000,"totalMaxDurationMs":45000},"audio":{"maxDurationMs":15000},"fullReferenceMode":{"label":"全能参考","descriptionWithImages":"最多 9 图 / 3 视频 / 3 音频，三类合计不超过 12"},"validationHint":"全能参考最多 9 图、3 视频、3 音频，三类合计不超过 12；首尾帧与全能参考互斥。","showTempMediaHint":true,"prependReferenceGuide":true}',
    '{"resolution":{"enabled":true,"options":[{"value":"480p","label":"480p"},{"value":"720p","label":"720p"}]},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"}]},"duration":{"enabled":true,"numericOptions":[4,5,6,7,8,9,10,11,12,13,14,15],"min":4,"max":15},"generateAudio":{"enabled":true,"hint":"是否生成原生音频，默认开启"},"watermark":{"enabled":false},"seed":{"enabled":true,"min":0,"max":2147483647,"hint":"可选整数种子；相同输入可用于复现，显式 0 也会透传。"},"widthHeight":{"enabled":false},"frameInputs":{"enabled":true,"hint":"首尾帧与 9/3/3 全能参考互斥；成对指定 first + last"}}',
    '[]',
    '[{"text":"Seedance 2.0：480p / 720p、4–15 秒任意整数、可选整数 seed；支持 9 图 / 3 视频 / 3 音频且合计不超过 12，也支持首尾帧。"}]',
    EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
)
ON CONFLICT (capability, profile_id) DO UPDATE SET
    match = EXCLUDED.match, sort_order = EXCLUDED.sort_order,
    api_mode = EXCLUDED.api_mode, payload_builder = EXCLUDED.payload_builder,
    reference_limits = EXCLUDED.reference_limits, params = EXCLUDED.params,
    option_rules = EXCLUDED.option_rules, hints = EXCLUDED.hints,
    deleted_at = NULL, updated_time = EXCLUDED.updated_time;

INSERT INTO models (
    model_name, description, tags, vendor_id, endpoints, status, sync_official,
    video_profile_id, created_time, updated_time
)
SELECT v.model_name, v.description, v.tags, 1,
    '{"openai-video":{"path":"/v1/videos","method":"POST"}}',
    1, 0, 'video-tpl-cy-sd5-seedance-933-async',
    EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('cy-sd5-seedance-2.0', 'Seedance 2.0 标准版。支持 480p/720p 与 9 图、3 视频、3 音频全能参考。', 'video,seedance,sd5,933'),
    ('cy-sd5-seedance-2.0-fast', 'Seedance 2.0 Fast。参数同标准版，快速出片。', 'video,seedance,sd5,933,fast')
) AS v(model_name, description, tags)
WHERE NOT EXISTS (
    SELECT 1 FROM models m WHERE m.model_name = v.model_name AND m.deleted_at IS NULL
);

UPDATE models AS m SET
    description = v.description, tags = v.tags, vendor_id = 1,
    endpoints = '{"openai-video":{"path":"/v1/videos","method":"POST"}}',
    status = 1, sync_official = 0,
    video_profile_id = 'video-tpl-cy-sd5-seedance-933-async',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('cy-sd5-seedance-2.0', 'Seedance 2.0 标准版。支持 480p/720p 与 9 图、3 视频、3 音频全能参考。', 'video,seedance,sd5,933'),
    ('cy-sd5-seedance-2.0-fast', 'Seedance 2.0 Fast。参数同标准版，快速出片。', 'video,seedance,sd5,933,fast')
) AS v(model_name, description, tags)
WHERE m.model_name = v.model_name AND m.deleted_at IS NULL;

COMMIT;
