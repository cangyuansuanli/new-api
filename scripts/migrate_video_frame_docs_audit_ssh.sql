-- 补齐 SD7 / SD8 Seedance 首尾帧能力、Profile 提示与模型描述。
-- cy-origin: docker exec -i newapi-postgres psql -U root -d new-api -v ON_ERROR_STOP=1 < migrate_video_frame_docs_audit_ssh.sql

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
        '{"enabled":true,"hint":"首尾帧须成对提供，且与参考图、参考视频、参考音频互斥"}',
        '[{"text":"固定 720p，按条计费；支持 4–15 秒与多种画幅比例。"},{"text":"支持文生、图生、多参参考（5 图 / 3 视频 / 3 音频）及成对首尾帧；两种参考模式互斥。"}]',
        '固定 720p；最多 5 张参考图、3 段参考视频、3 段参考音频；首尾帧须成对提供，且与普通参考素材互斥。'
    ),
    (
        'video-tpl-magica-seedance-1080p-async',
        '{"enabled":true,"hint":"首尾帧须成对提供，且与参考图、参考视频、参考音频互斥"}',
        '[{"text":"固定 1080p，按条计费；支持 4–15 秒与多种画幅比例。"},{"text":"支持文生、图生、多参参考（5 图 / 3 视频 / 3 音频）及成对首尾帧；两种参考模式互斥。"}]',
        '固定 1080p；最多 5 张参考图、3 段参考视频、3 段参考音频；首尾帧须成对提供，且与普通参考素材互斥。'
    ),
    (
        'video-tpl-sd8-seedance-facepass-async',
        '{"enabled":true,"hint":"首尾帧须成对提供，且与参考图、参考视频、参考音频互斥"}',
        '[{"text":"¥2.9/条；时长仅支持 5、10、15 秒。"},{"text":"卡脸线路：含人物参考图须遮眼（贴纸/马赛克遮挡眼部）后再上传。"},{"text":"支持最多 9 图、3 视频、3 音频及成对首尾帧；首尾帧与普通参考素材互斥。"}]',
        '最多 9 张参考图、3 段参考视频、3 段参考音频；含人物参考图须遮眼；首尾帧须成对提供，且与普通参考素材互斥。'
    ),
    (
        'video-tpl-sd8-seedance-fast-async',
        '{"enabled":true,"hint":"首尾帧须成对提供，且与普通参考图互斥"}',
        '[{"text":"¥1.9/条；时长仅支持 5、10、15 秒。"},{"text":"支持最多 9 张普通参考图或成对首尾帧；不支持参考视频、参考音频。"}]',
        '支持最多 9 张普通参考图或成对首尾帧，两种模式互斥；不支持参考视频、参考音频。'
    )
) AS v(profile_id, frame_inputs, hints, validation_hint)
WHERE model_ui_param_profiles.capability = 'video'
  AND model_ui_param_profiles.profile_id = v.profile_id
  AND model_ui_param_profiles.deleted_at IS NULL;

UPDATE models AS m
SET description = v.description,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('cy-sd7-seedance-2.0-720p', 'Seedance 2.0 720p：固定 720p，按条计费；支持文生、图生、多参参考与成对首尾帧。'),
    ('cy-sd7-seedance-2.0-1080p', 'Seedance 2.0 1080p：固定 1080p，按条计费；支持文生、图生、多参参考与成对首尾帧。'),
    ('cy-sd8-seedance-2.0', 'Seedance 2.0 卡脸版：支持最多 9 图、3 视频、3 音频及成对首尾帧；含人物参考图须遮眼。'),
    ('cy-sd8-seedance-2.0-fast', 'Seedance 2.0 快速版：支持最多 9 张普通参考图或成对首尾帧。')
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
      'video-tpl-magica-seedance-1080p-async',
      'video-tpl-sd8-seedance-facepass-async',
      'video-tpl-sd8-seedance-fast-async'
  )
  AND deleted_at IS NULL
ORDER BY profile_id;
