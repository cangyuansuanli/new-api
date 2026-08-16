-- 将仍依赖 model_channel_prefixes 自动剥离的公开名固化为显式展示名。
-- 执行后可安全移除前缀剥离逻辑；客户可见模型名保持不变。
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
