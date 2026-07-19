-- cy-sd2 / 腾达 Seedance 2.0：独立 UI profile（源站 SSH 执行）
-- contabo: docker exec -i newapi-postgres psql -U root -d new-api < migrate_cy_sd2_seedance_profile_ssh.sql
-- 部署 new-api 后执行；seed 数据见 scripts/seed_data/model_ui_params_video.json → video-tpl-cy-sd2-seedance-async

BEGIN;

INSERT INTO model_ui_param_profiles (
    capability, profile_id, match, sort_order, api_mode, payload_builder,
    requires_reference_media, poll, poll_status, reference_limits,
    params, option_rules, hints, created_time, updated_time
) VALUES (
    'video', 'video-tpl-cy-sd2-seedance-async', '["cy-sd2-seedance"]', 82,
    'videos-json-async', 'seedance-flat', FALSE, '{}', NULL,
    '{"images":9,"videos":3,"audios":3,"imageMaxBytes":31457280,"videoMaxBytes":52428800,"audioMaxBytes":15728640,"video":{"minDurationMs":2000,"maxDurationMs":15000,"totalMaxDurationMs":15000,"minWidth":300,"maxWidth":6000,"minHeight":300,"maxHeight":6000,"minAspectRatio":0.4,"maxAspectRatio":2.5},"fullReferenceMode":{"label":"全能参考","descriptionWithImages":"933：图 + 可选视频/音频"},"validationHint":"参考视频需为 mp4/mov，H.264/H.265，FPS 24-60。","showTempMediaHint":true,"prependReferenceGuide":true}',
    '{"resolution":{"enabled":true,"options":[{"value":"480p","label":"480p"},{"value":"720p","label":"720p"}]},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"},{"value":"1:1","label":"方形"},{"value":"21:9","label":"宽银幕"},{"value":"3:4","label":"3:4"},{"value":"4:3","label":"4:3"}]},"duration":{"enabled":true,"numericOptions":[4,5,6,7,8,9,10,11,12,13,14,15],"min":4,"max":15},"generateAudio":{"enabled":false},"watermark":{"enabled":false},"seed":{"enabled":false},"widthHeight":{"enabled":false},"frameInputs":{"enabled":true,"hint":"首尾帧与 933 参考互斥；成对指定 first + last"}}',
    '[]',
    '[{"text":"Seedance 2.0 特惠（cy-sd2）：480p/720p、4–15 秒；933 全能参考与首尾帧；客户端 flat JSON，relay 转 Tengda content[]。"}]',
    EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
)
ON CONFLICT (capability, profile_id) DO UPDATE SET
    match = EXCLUDED.match,
    sort_order = EXCLUDED.sort_order,
    api_mode = EXCLUDED.api_mode,
    payload_builder = EXCLUDED.payload_builder,
    reference_limits = EXCLUDED.reference_limits,
    params = EXCLUDED.params,
    hints = EXCLUDED.hints,
    deleted_at = NULL,
    updated_time = EXCLUDED.updated_time;

UPDATE models SET
    video_profile_id = 'video-tpl-cy-sd2-seedance-async',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE deleted_at IS NULL
  AND (
      model_name LIKE 'cy-sd2-seedance%'
      OR model_name = 'tengd-Seedance-2.0'
  );

COMMIT;

SELECT model_name, video_profile_id
FROM models
WHERE deleted_at IS NULL
  AND (model_name LIKE 'cy-sd2-seedance%' OR model_name = 'tengd-Seedance-2.0')
ORDER BY model_name;
