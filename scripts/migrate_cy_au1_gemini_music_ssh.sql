-- cy-au1-gemini-music：OAIREGBox Gemini 音乐（internal 注册名 → public gemini-music → upstream gemini-music）
-- 源站: docker exec -i newapi-postgres psql -U root -d new-api < migrate_cy_au1_gemini_music_ssh.sql

BEGIN;

INSERT INTO model_channel_prefixes (prefix, note, enabled, sort_order, created_time, updated_time)
VALUES ('cy-au1-', 'Gemini 音乐 · 线路 A', TRUE, 127, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (prefix) DO UPDATE SET
  note = EXCLUDED.note,
  enabled = TRUE,
  updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT;

-- 若存在旧 internal 名，统一改为 cy-au1-
UPDATE models SET
  model_name = 'cy-au1-gemini-music',
  description = 'Gemini 音乐生成：异步 POST /v1/audio/generations，按次 ¥0.50/首。',
  tags = 'audio,music,gemini',
  endpoints = '{"openai-audio":{"path":"/v1/audio/generations","method":"POST"}}',
  status = 1,
  updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE deleted_at IS NULL
  AND model_name IN ('oairegbox-gemini-music', 'cy-au1-gemini-music');

INSERT INTO models (
  model_name, description, tags, vendor_id, endpoints, status, sync_official,
  created_time, updated_time
)
SELECT
  'cy-au1-gemini-music',
  'Gemini 音乐生成：异步 POST /v1/audio/generations，按次 ¥0.50/首。',
  'audio,music,gemini',
  12,
  '{"openai-audio":{"path":"/v1/audio/generations","method":"POST"}}',
  1,
  0,
  EXTRACT(EPOCH FROM NOW())::BIGINT,
  EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE NOT EXISTS (
  SELECT 1 FROM models WHERE model_name = 'cy-au1-gemini-music' AND deleted_at IS NULL
);

UPDATE abilities SET model = 'cy-au1-gemini-music'
WHERE model IN ('oairegbox-gemini-music', 'cy-au1-gemini-music');

-- 挂到 OAIREGBox OpenAI 渠道（与 gemini-image / gpt-image 同池）
DO $$
DECLARE
  ch_id INT;
  mapping jsonb;
BEGIN
  SELECT id INTO ch_id
  FROM channels
  WHERE status = 1
    AND deleted_at IS NULL
    AND base_url ILIKE '%oairegbox.cc%'
    AND (
      models ILIKE '%gemini-image%'
      OR models ILIKE '%cy-sd1-gemini-image%'
      OR models ILIKE '%gpt-image%'
      OR models ILIKE '%cy-sd1-gpt-image%'
    )
  ORDER BY id
  LIMIT 1;

  IF ch_id IS NULL THEN
    RAISE EXCEPTION 'no oairegbox image channel found for cy-au1-gemini-music';
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
    )::text
  WHERE id = ch_id;

  DELETE FROM abilities
  WHERE channel_id = ch_id AND model = 'cy-au1-gemini-music';

  INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
  SELECT g.grp, 'cy-au1-gemini-music', ch_id, TRUE, 0, 0
  FROM (VALUES ('AUDIO'), ('全模型-无claude/gpt'), ('gemini-高速')) AS g(grp);
END $$;

COMMIT;

SELECT model_name, status, endpoints FROM models
WHERE model_name = 'cy-au1-gemini-music' AND deleted_at IS NULL;
