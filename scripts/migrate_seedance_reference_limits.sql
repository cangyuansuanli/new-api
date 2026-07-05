-- Seedance 校验规则迁入 reference_limits；移除专用 validation_key。

UPDATE model_ui_param_profiles
SET validation_key = '',
    reference_limits = '{
  "images": 9,
  "videos": 3,
  "audios": 3,
  "imageMaxBytes": 31457280,
  "videoMaxBytes": 52428800,
  "audioMaxBytes": 15728640,
  "video": {
    "minDurationMs": 2000,
    "maxDurationMs": 15000,
    "totalMaxDurationMs": 15000,
    "minWidth": 300,
    "maxWidth": 6000,
    "minHeight": 300,
    "maxHeight": 6000,
    "minAspectRatio": 0.4,
    "maxAspectRatio": 2.5
  },
  "fullReferenceMode": {
    "label": "全能参考",
    "descriptionWithImages": "933：图 + 可选视频/音频"
  },
  "validationHint": "参考视频需为 mp4/mov，H.264/H.265，FPS 24-60。",
  "showTempMediaHint": true,
  "prependReferenceGuide": true
}'::text,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video'
  AND profile_id IN (
    'video-tpl-seedance-480p-async',
    'video-tpl-seedance-720p-async',
    'video-tpl-seedance-1080p-async',
    'video-tpl-seedance-4k-async'
  );

UPDATE model_ui_param_profiles
SET validation_key = '',
    reference_limits = '{
  "images": 4,
  "videos": 3,
  "audios": 1,
  "imageMaxBytes": 31457280,
  "videoMaxBytes": 52428800,
  "audioMaxBytes": 15728640,
  "video": {
    "minDurationMs": 4000,
    "maxDurationMs": 15000,
    "totalMaxDurationMs": 15000,
    "minWidth": 720,
    "maxWidth": 2160,
    "minHeight": 720,
    "maxHeight": 2160
  },
  "audio": {
    "maxDurationMs": 15000
  },
  "fullReferenceMode": {
    "label": "多模态",
    "descriptionWithImages": "多模态：图 + 可选视频/音频"
  },
  "validationHint": "参考视频 mp4/mov，单条 4–15 秒、最多 3 条总时长 ≤15 秒；参考音频 ≤15 秒；素材宽高 720–2160px。",
  "showTempMediaHint": true,
  "prependReferenceGuide": true
}'::text,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video'
  AND profile_id = 'video-tpl-cy-sd4-seedance-async';
