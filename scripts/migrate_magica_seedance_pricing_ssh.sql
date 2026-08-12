-- cy-sd7 定价调整：720p ¥3.9/条，1080p ¥4.9/条（均按条计费）
BEGIN;

UPDATE models SET
    description = 'Seedance 2.0 720p：固定 720p，按条计费（¥3.9/条）；支持文生、图生与多参参考。',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE model_name = 'cy-sd7-seedance-2.0-720p' AND deleted_at IS NULL;

UPDATE models SET
    description = 'Seedance 2.0 1080p：固定 1080p，按条计费（¥4.9/条）；支持文生、图生与多参参考。',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE model_name = 'cy-sd7-seedance-2.0-1080p' AND deleted_at IS NULL;

UPDATE model_ui_param_profiles SET
    hints = '[{"text":"固定 720p，按条计费（¥3.9/条）；支持 4–15 秒与多种画幅比例。"},{"text":"支持文生、图生与多参参考（5 图 / 3 视频 / 3 音频）。"}]',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video' AND profile_id = 'video-tpl-magica-seedance-720p-async';

UPDATE model_ui_param_profiles SET
    hints = '[{"text":"固定 1080p，按条计费（¥4.9/条）；支持 4–15 秒与多种画幅比例。"},{"text":"支持文生、图生与多参参考（5 图 / 3 视频 / 3 音频）。"}]',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video' AND profile_id = 'video-tpl-magica-seedance-1080p-async';

COMMIT;

-- api_doc + ModelPrice + billing_mode 由 seed_magica_seedance_api_doc.py 写入
