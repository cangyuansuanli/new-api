-- 对齐 Adobe2API Veo/Kling 视频 Profile 与统一视频入站契约。
-- 生产执行示例：
--   ssh cy-origin 'docker exec -i newapi-postgres psql -v ON_ERROR_STOP=1 -U root -d new-api' \
--     < scripts/migrate_adobe2api_video_profiles_complete_ssh.sql

BEGIN;

UPDATE model_ui_param_profiles AS p
SET reference_limits = v.reference_limits,
    params = v.params,
    hints = v.hints,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT,
    deleted_at = NULL
FROM (VALUES
    (
        'video-tpl-adobe-veo31-json-async',
        '{"images":3,"videos":0,"audios":0,"imageMaxBytes":31457280,"fullReferenceMode":{"label":"参考图或首尾帧","descriptionWithImages":"普通参考图最多 3 张；首尾帧须成对提供"},"validationHint":"普通参考图最多 3 张，或使用成对首尾帧；两种模式互斥。","showTempMediaHint":true}',
        '{"resolution":{"enabled":true,"options":[{"value":"720p","label":"720p"},{"value":"1080p","label":"1080p"}]},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"}]},"duration":{"enabled":true,"min":4,"max":8,"numericOptions":[4,6,8]},"generateAudio":{"enabled":true},"watermark":{"enabled":false},"seed":{"enabled":true},"widthHeight":{"enabled":false},"frameInputs":{"enabled":true,"hint":"首尾帧须成对提供，且与普通参考图互斥"}}',
        '[{"text":"Veo 3.1：支持 4/6/8 秒、720p/1080p、种子、音频；普通参考图最多 3 张，或首尾帧 2 张。"}]'
    ),
    (
        'video-tpl-adobe-veo31-fast-json-async',
        '{"images":0,"videos":0,"audios":0,"imageMaxBytes":31457280,"fullReferenceMode":{"label":"首尾帧","descriptionWithImages":"仅支持成对首尾帧，不支持普通参考图"},"validationHint":"仅支持成对首尾帧参考图；不支持普通参考图。","showTempMediaHint":true}',
        '{"resolution":{"enabled":true,"options":[{"value":"720p","label":"720p"},{"value":"1080p","label":"1080p"}]},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"}]},"duration":{"enabled":true,"min":4,"max":8,"numericOptions":[4,6,8]},"generateAudio":{"enabled":true},"watermark":{"enabled":false},"seed":{"enabled":true},"widthHeight":{"enabled":false},"frameInputs":{"enabled":true,"hint":"仅支持成对首尾帧"}}',
        '[{"text":"Veo 3.1 Fast：仅支持首尾帧 2 张参考图；支持 4/6/8 秒、720p/1080p、种子和音频。"}]'
    ),
    (
        'video-tpl-adobe-kling3-json-async',
        '{"images":0,"videos":0,"audios":0,"imageMaxBytes":31457280,"fullReferenceMode":{"label":"首尾帧","descriptionWithImages":"仅支持成对首尾帧，不支持普通参考图"},"validationHint":"仅支持成对首尾帧参考图；不支持普通参考图。","showTempMediaHint":true}',
        '{"resolution":{"enabled":true,"options":[{"value":"720p","label":"720p"},{"value":"1080p","label":"1080p"}]},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"}]},"duration":{"enabled":true,"min":3,"max":15,"numericOptions":[3,4,5,6,7,8,9,10,11,12,13,14,15]},"generateAudio":{"enabled":true},"watermark":{"enabled":false},"seed":{"enabled":true},"widthHeight":{"enabled":false},"frameInputs":{"enabled":true,"hint":"仅支持成对首尾帧"}}',
        '[{"text":"Kling 3.0：支持 3–15 秒、720p/1080p、种子、音频和首尾帧。"}]'
    ),
    (
        'video-tpl-adobe-kling3-omni-json-async',
        '{"images":0,"videos":0,"audios":0,"imageMaxBytes":31457280,"fullReferenceMode":{"label":"首尾帧","descriptionWithImages":"仅支持成对首尾帧，不支持普通参考图"},"validationHint":"仅支持成对首尾帧；普通参考图会被拒绝。","showTempMediaHint":true}',
        '{"resolution":{"enabled":true,"options":[{"value":"720p","label":"720p"},{"value":"1080p","label":"1080p"}]},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"}]},"duration":{"enabled":true,"min":3,"max":15,"numericOptions":[3,4,5,6,7,8,9,10,11,12,13,14,15]},"generateAudio":{"enabled":true},"watermark":{"enabled":false},"seed":{"enabled":true},"widthHeight":{"enabled":false},"frameInputs":{"enabled":true,"hint":"仅支持成对首尾帧"}}',
        '[{"text":"Kling 3.0 Omni：支持 3–15 秒、720p/1080p、种子、音频和成对首尾帧；不支持普通参考图。"}]'
    )
) AS v(profile_id, reference_limits, params, hints)
WHERE p.capability = 'video'
  AND p.profile_id = v.profile_id;

-- Profiles must be complete even when a model has no active pricing row yet.
UPDATE model_ui_param_profiles
SET params = jsonb_set(
        jsonb_set(
            jsonb_set(params::jsonb, '{watermark}', '{"enabled":false}'::jsonb, true),
            '{widthHeight}', '{"enabled":false}'::jsonb, true
        ),
        '{frameInputs}', '{"enabled":true}'::jsonb, true
    )::text,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video'
  AND profile_id IN (
      'video-tpl-adobe-veo31-json-async',
      'video-tpl-adobe-veo31-fast-json-async',
      'video-tpl-adobe-kling3-json-async',
      'video-tpl-adobe-kling3-omni-json-async'
  )
  AND deleted_at IS NULL;

COMMIT;

SELECT profile_id,
       reference_limits::jsonb AS reference_limits,
       params::jsonb ? 'watermark' AS has_watermark,
       params::jsonb ? 'widthHeight' AS has_width_height,
       params::jsonb #>> '{frameInputs,enabled}' AS frame_inputs,
       params::jsonb #>> '{seed,enabled}' AS seed
FROM model_ui_param_profiles
WHERE capability = 'video'
  AND profile_id IN (
      'video-tpl-adobe-veo31-json-async',
      'video-tpl-adobe-veo31-fast-json-async',
      'video-tpl-adobe-kling3-json-async',
      'video-tpl-adobe-kling3-omni-json-async'
  )
  AND deleted_at IS NULL
ORDER BY profile_id;
