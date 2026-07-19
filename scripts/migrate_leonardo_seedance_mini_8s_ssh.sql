-- Leonardo Seedance 2.0 Mini 8s 商品（1300 积分号池）
-- 仅允许客户请求 4–8 秒；上游映射仍为 seedance-2.0-mini。
-- contabo: docker exec -i newapi-postgres psql -U root -d new-api < migrate_leonardo_seedance_mini_8s_ssh.sql

BEGIN;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM model_public_aliases
        WHERE public_name = 'seedance-2.0-mini-8s'
          AND internal_name <> 'cy-sd4-seedance-2.0-mini-8s'
    ) THEN
        RAISE EXCEPTION 'public model alias seedance-2.0-mini-8s is already used by another model';
    END IF;
END $$;

-- 专属 UI profile：客户只能看到并选择 4/5/6/7/8 秒。
INSERT INTO model_ui_param_profiles (
    capability, profile_id, api_mode, payload_builder, validation_key,
    requires_reference_media, poll_status, poll, reference_limits,
    params, option_rules, hints, note, created_time, updated_time
) VALUES (
    'video',
    'video-tpl-seedance-mini-8s-async',
    'videos-json-async',
    'seedance-flat',
    'seedance-mini-8s',
    FALSE,
    '',
    '{}',
    '{
      "images": 4,
      "videos": 3,
      "audios": 1,
      "imageMaxBytes": 31457280,
      "videoMaxBytes": 52428800,
      "audioMaxBytes": 15728640,
      "video": {
        "minDurationMs": 4000,
        "maxDurationMs": 8000,
        "totalMaxDurationMs": 8000,
        "minWidth": 720,
        "maxWidth": 2160,
        "minHeight": 720,
        "maxHeight": 2160
      },
      "audio": { "maxDurationMs": 8000 },
      "fullReferenceMode": {
        "label": "多模态",
        "descriptionWithImages": "多模态：图 + 可选视频/音频"
      },
      "validationHint": "参考视频 mp4/mov，单条 4–8 秒、最多 3 条总时长 ≤8 秒；参考音频 ≤8 秒；素材宽高 720–2160px。",
      "showTempMediaHint": true,
      "prependReferenceGuide": true
    }',
    '{
      "resolution": {
        "enabled": true,
        "options": [
          {"value": "480p", "label": "标准 480p"},
          {"value": "720p", "label": "HD 720p"}
        ]
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
        "numericOptions": [4, 5, 6, 7, 8],
        "min": 4,
        "max": 8
      },
      "generateAudio": {"enabled": true, "hint": "是否生成原生音频，默认开启"},
      "watermark": {"enabled": false},
      "seed": {"enabled": false},
      "widthHeight": {"enabled": false},
      "frameInputs": {
        "enabled": true,
        "hint": "首尾帧与多模态（参考图/视频/音频）二选一；成对指定 first + last"
      }
    }',
    '[]',
    '[
      {"text": "Seedance 2.0 Mini 8 秒特惠：标准 480p / HD 720p；输出时长 4–8 秒。"},
      {"text": "多模态 4 图 / 3 视频（总时长 ≤8 秒）/ 1 音频（≤8 秒）；与首尾帧互斥。"}
    ]',
    'Mini 8 秒特惠专用 8 秒产品；按次计费，失败不计费。',
    EXTRACT(EPOCH FROM NOW())::BIGINT,
    EXTRACT(EPOCH FROM NOW())::BIGINT
)
ON CONFLICT (capability, profile_id) DO UPDATE SET
    api_mode = EXCLUDED.api_mode,
    payload_builder = EXCLUDED.payload_builder,
    validation_key = EXCLUDED.validation_key,
    requires_reference_media = EXCLUDED.requires_reference_media,
    poll_status = EXCLUDED.poll_status,
    poll = EXCLUDED.poll,
    reference_limits = EXCLUDED.reference_limits,
    params = EXCLUDED.params,
    option_rules = EXCLUDED.option_rules,
    hints = EXCLUDED.hints,
    note = EXCLUDED.note,
    updated_time = EXCLUDED.updated_time;

-- 新商品模型元数据。
INSERT INTO models (
    model_name, description, tags, vendor_id, endpoints, status, sync_official,
    video_profile_id, api_doc, created_time, updated_time
) SELECT
    'cy-sd4-seedance-2.0-mini-8s',
    'Seedance 2.0 Mini 8 秒特惠。480p / 720p，支持 4–8 秒。',
    'video,seedance,subscription,mini,8s',
    4,
    '{"openai-video":{"path":"/v1/videos","method":"POST"}}',
    1,
    0,
    'video-tpl-seedance-mini-8s-async',
    jsonb_build_object(
        'dispatch_mode', 'async',
        'intro', 'Seedance 2.0 Mini 8 秒特惠视频。模型：{{model}}。按条计费，失败不计费。',
        'endpoints', jsonb_build_array(
            jsonb_build_object('method', 'POST', 'path', '{{base}}/videos'),
            jsonb_build_object('method', 'GET', 'path', '{{base}}/videos/{task_id}'),
            jsonb_build_object('method', 'GET', 'path', '{{base}}/videos/{task_id}/content')
        ),
        'params', jsonb_build_array(
            jsonb_build_object('name', 'model', 'description', '传 public 名 seedance-2.0-mini-8s。'),
            jsonb_build_object('name', 'prompt', 'description', '必填，视频描述。'),
            jsonb_build_object('name', 'duration', 'description', '时长 4–8 秒整数，省略时按上游默认值处理。'),
            jsonb_build_object('name', 'resolution', 'description', '480p 或 720p。'),
            jsonb_build_object('name', 'aspect_ratio', 'description', '16:9、9:16、1:1、21:9、3:4 或 4:3。')
        ),
        'basic_request_json', jsonb_build_object(
            'model', 'seedance-2.0-mini-8s',
            'prompt', '雨夜霓虹街道，镜头缓慢推进，电影感光影',
            'aspect_ratio', '16:9',
            'duration', 8,
            'resolution', '720p'
        )
    )::text,
    EXTRACT(EPOCH FROM NOW())::BIGINT,
    EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE NOT EXISTS (
    SELECT 1
    FROM models m
    WHERE m.model_name = 'cy-sd4-seedance-2.0-mini-8s'
      AND m.deleted_at IS NULL
);

UPDATE models
SET description = 'Seedance 2.0 Mini 8 秒特惠。480p / 720p，支持 4–8 秒。',
    tags = 'video,seedance,subscription,mini,8s',
    vendor_id = 4,
    endpoints = '{"openai-video":{"path":"/v1/videos","method":"POST"}}',
    status = 1,
    sync_official = 0,
    video_profile_id = 'video-tpl-seedance-mini-8s-async',
    api_doc = jsonb_build_object(
        'dispatch_mode', 'async',
        'intro', 'Seedance 2.0 Mini 8 秒特惠视频。模型：{{model}}。按条计费，失败不计费。',
        'params', jsonb_build_array(
            jsonb_build_object('name', 'model', 'description', '传 public 名 seedance-2.0-mini-8s。'),
            jsonb_build_object('name', 'prompt', 'description', '必填，视频描述。'),
            jsonb_build_object('name', 'duration', 'description', '时长 4–8 秒整数。'),
            jsonb_build_object('name', 'resolution', 'description', '480p 或 720p。')
        ),
        'basic_request_json', jsonb_build_object(
            'model', 'seedance-2.0-mini-8s',
            'prompt', '雨夜霓虹街道，镜头缓慢推进，电影感光影',
            'aspect_ratio', '16:9',
            'duration', 8,
            'resolution', '720p'
        )
    )::text,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE model_name = 'cy-sd4-seedance-2.0-mini-8s'
  AND deleted_at IS NULL;

INSERT INTO model_public_aliases (internal_name, public_name, created_time, updated_time)
VALUES (
    'cy-sd4-seedance-2.0-mini-8s',
    'seedance-2.0-mini-8s',
    EXTRACT(EPOCH FROM NOW())::BIGINT,
    EXTRACT(EPOCH FROM NOW())::BIGINT
)
ON CONFLICT (internal_name) DO UPDATE SET
    public_name = EXCLUDED.public_name,
    updated_time = EXCLUDED.updated_time,
    deleted_at = NULL;

-- 三个 Leonardo 实例全部加入映射；渠道状态/优先级沿用现有配置。
UPDATE channels
SET models = CASE
        WHEN POSITION('cy-sd4-seedance-2.0-mini-8s' IN models) > 0 THEN models
        WHEN COALESCE(models, '') = '' THEN 'cy-sd4-seedance-2.0-mini-8s'
        ELSE models || ',cy-sd4-seedance-2.0-mini-8s'
    END,
    model_mapping = jsonb_set(
        COALESCE(NULLIF(model_mapping, '')::jsonb, '{}'::jsonb),
        '{cy-sd4-seedance-2.0-mini-8s}',
        '"seedance-2.0-mini"'::jsonb,
        true
    )::text
WHERE id IN (67, 82, 83);

DELETE FROM abilities
WHERE model = 'cy-sd4-seedance-2.0-mini-8s'
  AND channel_id IN (67, 82, 83);

INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
VALUES
    ('VIDEO', 'cy-sd4-seedance-2.0-mini-8s', 67, true, 0, 90),
    ('全模型-无claude/gpt', 'cy-sd4-seedance-2.0-mini-8s', 67, true, 0, 90),
    ('VIDEO', 'cy-sd4-seedance-2.0-mini-8s', 82, true, 100, 100),
    ('全模型-无claude/gpt', 'cy-sd4-seedance-2.0-mini-8s', 82, true, 100, 100),
    ('VIDEO', 'cy-sd4-seedance-2.0-mini-8s', 83, false, 120, 100),
    ('全模型-无claude/gpt', 'cy-sd4-seedance-2.0-mini-8s', 83, false, 120, 100);

-- 按次计费，沿用现有 mini 商品价格 ¥1.90/次（ModelPrice 存 USD）。
INSERT INTO options (key, value)
VALUES
    ('ModelPrice', jsonb_build_object('cy-sd4-seedance-2.0-mini-8s', 1.9)::text),
    ('billing_setting.billing_mode', jsonb_build_object('cy-sd4-seedance-2.0-mini-8s', 'per_request')::text),
    ('billing_setting.request_unit', jsonb_build_object('cy-sd4-seedance-2.0-mini-8s', 'generation')::text)
ON CONFLICT (key) DO UPDATE SET
    value = CASE
        WHEN options.key = 'ModelPrice' THEN
            jsonb_set(options.value::jsonb, '{cy-sd4-seedance-2.0-mini-8s}', '1.9'::jsonb, true)::text
        WHEN options.key = 'billing_setting.billing_mode' THEN
            jsonb_set(options.value::jsonb, '{cy-sd4-seedance-2.0-mini-8s}', '"per_request"'::jsonb, true)::text
        ELSE
            jsonb_set(options.value::jsonb, '{cy-sd4-seedance-2.0-mini-8s}', '"generation"'::jsonb, true)::text
    END;

COMMIT;

SELECT model_name, video_profile_id, description
FROM models
WHERE model_name = 'cy-sd4-seedance-2.0-mini-8s' AND deleted_at IS NULL;

SELECT internal_name, public_name
FROM model_public_aliases
WHERE internal_name = 'cy-sd4-seedance-2.0-mini-8s';

SELECT id, models, model_mapping, status, priority
FROM channels
WHERE id IN (67, 82, 83)
ORDER BY id;
