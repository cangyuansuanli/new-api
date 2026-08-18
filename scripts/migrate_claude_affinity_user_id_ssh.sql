-- 将 Claude 渠道亲和从客户端 metadata.user_id 优先切换为网关认证用户 ID。
-- 源站执行：
-- ssh cy-origin 'docker exec -i newapi-postgres psql -U root -d new-api -v ON_ERROR_STOP=1' < scripts/migrate_claude_affinity_user_id_ssh.sql

BEGIN;

UPDATE options
SET value = (
    SELECT jsonb_agg(
        CASE
            WHEN rule->>'name' = 'claude cli trace' THEN
                jsonb_set(
                    rule,
                    '{key_sources}',
                    '[{"type":"context_int","key":"id"},{"type":"gjson","path":"metadata.user_id"}]'::jsonb,
                    TRUE
                )
            ELSE rule
        END
    )::text
    FROM jsonb_array_elements(value::jsonb) AS rules(rule)
)
WHERE key = 'channel_affinity_setting.rules';

COMMIT;

SELECT rule->>'name' AS rule_name, rule->'key_sources' AS key_sources
FROM options, jsonb_array_elements(value::jsonb) AS rules(rule)
WHERE key = 'channel_affinity_setting.rules'
  AND rule->>'name' = 'claude cli trace';
