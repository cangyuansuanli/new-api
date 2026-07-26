-- Leonardo Seedance（cy-sd4）：移除参考视频本地像素上限（maxWidth/maxHeight/minWidth/minHeight）。
-- 参考视频分辨率改由 leonardo-web2api 上传后 Leonardo GetUploadedMediaById / statusReason 判定。
-- contabo: docker exec -i newapi-postgres psql -U root -d new-api < migrate_leonardo_seedance_remove_video_dimension_precheck_ssh.sql

BEGIN;

UPDATE model_ui_param_profiles
SET reference_limits = '{
  "images": 4,
  "videos": 3,
  "audios": 1,
  "imageMaxBytes": 31457280,
  "videoMaxBytes": 52428800,
  "audioMaxBytes": 15728640,
  "video": {
    "minDurationMs": 4000,
    "maxDurationMs": 15000,
    "totalMaxDurationMs": 15000
  },
  "audio": {
    "maxDurationMs": 15000
  },
  "fullReferenceMode": {
    "label": "多模态",
    "descriptionWithImages": "多模态：图 + 可选视频/音频"
  },
  "validationHint": "参考视频 mp4/mov，单条 4–15 秒、最多 3 条总时长 ≤15 秒；参考音频 ≤15 秒；参考视频像素上限由 Leonardo 上游判定（Seedance 2.0 官方 API 未写宽高限制）。",
  "showTempMediaHint": true,
  "prependReferenceGuide": true
}'::text,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video'
  AND profile_id IN (
    'video-tpl-cy-sd4-seedance-async',
    'video-tpl-seedance-subscription-async'
  );

UPDATE model_ui_param_profiles
SET reference_limits = jsonb_set(
        jsonb_set(
            reference_limits::jsonb,
            '{video}',
            (reference_limits::jsonb -> 'video') - 'minWidth' - 'maxWidth' - 'minHeight' - 'maxHeight',
            true
        ),
        '{validationHint}',
        '"参考视频 mp4/mov，单条 4–8 秒、最多 3 条总时长 ≤8 秒；参考音频 ≤8 秒；参考视频像素上限由 Leonardo 上游判定。"'::jsonb,
        true
    )::text,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video'
  AND profile_id = 'video-tpl-seedance-mini-8s-async';

COMMIT;

SELECT profile_id, reference_limits
FROM model_ui_param_profiles
WHERE capability = 'video'
  AND profile_id IN (
    'video-tpl-cy-sd4-seedance-async',
    'video-tpl-seedance-subscription-async',
    'video-tpl-seedance-mini-8s-async'
  )
ORDER BY profile_id;
