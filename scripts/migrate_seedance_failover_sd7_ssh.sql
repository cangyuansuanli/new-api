-- Seedance 2.0 failover via model_routing_aliases (legacy 名) + sd4 显式展示名
--
-- 展示名（model_public_aliases，模型广场独立展示）：
--   sd7-seedance-2.0-720p / sd4-seedance-2.0 / sd4-seedance-2.0-fast
-- 路由名（model_routing_aliases，可随时切换指向不同 internal）：
--   seedance-2.0 → cy-sd7-seedance-2.0-720p
--   seedance-2.0-fast → cy-sd8-seedance-2.0-fast
--
-- 源站: ssh cy-origin 'docker exec -i newapi-postgres psql -U root -d new-api -v ON_ERROR_STOP=1' < migrate_seedance_failover_sd7_ssh.sql
-- 需 new-api 代码支持 model_routing_aliases 后再滚动重启

BEGIN;

CREATE TABLE IF NOT EXISTS model_routing_aliases (
    id BIGSERIAL PRIMARY KEY,
    public_name VARCHAR(255) NOT NULL,
    internal_name VARCHAR(255) NOT NULL,
    note VARCHAR(255) DEFAULT '',
    created_time BIGINT,
    updated_time BIGINT,
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_model_routing_alias_public ON model_routing_aliases (public_name);
CREATE INDEX IF NOT EXISTS idx_model_routing_alias_internal ON model_routing_aliases (internal_name);
CREATE INDEX IF NOT EXISTS idx_model_routing_aliases_deleted_at ON model_routing_aliases (deleted_at);

-- sd4 显式展示名（不占用 legacy seedance-2.0）
INSERT INTO model_public_aliases (internal_name, public_name, created_time, updated_time)
VALUES
    ('cy-sd4-seedance-2.0',      'sd4-seedance-2.0',      EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-sd4-seedance-2.0-fast', 'sd4-seedance-2.0-fast', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-sd7-seedance-2.0-720p', 'sd7-seedance-2.0-720p', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (internal_name) DO UPDATE SET
    public_name = EXCLUDED.public_name,
    deleted_at = NULL,
    updated_time = EXCLUDED.updated_time;

-- 恢复 sd7 展示名（若曾被误改为 seedance-2.0）
UPDATE model_public_aliases
SET public_name = 'sd7-seedance-2.0-720p',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE internal_name = 'cy-sd7-seedance-2.0-720p'
  AND public_name <> 'sd7-seedance-2.0-720p';

-- legacy 路由名：仅入站解析，不影响 sd7/sd4 广场展示名
INSERT INTO model_routing_aliases (public_name, internal_name, note, created_time, updated_time)
VALUES
    (
        'seedance-2.0',
        'cy-sd7-seedance-2.0-720p',
        'Legacy neutral name; switch internal to change active sdx route without client API changes.',
        EXTRACT(EPOCH FROM NOW())::BIGINT,
        EXTRACT(EPOCH FROM NOW())::BIGINT
    ),
    (
        'seedance-2.0-fast',
        'cy-sd8-seedance-2.0-fast',
        'Legacy neutral fast name; switch internal to change active sdx route.',
        EXTRACT(EPOCH FROM NOW())::BIGINT,
        EXTRACT(EPOCH FROM NOW())::BIGINT
    )
ON CONFLICT (public_name) DO UPDATE SET
    internal_name = EXCLUDED.internal_name,
    note = EXCLUDED.note,
    deleted_at = NULL,
    updated_time = EXCLUDED.updated_time;

COMMIT;

SELECT 'display aliases' AS section, internal_name, public_name
FROM model_public_aliases
WHERE deleted_at IS NULL
  AND internal_name IN (
      'cy-sd4-seedance-2.0', 'cy-sd4-seedance-2.0-fast',
      'cy-sd7-seedance-2.0-720p', 'cy-sd7-seedance-2.0-1080p'
  )
ORDER BY public_name;

SELECT 'routing aliases' AS section, public_name, internal_name, note
FROM model_routing_aliases
WHERE deleted_at IS NULL
  AND public_name IN ('seedance-2.0', 'seedance-2.0-fast')
ORDER BY public_name;
