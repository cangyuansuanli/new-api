-- cy-sd7（Magica）不支持首尾帧：修正 profile、模型描述。
-- cy-origin: docker exec -i newapi-postgres psql -U root -d new-api -v ON_ERROR_STOP=1 < migrate_sd7_no_frame_ssh.sql
-- 随后执行 seed_magica_seedance_api_doc.py 与 sync_enabled_video_api_docs.py 刷新 api_doc。

BEGIN;

UPDATE model_ui_param_profiles
SET params = jsonb_set(params::jsonb, '{frameInputs}', v.frame_inputs::jsonb, true)::text,
    hints = v.hints,
    reference_limits = jsonb_set(
        reference_limits::jsonb,
        '{validationHint}',
        to_jsonb(v.validation_hint),
        true
    )::text,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    (
        'video-tpl-magica-seedance-720p-async',
        '{"enabled":false,"hint":"Magica Seedance 不支持首尾帧"}',
        '[{"text":"固定 720p，按条计费；支持 4–15 秒与多种画幅比例。"},{"text":"支持文生、图生、多参参考（5 图 / 3 视频 / 3 音频）；不支持首尾帧。"}]',
        '固定 720p；最多 5 张参考图、3 段参考视频、3 段参考音频；不支持首尾帧。'
    ),
    (
        'video-tpl-magica-seedance-1080p-async',
        '{"enabled":false,"hint":"Magica Seedance 不支持首尾帧"}',
        '[{"text":"固定 1080p，按条计费；支持 4–15 秒与多种画幅比例。"},{"text":"支持文生、图生、多参参考（5 图 / 3 视频 / 3 音频）；不支持首尾帧。"}]',
        '固定 1080p；最多 5 张参考图、3 段参考视频、3 段参考音频；不支持首尾帧。'
    )
) AS v(profile_id, frame_inputs, hints, validation_hint)
WHERE model_ui_param_profiles.capability = 'video'
  AND model_ui_param_profiles.profile_id = v.profile_id
  AND model_ui_param_profiles.deleted_at IS NULL;

UPDATE models AS m
SET description = v.description,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('cy-sd7-seedance-2.0-720p', 'Seedance 2.0 720p：固定 720p，按条计费；支持文生、图生、多参参考。'),
    ('cy-sd7-seedance-2.0-1080p', 'Seedance 2.0 1080p：固定 1080p，按条计费；支持文生、图生、多参参考。')
) AS v(model_name, description)
WHERE m.model_name = v.model_name
  AND m.deleted_at IS NULL;

COMMIT;

SELECT profile_id, params::jsonb #>> '{frameInputs,enabled}' AS frame_inputs,
       reference_limits::jsonb ->> 'validationHint' AS validation_hint
FROM model_ui_param_profiles
WHERE capability = 'video'
  AND profile_id IN (
      'video-tpl-magica-seedance-720p-async',
      'video-tpl-magica-seedance-1080p-async'
  )
  AND deleted_at IS NULL
ORDER BY profile_id;

SELECT model_name, description FROM models WHERE model_name LIKE 'cy-sd7%' AND deleted_at IS NULL;
