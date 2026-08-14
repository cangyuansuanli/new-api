-- cy-sd8 产品/定价/profile 修正（源站已跑过 migrate_sd8_seedance_ssh.sql 时执行）
BEGIN;

UPDATE models AS m SET
    description = v.description,
    video_profile_id = v.profile_id,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('cy-sd8-seedance-2.0', 'Seedance 2.0 卡脸版：¥2.9/条；9 图 + 3 视频 + 3 音频；含人物参考图须遮眼。', 'video-tpl-sd8-seedance-facepass-async'),
    ('cy-sd8-seedance-2.0-fast', 'Seedance 2.0 快速版：¥1.9/条；仅支持最多 9 张参考图。', 'video-tpl-sd8-seedance-fast-async')
) AS v(model_name, description, profile_id)
WHERE m.model_name = v.model_name AND m.deleted_at IS NULL;

COMMIT;

SELECT model_name, description, video_profile_id FROM models WHERE model_name LIKE 'cy-sd8%' AND deleted_at IS NULL;
