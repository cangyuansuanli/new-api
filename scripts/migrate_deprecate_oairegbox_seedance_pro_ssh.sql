-- 下线 OAIREGBox Seedance 满血三模型（已由 oairegbox-seedance-2.0-* 替代）
-- contabo: docker exec -i newapi-postgres psql -U root -d new-api < migrate_deprecate_oairegbox_seedance_pro_ssh.sql

BEGIN;

UPDATE models SET deleted_at = NOW(), updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE model_name IN (
    'oairegbox-seedance-pro-720p',
    'oairegbox-seedance-fast-720p',
    'oairegbox-seedance-pro-1080p'
) AND deleted_at IS NULL;

DELETE FROM abilities WHERE model IN (
    'oairegbox-seedance-pro-720p',
    'oairegbox-seedance-fast-720p',
    'oairegbox-seedance-pro-1080p'
);

COMMIT;

-- ModelPrice / ModelRatio 由 seed 脚本清理（JSON 字段）
