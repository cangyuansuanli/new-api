-- MiniMax H3 2K（Leonardo leonardo-france / leonardo-web2api hailuo-03）
-- contabo: docker exec -i newapi-postgres psql -U root -d new-api < migrate_cy_sd4_minimax_h3_prod.sql
-- api_doc（与 seedance-2.0 同结构）: migrate_cy_sd4_minimax_h3_api_doc.sql
--  regenerate: python3 scripts/apply_model_api_doc_from_json.py cy-sd4-minimax-h3-2k scripts/seed_data/api_doc_cy_sd4_minimax_h3_2k.json > scripts/migrate_cy_sd4_minimax_h3_api_doc.sql

BEGIN;

INSERT INTO model_ui_param_profiles (
    capability, profile_id, match, api_mode, payload_builder, validation_key,
    requires_reference_media, poll_status, poll, reference_limits,
    params, option_rules, hints, note, created_time, updated_time
) VALUES (
    'video',
    'video-tpl-minimax-h3-2k-async',
    '["cy-sd4-minimax-h3"]',
    'videos-json-async',
    'seedance-flat',
    '',
    FALSE,
    '',
    '{}',
    '{
      "images": 5,
      "videos": 0,
      "audios": 3,
      "imageMaxBytes": 26214400,
      "audioMaxBytes": 15728640,
      "audio": {
        "maxDurationMs": 15000,
        "totalMaxDurationMs": 15000
      },
      "fullReferenceMode": {
        "label": "多模态",
        "descriptionWithImages": "多模态：参考图 + 可选参考音频（不支持参考视频）"
      },
      "validationHint": "参考图 png/jpg/webp ≤25MB（最多 5）；参考音频 mp3/wav ≤15MB，最多 3 段、合计 ≤15 秒。不支持参考视频。提交后由平台校验。",
      "showTempMediaHint": true,
      "prependReferenceGuide": true
    }',
    '{
      "resolution": {
        "enabled": true,
        "options": [{"value": "2k", "label": "2K"}]
      },
      "ratio": {
        "enabled": true,
        "options": [
          {"value": "16:9", "label": "横屏"},
          {"value": "9:16", "label": "竖屏"},
          {"value": "1:1", "label": "方形"},
          {"value": "21:9", "label": "宽银幕"},
          {"value": "3:4", "label": "3:4"},
          {"value": "4:3", "label": "4:3"}
        ]
      },
      "duration": {
        "enabled": true,
        "numericOptions": [5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15],
        "min": 5,
        "max": 15
      },
      "generateAudio": {
        "enabled": true,
        "hint": "是否生成原生音频（首尾帧模式下不可用）"
      },
      "watermark": {"enabled": false},
      "seed": {"enabled": false},
      "widthHeight": {"enabled": false},
      "frameInputs": {
        "enabled": true,
        "hint": "首尾帧与多模态（参考图/音频）二选一；首尾帧下不可开原生音频"
      }
    }',
    '[]',
    '[{"text": "2K 成片；多模态 5 图 / 3 音频（合计 ≤15 秒）；与首尾帧互斥。"}]',
    'MiniMax H3 2K 异步视频；按条计费。',
    EXTRACT(EPOCH FROM NOW())::BIGINT,
    EXTRACT(EPOCH FROM NOW())::BIGINT
)
ON CONFLICT (capability, profile_id) DO UPDATE SET
    match = EXCLUDED.match,
    api_mode = EXCLUDED.api_mode,
    payload_builder = EXCLUDED.payload_builder,
    reference_limits = EXCLUDED.reference_limits,
    params = EXCLUDED.params,
    hints = EXCLUDED.hints,
    note = EXCLUDED.note,
    updated_time = EXCLUDED.updated_time;

INSERT INTO models (
    model_name, description, tags, vendor_id, endpoints, status, sync_official,
    video_profile_id, created_time, updated_time
) SELECT
    'cy-sd4-minimax-h3-2k',
    'MiniMax H3 2K。文生/图生/多模态/首尾帧，2K，5–15 秒。',
    'video,minimax,h3,2k',
    4,
    '{"openai-video":{"path":"/v1/videos","method":"POST"}}',
    1,
    0,
    'video-tpl-minimax-h3-2k-async',
    EXTRACT(EPOCH FROM NOW())::BIGINT,
    EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE NOT EXISTS (
    SELECT 1 FROM models m
    WHERE m.model_name = 'cy-sd4-minimax-h3-2k' AND m.deleted_at IS NULL
);

UPDATE models SET
    description = 'MiniMax H3 2K。文生/图生/多模态/首尾帧，2K，5–15 秒。',
    tags = 'video,minimax,h3,2k',
    vendor_id = 4,
    endpoints = '{"openai-video":{"path":"/v1/videos","method":"POST"}}',
    status = 1,
    sync_official = 0,
    video_profile_id = 'video-tpl-minimax-h3-2k-async',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE model_name = 'cy-sd4-minimax-h3-2k' AND deleted_at IS NULL;

INSERT INTO model_public_aliases (internal_name, public_name, created_time, updated_time)
VALUES (
    'cy-sd4-minimax-h3-2k',
    'minimax-h3-2k',
    EXTRACT(EPOCH FROM NOW())::BIGINT,
    EXTRACT(EPOCH FROM NOW())::BIGINT
)
ON CONFLICT (internal_name) DO UPDATE SET
    public_name = EXCLUDED.public_name,
    updated_time = EXCLUDED.updated_time,
    deleted_at = NULL;

-- 新渠道：leonardo-france 线路，仅 H3 2K
INSERT INTO channels (
    type, key, status, name, weight, created_time, base_url, models, "group",
    model_mapping, priority, auto_ban, tag
)
SELECT
    55,
    c.key,
    1,
    'leonardo-minimax-h3-2k',
    100,
    EXTRACT(EPOCH FROM NOW())::BIGINT,
    c.base_url,
    'cy-sd4-minimax-h3-2k',
    c."group",
    '{"cy-sd4-minimax-h3-2k":"hailuo-03"}',
    100,
    c.auto_ban,
    'leonardo-minimax-h3'
FROM channels c
WHERE c.id = 92
  AND NOT EXISTS (
    SELECT 1 FROM channels ch
    WHERE ch.name = 'leonardo-minimax-h3-2k' AND ch.base_url = c.base_url
  );

INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT g.grp, 'cy-sd4-minimax-h3-2k', ch.id, true, 100, 100
FROM channels ch
CROSS JOIN (VALUES ('VIDEO'), ('全模型-无claude/gpt')) AS g(grp)
WHERE ch.name = 'leonardo-minimax-h3-2k'
  AND NOT EXISTS (
    SELECT 1 FROM abilities a
    WHERE a.channel_id = ch.id AND a.model = 'cy-sd4-minimax-h3-2k' AND a."group" = g.grp
  );

INSERT INTO options (key, value)
VALUES
    ('ModelPrice', jsonb_build_object('cy-sd4-minimax-h3-2k', 4.9)::text),
    ('billing_setting.billing_mode', jsonb_build_object('cy-sd4-minimax-h3-2k', 'per_request')::text),
    ('billing_setting.request_unit', jsonb_build_object('cy-sd4-minimax-h3-2k', 'generation')::text)
ON CONFLICT (key) DO UPDATE SET
    value = CASE
        WHEN options.key = 'ModelPrice' THEN
            jsonb_set(options.value::jsonb, '{cy-sd4-minimax-h3-2k}', '4.9'::jsonb, true)::text
        WHEN options.key = 'billing_setting.billing_mode' THEN
            jsonb_set(options.value::jsonb, '{cy-sd4-minimax-h3-2k}', '"per_request"'::jsonb, true)::text
        ELSE
            jsonb_set(options.value::jsonb, '{cy-sd4-minimax-h3-2k}', '"generation"'::jsonb, true)::text
    END;

COMMIT;

SELECT model_name, video_profile_id, description
FROM models WHERE model_name = 'cy-sd4-minimax-h3-2k' AND deleted_at IS NULL;

SELECT internal_name, public_name FROM model_public_aliases WHERE internal_name = 'cy-sd4-minimax-h3-2k';

SELECT id, name, status, base_url, models, model_mapping
FROM channels WHERE name = 'leonardo-minimax-h3-2k';

SELECT a."group", a.model, a.channel_id, a.enabled, a.priority, a.weight
FROM abilities a
JOIN channels ch ON ch.id = a.channel_id
WHERE ch.name = 'leonardo-minimax-h3-2k';

SELECT jsonb_extract_path_text(value::jsonb, 'cy-sd4-minimax-h3-2k') AS model_price
FROM options WHERE key = 'ModelPrice';
