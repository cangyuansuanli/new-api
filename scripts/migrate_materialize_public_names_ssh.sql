-- 将仍依赖 model_channel_prefixes 自动剥离的模型固化为品牌展示名，
-- 并把原中性名写入 model_routing_aliases 以保持 API 兼容和可切换性。
-- 源站: ssh cy-origin 'docker exec -i newapi-postgres psql -U root -d new-api -v ON_ERROR_STOP=1' < migrate_materialize_public_names_ssh.sql

BEGIN;

INSERT INTO model_public_aliases (internal_name, public_name, created_time, updated_time)
VALUES
    ('cy-au1-gemini-music',       'au1-gemini-music',       EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-gv2-grok-video',         'gv2-grok-video',         EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-gv2-grok-video-1.5',     'gv2-grok-video-1.5',     EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-img1-gpt-image-2',       'img1-gpt-image-2',       EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-sd1-omni-fast',          'sd1-omni-fast',          EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-sd1-omni-fast-no-water', 'sd1-omni-fast-no-water', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-sd1-omni-v2v',           'sd1-omni-v2v',           EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-sd1-omni-v2v-no-water',  'sd1-omni-v2v-no-water',  EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-sd1-veo-clean',          'sd1-veo-clean',          EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (internal_name) DO UPDATE SET
    public_name = EXCLUDED.public_name,
    deleted_at = NULL,
    updated_time = EXCLUDED.updated_time;

INSERT INTO model_routing_aliases (public_name, internal_name, note, created_time, updated_time)
VALUES
    ('gemini-music',       'cy-au1-gemini-music',       'Neutral API route -> AU1', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('grok-video',         'cy-gv2-grok-video',         'Neutral API route -> GV2', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('grok-video-1.5',     'cy-gv2-grok-video-1.5',     'Neutral API route -> GV2', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('gpt-image-2',        'cy-img1-gpt-image-2',       'Neutral API route -> IMG1', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('omni-fast',          'cy-sd1-omni-fast',          'Neutral API route -> SD1', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('omni-fast-no-water', 'cy-sd1-omni-fast-no-water', 'Neutral API route -> SD1', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('omni-v2v',           'cy-sd1-omni-v2v',           'Neutral API route -> SD1', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('omni-v2v-no-water',  'cy-sd1-omni-v2v-no-water',  'Neutral API route -> SD1', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('veo-clean',          'cy-sd1-veo-clean',          'Neutral API route -> SD1', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (public_name) DO UPDATE SET
    internal_name = EXCLUDED.internal_name,
    note = EXCLUDED.note,
    deleted_at = NULL,
    updated_time = EXCLUDED.updated_time;

COMMIT;

SELECT internal_name, public_name
FROM model_public_aliases
WHERE deleted_at IS NULL
  AND internal_name IN (
      'cy-au1-gemini-music',
      'cy-gv2-grok-video', 'cy-gv2-grok-video-1.5',
      'cy-img1-gpt-image-2',
      'cy-sd1-omni-fast', 'cy-sd1-omni-fast-no-water',
      'cy-sd1-omni-v2v', 'cy-sd1-omni-v2v-no-water',
      'cy-sd1-veo-clean'
  )
ORDER BY internal_name;
