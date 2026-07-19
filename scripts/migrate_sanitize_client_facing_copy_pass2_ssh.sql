-- Pass 2: 清理 video/image profile hints 与 api_doc 中残留的 Adobe Firefly / 上游 字样。
-- contabo: docker exec -i newapi-postgres psql -U root -d new-api < migrate_sanitize_client_facing_copy_pass2_ssh.sql

BEGIN;

CREATE OR REPLACE FUNCTION pg_temp.sanitize_client_copy_v2(input text)
RETURNS text
LANGUAGE plpgsql
IMMUTABLE
AS $$
DECLARE
    out text := COALESCE(input, '');
BEGIN
    IF out = '' THEN
        RETURN out;
    END IF;

    out := replace(out, 'Adobe Firefly ', '');
    out := replace(out, 'Adobe2API ', '');
    out := replace(out, '并发生成可能触发上游限流', '并发生成可能触发限流');
    out := replace(out, '输出由上游固定为 PNG', '输出固定为 PNG');
    out := replace(out, '省略时按上游默认值处理', '省略时按平台默认值处理');
    out := replace(out, '为降低网页上游超时概率', '为降低异步超时概率');
    out := replace(out, '网页线路仅保证', '平台仅保证');
    out := replace(out, '网页线路', '平台');
    out := replace(out, '上游', '平台');

    RETURN out;
END;
$$;

UPDATE model_ui_param_profiles
SET hints = pg_temp.sanitize_client_copy_v2(hints),
    note = pg_temp.sanitize_client_copy_v2(note),
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE hints ~* 'Adobe Firefly|Adobe2API|上游'
   OR note ~* 'Adobe Firefly|Adobe2API|上游';

UPDATE models
SET api_doc = pg_temp.sanitize_client_copy_v2(api_doc),
    description = pg_temp.sanitize_client_copy_v2(description),
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE deleted_at IS NULL
  AND (api_doc ~* 'Adobe Firefly|Adobe2API|上游' OR description ~* 'Adobe Firefly|Adobe2API|上游');

COMMIT;

SELECT profile_id, left(hints, 100) AS hints_preview
FROM model_ui_param_profiles
WHERE hints ~* 'Adobe Firefly|Adobe2API|上游|Leonardo|cy-sd[0-9]|adobe2api'
ORDER BY profile_id;
