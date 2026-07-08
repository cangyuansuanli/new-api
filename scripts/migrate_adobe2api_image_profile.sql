-- Adobe2API 4K image profile: keep UI params aligned with upstream support.
-- Source /opt/adobe currently accepts aspect_ratio + output_resolution and writes PNG.

INSERT INTO model_ui_param_profiles (
    capability, profile_id, api_mode, requires_reference_media,
    poll, reference_limits, params, option_rules, hints,
    created_time, updated_time
) VALUES (
    'image',
    'image-tpl-adobe2api-4k',
    'images-json-async',
    false,
    '{}'::jsonb,
    '{}'::jsonb,
    '{
        "quality": {
            "enabled": true,
            "options": [
                {"value": "auto", "label": "自动"},
                {"value": "high", "label": "4K"},
                {"value": "medium", "label": "2K"},
                {"value": "low", "label": "1K"}
            ]
        },
        "aspectRatio": {
            "enabled": true,
            "options": [
                {"value": "1:1", "label": "1:1", "size": "1024x1024", "width": 1024, "height": 1024, "icon": "square"},
                {"value": "3:2", "label": "3:2", "size": "1536x1024", "width": 1536, "height": 1024, "icon": "landscape"},
                {"value": "2:3", "label": "2:3", "size": "1024x1536", "width": 1024, "height": 1536, "icon": "portrait"},
                {"value": "1:1-2k", "label": "1:1(2K)", "size": "2048x2048", "width": 2048, "height": 2048, "icon": "square"},
                {"value": "16:9-2k", "label": "16:9(2K)", "size": "2048x1152", "width": 2048, "height": 1152, "icon": "landscape"},
                {"value": "9:16-2k", "label": "9:16(2K)", "size": "1152x2048", "width": 1152, "height": 2048, "icon": "portrait"},
                {"value": "16:9-4k", "label": "16:9(4K)", "size": "3840x2160", "width": 3840, "height": 2160, "icon": "landscape"},
                {"value": "9:16-4k", "label": "9:16(4K)", "size": "2160x3840", "width": 2160, "height": 3840, "icon": "portrait"}
            ]
        },
        "customDimensions": {"enabled": false},
        "count": {"enabled": true, "min": 1, "max": 1, "quickCount": 1},
        "background": {"enabled": false},
        "outputFormat": {"enabled": false},
        "outputCompression": {"enabled": false},
        "moderation": {"enabled": false}
    }'::jsonb,
    '[]'::jsonb,
    '[
        {"text": "Adobe2API 上游支持比例与 1K/2K/4K 分辨率档位。"},
        {"text": "输出由上游固定为 PNG；格式、压缩、背景和审核参数不开放。"}
    ]'::jsonb,
    EXTRACT(EPOCH FROM NOW())::bigint,
    EXTRACT(EPOCH FROM NOW())::bigint
)
ON CONFLICT (capability, profile_id) DO UPDATE SET
    api_mode = EXCLUDED.api_mode,
    params = EXCLUDED.params,
    option_rules = EXCLUDED.option_rules,
    hints = EXCLUDED.hints,
    updated_time = EXTRACT(EPOCH FROM NOW())::bigint,
    deleted_at = NULL;

UPDATE models
SET image_profile_id = 'image-tpl-adobe2api-4k',
    updated_time = EXTRACT(EPOCH FROM NOW())::bigint
WHERE model_name = 'cy-img2-gpt-image-2-4k'
  AND deleted_at IS NULL;

SELECT model_name, image_profile_id
FROM models
WHERE model_name = 'cy-img2-gpt-image-2-4k';
