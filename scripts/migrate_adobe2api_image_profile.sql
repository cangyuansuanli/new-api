-- Adobe2API fixed 4K image profile. Resolution is encoded in the model SKU;
-- the UI only selects aspect_ratio and the adapter always emits image_size=4K.

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
            "enabled": false
        },
        "aspectRatio": {
            "enabled": true,
            "options": [
                {"value": "1:1", "label": "1:1", "width": 4096, "height": 4096, "icon": "square"},
                {"value": "4:3", "label": "4:3", "width": 4096, "height": 3072, "icon": "landscape"},
                {"value": "3:4", "label": "3:4", "width": 3072, "height": 4096, "icon": "portrait"},
                {"value": "16:9", "label": "16:9", "width": 3840, "height": 2160, "icon": "landscape"},
                {"value": "9:16", "label": "9:16", "width": 2160, "height": 3840, "icon": "portrait"}
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
        {"text": "Adobe2API 4K 固定档位；分辨率由模型 SKU 决定。"},
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
WHERE model_name LIKE 'adobe-firefly-%-4k'
  AND deleted_at IS NULL;

SELECT model_name, image_profile_id
FROM models
WHERE model_name LIKE 'adobe-firefly-%-4k';
