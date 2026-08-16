-- MiniMax3 画布首尾帧 Profile：网页无声音开关，画布隐藏该参数并启用成对首尾帧。
-- cy-origin: docker exec -i newapi-postgres psql -U root -d new-api -v ON_ERROR_STOP=1 < migrate_minimax3_canvas_frame_profile_ssh.sql

BEGIN;

UPDATE model_ui_param_profiles
SET params = jsonb_set(
        jsonb_set(params::jsonb, '{generateAudio}', '{"enabled":false}'::jsonb, true),
        '{frameInputs}',
        '{"enabled":true,"hint":"首尾帧与多模态（参考图/视频/音频）二选一"}'::jsonb,
        true
    )::text,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video'
  AND profile_id = 'video-tpl-minimax-h3-2k-async'
  AND deleted_at IS NULL;

COMMIT;

SELECT profile_id,
       params::jsonb #>> '{frameInputs,enabled}' AS frame_inputs_enabled,
       params::jsonb #>> '{generateAudio,enabled}' AS generate_audio_control
FROM model_ui_param_profiles
WHERE capability = 'video'
  AND profile_id = 'video-tpl-minimax-h3-2k-async'
  AND deleted_at IS NULL;
