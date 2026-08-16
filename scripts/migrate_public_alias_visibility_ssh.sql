-- 渠道展示名可单独从模型广场和 /v1/models 隐藏；API 路由与 internal 兼容调用不受影响。
-- 源站: ssh cy-origin 'docker exec -i newapi-postgres psql -U root -d new-api -v ON_ERROR_STOP=1' < migrate_public_alias_visibility_ssh.sql

BEGIN;

ALTER TABLE model_public_aliases
    ADD COLUMN IF NOT EXISTS hidden_from_marketplace BOOLEAN NOT NULL DEFAULT FALSE;

-- 这些模型以中性 routing alias 作为默认客户模型 ID，不额外展示渠道卡片。
UPDATE model_public_aliases
SET hidden_from_marketplace = TRUE,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE deleted_at IS NULL
  AND internal_name IN (
      'cy-au1-gemini-music',
      'cy-gv2-grok-video', 'cy-gv2-grok-video-1.5',
      'cy-img1-gpt-image-2',
      'cy-sd1-omni-fast', 'cy-sd1-omni-fast-no-water',
      'cy-sd1-omni-v2v', 'cy-sd1-omni-v2v-no-water',
      'cy-sd1-veo-clean'
  );

COMMIT;

SELECT internal_name, public_name, hidden_from_marketplace
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
