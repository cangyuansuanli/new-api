-- cy-au1-gemini-music：OAIREGBox Gemini 音乐（internal 注册名 → public gemini-music → upstream gemini-music）
-- 源站: docker exec -i newapi-postgres psql -U root -d new-api < migrate_cy_au1_gemini_music_ssh.sql

BEGIN;

ALTER TABLE models ADD COLUMN IF NOT EXISTS audio_profile_id varchar(128);

INSERT INTO model_channel_prefixes (prefix, note, enabled, sort_order, created_time, updated_time)
VALUES ('cy-au1-', 'Gemini 音乐 · 线路 A', TRUE, 127, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (prefix) DO UPDATE SET
  note = EXCLUDED.note,
  enabled = TRUE,
  updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT;

-- 若存在旧 internal 名，统一改为 cy-au1-
UPDATE models SET
  model_name = 'cy-au1-gemini-music',
  description = 'Gemini 音乐生成：异步 POST /v1/audio/generations，按次 ¥0.99/条。',
  tags = 'audio,music,gemini',
  endpoints = '{"openai-audio":{"path":"/v1/audio/generations","method":"POST"}}',
  audio_profile_id = 'audio-tpl-gemini-music',
  status = 1,
  updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE deleted_at IS NULL
  AND model_name IN ('oairegbox-gemini-music', 'cy-au1-gemini-music');

INSERT INTO models (
  model_name, description, tags, vendor_id, endpoints, audio_profile_id, status, sync_official,
  created_time, updated_time
)
SELECT
  'cy-au1-gemini-music',
  'Gemini 音乐生成：异步 POST /v1/audio/generations，按次 ¥0.99/条。',
  'audio,music,gemini',
  12,
  '{"openai-audio":{"path":"/v1/audio/generations","method":"POST"}}',
  'audio-tpl-gemini-music',
  1,
  0,
  EXTRACT(EPOCH FROM NOW())::BIGINT,
  EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE NOT EXISTS (
  SELECT 1 FROM models WHERE model_name = 'cy-au1-gemini-music' AND deleted_at IS NULL
);

UPDATE abilities SET model = 'cy-au1-gemini-music'
WHERE model IN ('oairegbox-gemini-music', 'cy-au1-gemini-music');

-- UI profile registry + profile（audio capability）
INSERT INTO model_ui_param_registries (capability, default_profile_id, poll_defaults, updated_time)
VALUES (
  'audio',
  'audio-tpl-gemini-music',
  '{"audio-json-async":{"delayMs":5000,"maxAttempts":24}}',
  EXTRACT(EPOCH FROM NOW())::BIGINT
)
ON CONFLICT (capability) DO UPDATE SET
  default_profile_id = EXCLUDED.default_profile_id,
  poll_defaults = EXCLUDED.poll_defaults,
  updated_time = EXCLUDED.updated_time;

INSERT INTO model_ui_param_profiles (
  capability, profile_id, api_mode, params, poll, hints, option_rules, reference_limits, wire_config,
  requires_reference_media, created_time, updated_time
)
SELECT
  'audio',
  'audio-tpl-gemini-music',
  'audio-json-async',
  '{}',
  '{"delayMs":5000,"maxAttempts":24}',
  '[{"text":"音乐生成为异步任务，提交后自动轮询，完成后从 URL 下载。"}]',
  '[]',
  '{}',
  '{}',
  FALSE,
  EXTRACT(EPOCH FROM NOW())::BIGINT,
  EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE NOT EXISTS (
  SELECT 1 FROM model_ui_param_profiles
  WHERE capability = 'audio' AND profile_id = 'audio-tpl-gemini-music'
);

-- 挂到已有 OAIREGBox 渠道（优先已含 cy-au1-gemini-music 的行，如 #49 Gemini-omni）
DO $$
DECLARE
  ch_id INT;
BEGIN
  SELECT id INTO ch_id
  FROM channels
  WHERE status = 1
    AND base_url ILIKE '%oairegbox%'
    AND models ILIKE '%cy-au1-gemini-music%'
  ORDER BY id
  LIMIT 1;

  IF ch_id IS NULL THEN
    SELECT id INTO ch_id
    FROM channels
    WHERE status = 1
      AND base_url ILIKE '%oairegbox%'
      AND (models ILIKE '%gemini-image%' OR models ILIKE '%cy-sd1-gemini-image%' OR models ILIKE '%gpt-image%')
    ORDER BY id
    LIMIT 1;
  END IF;

  IF ch_id IS NULL THEN
    RAISE EXCEPTION 'no oairegbox channel found for cy-au1-gemini-music';
  END IF;

  UPDATE channels SET
    models = CASE
      WHEN models IS NULL OR btrim(models) = '' THEN 'cy-au1-gemini-music'
      WHEN models ILIKE '%cy-au1-gemini-music%' THEN models
      ELSE models || ',cy-au1-gemini-music'
    END,
    model_mapping = (
      COALESCE(NULLIF(btrim(model_mapping), '')::jsonb, '{}'::jsonb)
      || jsonb_build_object('cy-au1-gemini-music', 'gemini-music')
    )::text,
    "group" = CASE
      WHEN "group" IS NULL OR btrim("group") = '' THEN 'AUDIO'
      WHEN "group" ILIKE '%AUDIO%' THEN "group"
      ELSE "group" || ',AUDIO'
    END
  WHERE id = ch_id;

  DELETE FROM abilities
  WHERE channel_id = ch_id AND model = 'cy-au1-gemini-music';

  INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
  SELECT g.grp, 'cy-au1-gemini-music', ch_id, TRUE, 0, 0
  FROM (VALUES ('AUDIO'), ('全模型-无claude/gpt'), ('gemini-高速'), ('VIDEO')) AS g(grp)
  WHERE NOT EXISTS (
    SELECT 1 FROM abilities a
    WHERE a.channel_id = ch_id AND a.model = 'cy-au1-gemini-music' AND a."group" = g.grp
  );
END $$;

COMMIT;

SELECT model_name, status, endpoints, audio_profile_id FROM models
WHERE model_name = 'cy-au1-gemini-music' AND deleted_at IS NULL;
