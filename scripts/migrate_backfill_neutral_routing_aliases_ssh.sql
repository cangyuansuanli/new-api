-- 为品牌化模型广场卡片补齐中性 API 入站路由。
-- public alias 负责展示；routing alias 负责客户调用和后续渠道切换。
-- 源站: ssh cy-origin 'docker exec -i newapi-postgres psql -U root -d new-api -v ON_ERROR_STOP=1' < migrate_backfill_neutral_routing_aliases_ssh.sql

BEGIN;

-- gpt-image-2 是客户入站路由；IMG1 卡片保留独立品牌展示名。
UPDATE model_public_aliases
SET public_name = 'img1-gpt-image-2',
    deleted_at = NULL,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE internal_name = 'cy-img1-gpt-image-2';

INSERT INTO model_routing_aliases (public_name, internal_name, note, created_time, updated_time)
VALUES
    ('gemini-music',          'cy-au1-gemini-music',          'Neutral API route -> AU1', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('grok-video',            'cy-gv2-grok-video',            'Neutral API route -> GV2', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('grok-video-1.5',        'cy-gv2-grok-video-1.5',        'Neutral API route -> GV2', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('gpt-image-2',           'cy-img1-gpt-image-2',          'Neutral API route -> IMG1', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('omni-fast',             'cy-sd1-omni-fast',             'Neutral API route -> SD1', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('omni-fast-no-water',    'cy-sd1-omni-fast-no-water',    'Neutral API route -> SD1', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('omni-v2v',              'cy-sd1-omni-v2v',              'Neutral API route -> SD1', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('omni-v2v-no-water',     'cy-sd1-omni-v2v-no-water',     'Neutral API route -> SD1', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('veo-clean',             'cy-sd1-veo-clean',             'Neutral API route -> SD1', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('seedance-2.0-mini',      'cy-sd4-seedance-2.0-mini',     'Neutral API route -> SD4 Leonardo', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('seedance-2.0-720p',      'cy-sd7-seedance-2.0-720p',     'Neutral API route -> SD7 Magica', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('seedance-2.0-1080p',     'cy-sd7-seedance-2.0-1080p',    'Neutral API route -> SD7 Magica', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (public_name) DO UPDATE SET
    internal_name = EXCLUDED.internal_name,
    note = EXCLUDED.note,
    deleted_at = NULL,
    updated_time = EXCLUDED.updated_time;

COMMIT;

SELECT public_name, internal_name, note
FROM model_routing_aliases
WHERE deleted_at IS NULL
  AND public_name IN (
      'gemini-music',
      'grok-video', 'grok-video-1.5',
      'gpt-image-2',
      'omni-fast', 'omni-fast-no-water',
      'omni-v2v', 'omni-v2v-no-water',
      'veo-clean',
      'seedance-2.0-mini', 'seedance-2.0-720p', 'seedance-2.0-1080p'
  )
ORDER BY public_name;
