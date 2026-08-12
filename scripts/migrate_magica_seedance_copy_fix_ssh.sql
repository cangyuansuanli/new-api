-- cy-sd7 对外文案修正：去掉运维向「拉高清/省积分」描述（源站 SSH 执行）
BEGIN;

UPDATE models SET
    description = 'Seedance 2.0 720p：固定 720p，按条计费；支持文生、图生与多参参考。',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE model_name = 'cy-sd7-seedance-2.0-720p' AND deleted_at IS NULL;

UPDATE models SET
    description = 'Seedance 2.0 1080p：固定 1080p，按秒计费；支持文生、图生与多参参考。',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE model_name = 'cy-sd7-seedance-2.0-1080p' AND deleted_at IS NULL;

UPDATE model_ui_param_profiles SET
    reference_limits = '{"images":5,"videos":3,"audios":3,"total":11,"fullReferenceMode":{"label":"多参参考","descriptionWithImages":"最多 5 图、3 视频、3 音频；prompt 可用 @Image1 引用"},"validationHint":"固定 720p；最多 5 张参考图、3 段参考视频、3 段参考音频。","showTempMediaHint":true,"prependReferenceGuide":true}',
    hints = '[{"text":"固定 720p，按条计费；支持 4–15 秒与多种画幅比例。"},{"text":"支持文生、图生与多参参考（5 图 / 3 视频 / 3 音频）。"}]',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video' AND profile_id = 'video-tpl-magica-seedance-720p-async';

UPDATE model_ui_param_profiles SET
    reference_limits = '{"images":5,"videos":3,"audios":3,"total":11,"fullReferenceMode":{"label":"多参参考","descriptionWithImages":"最多 5 图、3 视频、3 音频；prompt 可用 @Image1 引用"},"validationHint":"固定 1080p；最多 5 张参考图、3 段参考视频、3 段参考音频。","showTempMediaHint":true,"prependReferenceGuide":true}',
    hints = '[{"text":"固定 1080p，按秒计费；支持 4–15 秒与多种画幅比例。"},{"text":"支持文生、图生与多参参考（5 图 / 3 视频 / 3 音频）。"}]',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video' AND profile_id = 'video-tpl-magica-seedance-1080p-async';

COMMIT;

SELECT model_name, description FROM models WHERE model_name LIKE 'cy-sd7%' AND deleted_at IS NULL;
