-- 模型广场 description 中性化（不暴露上游）
-- docker exec -i newapi-postgres psql -U root -d new-api < migrate_leonardo_seedance_description_neutral.sql

BEGIN;

UPDATE models AS m SET
    description = v.description,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('cy-sd4-seedance-2.0', 'Seedance 2.0 标准版。文生/图生/多模态/首尾帧，480p / HD 720p，4–15 秒。'),
    ('cy-sd4-seedance-2.0-fast', 'Seedance 2.0 Fast。更快出片，参数同标准版。'),
    ('leonardo-seedance-2.0', 'Seedance 2.0 标准版。文生/图生/多模态/首尾帧，480p / HD 720p，4–15 秒。'),
    ('leonardo-seedance-2.0-fast', 'Seedance 2.0 Fast。更快出片，参数同标准版。')
) AS v(model_name, description)
WHERE m.model_name = v.model_name AND m.deleted_at IS NULL;

COMMIT;

SELECT model_name, description
FROM models
WHERE model_name LIKE 'cy-sd4-seedance-%' AND deleted_at IS NULL
ORDER BY model_name;
