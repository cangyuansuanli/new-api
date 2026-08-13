-- 退役 Manju Sora2 视频能力；保留同渠道仍在使用的图片模型。
-- docker exec -i newapi-postgres psql -U root -d new-api < retire_manju_sora2_ssh.sql

BEGIN;

DELETE FROM abilities
WHERE model = 'manju-openai-sora2';

UPDATE models
SET status = 0,
    deleted_at = COALESCE(deleted_at, EXTRACT(EPOCH FROM NOW())::BIGINT),
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE model_name = 'manju-openai-sora2';

UPDATE channels
SET models = array_to_string(
        array_remove(string_to_array(models, ','), 'manju-openai-sora2'),
        ','
    )
WHERE models IS NOT NULL
  AND 'manju-openai-sora2' = ANY(string_to_array(models, ','));

COMMIT;

SELECT model_name, status, deleted_at
FROM models
WHERE model_name = 'manju-openai-sora2';

SELECT channel_id, model, enabled
FROM abilities
WHERE model = 'manju-openai-sora2';
