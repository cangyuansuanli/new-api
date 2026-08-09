-- Adobe2API channel 75: replace retired adobe-* video names with the current
-- provider-neutral internal names and standard public/upstream model names.
-- Sora2 is intentionally excluded because it has been retired upstream.

BEGIN;

INSERT INTO model_channel_prefixes (prefix, note, enabled, sort_order, created_time, updated_time)
VALUES ('cy-adobe-', 'Adobe2API 视频线路', TRUE, 131, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (prefix) DO UPDATE SET note = EXCLUDED.note, enabled = TRUE, deleted_at = NULL,
  updated_time = EXCLUDED.updated_time;

INSERT INTO model_ui_param_profiles (capability, profile_id, api_mode, requires_reference_media, poll, poll_status, reference_limits, params, option_rules, hints, created_time, updated_time)
VALUES
('video', 'video-tpl-adobe-veo31-json-async', 'videos-json-async', FALSE, '{}', NULL, '{"images":3,"videos":0,"audios":0}', '{"resolution":{"enabled":true,"options":[{"value":"720p","label":"720p"},{"value":"1080p","label":"1080p"}]},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"}]},"duration":{"enabled":true,"min":4,"max":8,"numericOptions":[4,6,8]},"generateAudio":{"enabled":true},"seed":{"enabled":true}}', '[]', '[{"text":"Veo 3.1：支持 4/6/8 秒、720p/1080p、种子、音频；普通参考图最多 3 张，或首尾帧 2 张。"}]', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
('video', 'video-tpl-adobe-veo31-fast-json-async', 'videos-json-async', FALSE, '{}', NULL, '{"images":2,"videos":0,"audios":0}', '{"resolution":{"enabled":true,"options":[{"value":"720p","label":"720p"},{"value":"1080p","label":"1080p"}]},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"}]},"duration":{"enabled":true,"min":4,"max":8,"numericOptions":[4,6,8]},"generateAudio":{"enabled":true},"seed":{"enabled":true}}', '[]', '[{"text":"Veo 3.1 Fast：仅支持首尾帧 2 张参考图；支持 4/6/8 秒、720p/1080p、种子和音频。"}]', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
('video', 'video-tpl-adobe-kling3-json-async', 'videos-json-async', FALSE, '{}', NULL, '{"images":2,"videos":0,"audios":0}', '{"resolution":{"enabled":true,"options":[{"value":"720p","label":"720p"},{"value":"1080p","label":"1080p"}]},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"}]},"duration":{"enabled":true,"min":3,"max":15},"generateAudio":{"enabled":true},"seed":{"enabled":true}}', '[]', '[{"text":"Kling 3.0：支持 3–15 秒、720p/1080p、种子、音频和首尾帧。"}]', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
('video', 'video-tpl-adobe-kling3-omni-json-async', 'videos-json-async', FALSE, '{}', NULL, '{"images":3,"videos":0,"audios":0}', '{"resolution":{"enabled":true,"options":[{"value":"720p","label":"720p"},{"value":"1080p","label":"1080p"}]},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"}]},"duration":{"enabled":true,"min":3,"max":15},"generateAudio":{"enabled":true},"seed":{"enabled":true}}', '[]', '[{"text":"Kling 3.0 Omni：支持 3–15 秒、720p/1080p、种子、音频、多模态参考和首尾帧。"}]', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
('video', 'video-tpl-adobe-gemini-omni-json-async', 'videos-json-async', FALSE, '{}', NULL, '{"images":4,"videos":0,"audios":0}', '{"resolution":{"enabled":true,"options":[{"value":"720p","label":"720p"}]},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"}]},"duration":{"enabled":true,"min":3,"max":10},"generateAudio":{"enabled":false},"seed":{"enabled":false}}', '[]', '[{"text":"Gemini Omni Flash：支持 3–10 秒、720p、16:9/9:16；单张图可作首帧，参考图组最多 4 张 style 图。"}]', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (capability, profile_id) DO UPDATE SET api_mode = EXCLUDED.api_mode, reference_limits = EXCLUDED.reference_limits, params = EXCLUDED.params, hints = EXCLUDED.hints, deleted_at = NULL, updated_time = EXCLUDED.updated_time;

WITH wanted(model, upstream, profile, description, tags) AS (VALUES
  ('cy-adobe-veo-3.1', 'veo-3.1', 'video-tpl-adobe-veo31-json-async', 'Veo 3.1 视频生成，支持 4/6/8 秒、720p/1080p、三张资产参考或首尾帧。', 'video,veo,adobe,firefly'),
  ('cy-adobe-veo-3.1-fast', 'veo-3.1-fast', 'video-tpl-adobe-veo31-fast-json-async', 'Veo 3.1 Fast 视频生成，仅支持首尾帧，支持 4/6/8 秒与 720p/1080p。', 'video,veo,adobe,firefly,fast'),
  ('cy-adobe-kling-3.0', 'kling-3.0', 'video-tpl-adobe-kling3-json-async', 'Kling 3.0 视频生成，支持 3–15 秒与 720p/1080p。', 'video,kling,adobe,firefly'),
  ('cy-adobe-kling-3.0-omni', 'kling-3.0-omni', 'video-tpl-adobe-kling3-omni-json-async', 'Kling 3.0 Omni 多模态视频生成，支持 3–15 秒与 720p/1080p。', 'video,kling,adobe,firefly,omni'),
  ('cy-adobe-gemini-omni-flash', 'gemini-omni-flash', 'video-tpl-adobe-gemini-omni-json-async', 'Gemini Omni Flash 视频生成，支持 3–10 秒与 720p。', 'video,gemini,adobe,firefly,omni')
)
INSERT INTO model_public_aliases (internal_name, public_name, created_time, updated_time)
SELECT model, upstream, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT FROM wanted
ON CONFLICT (internal_name) DO UPDATE SET public_name = EXCLUDED.public_name, deleted_at = NULL,
  updated_time = EXCLUDED.updated_time;

UPDATE channels SET
  name = 'Adobe2API 视频',
  models = 'cy-adobe-veo-3.1,cy-adobe-veo-3.1-fast,cy-adobe-kling-3.0,cy-adobe-kling-3.0-omni,cy-adobe-gemini-omni-flash,cy-sd5-seedance-2.0,cy-sd5-seedance-2.0-fast',
  model_mapping = '{"cy-adobe-veo-3.1":"veo-3.1","cy-adobe-veo-3.1-fast":"veo-3.1-fast","cy-adobe-kling-3.0":"kling-3.0","cy-adobe-kling-3.0-omni":"kling-3.0-omni","cy-adobe-gemini-omni-flash":"gemini-omni-flash","cy-sd5-seedance-2.0":"seedance-2.0","cy-sd5-seedance-2.0-fast":"seedance-2.0-fast"}',
  status = 1, updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE id = 75;

DELETE FROM abilities WHERE channel_id = 75 AND model IN (
  'adobe-sora2', 'adobe-sora2-pro', 'adobe-veo31', 'adobe-veo31-ref', 'adobe-veo31-fast',
  'cy-adobe-veo-3.1', 'cy-adobe-veo-3.1-fast', 'cy-adobe-kling-3.0',
  'cy-adobe-kling-3.0-omni', 'cy-adobe-gemini-omni-flash',
  'cy-sd5-seedance-2.0', 'cy-sd5-seedance-2.0-fast'
);
INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT g.grp, m.model, 75, TRUE, 0, 90
FROM (VALUES ('VIDEO'), ('全模型-无claude/gpt'), ('对接专用')) g(grp)
CROSS JOIN (VALUES
  ('cy-adobe-veo-3.1'), ('cy-adobe-veo-3.1-fast'), ('cy-adobe-kling-3.0'),
  ('cy-adobe-kling-3.0-omni'), ('cy-adobe-gemini-omni-flash'),
  ('cy-sd5-seedance-2.0'), ('cy-sd5-seedance-2.0-fast')
) m(model);

UPDATE models SET status = 0, updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE model_name IN ('adobe-sora2', 'adobe-sora2-pro', 'adobe-veo31', 'adobe-veo31-ref', 'adobe-veo31-fast')
  AND deleted_at IS NULL;

COMMIT;
