-- 允许多个 internal 绑定同一 public_name；仅禁止同时「模型广场可见 + models.status=1」。
-- 源站: ssh cy-origin 'docker exec -i newapi-postgres psql -U root -d new-api -v ON_ERROR_STOP=1' < migrate_public_alias_shared_name_ssh.sql

BEGIN;

DROP INDEX IF EXISTS uk_model_public_alias_public;
ALTER TABLE model_public_aliases DROP CONSTRAINT IF EXISTS idx_model_public_aliases_public_name;
DROP INDEX IF EXISTS idx_model_public_aliases_public_name;
CREATE INDEX IF NOT EXISTS idx_model_public_alias_public ON model_public_aliases (public_name);

COMMIT;

SELECT public_name, COUNT(*) AS alias_count
FROM model_public_aliases
WHERE deleted_at IS NULL
GROUP BY public_name
HAVING COUNT(*) > 1
ORDER BY public_name;
