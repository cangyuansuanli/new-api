-- migrate_leonardo_seedance_mini_ssh.sql
-- Leonardo Seedance 2.0 Mini（渠道 #67）：补 models 元数据。
-- contabo: docker exec -i newapi-postgres psql -U root -d new-api < migrate_leonardo_seedance_mini_ssh.sql

BEGIN;

-- 渠道 models / mapping / group（用户已加 mini 时幂等）
UPDATE channels SET
    models = 'cy-sd4-seedance-2.0,cy-sd4-seedance-2.0-fast,cy-sd4-seedance-2.0-mini',
    model_mapping = '{
  "cy-sd4-seedance-2.0": "seedance-2.0",
  "cy-sd4-seedance-2.0-fast": "seedance-2.0-fast",
  "cy-sd4-seedance-2.0-mini": "seedance-2.0-mini"
}'::text,
    "group" = 'VIDEO,全模型-无claude/gpt',
    status = 1
WHERE id = 67;

-- abilities（幂等）
DELETE FROM abilities WHERE channel_id = 67 AND model = 'cy-sd4-seedance-2.0-mini';

INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT g.grp, 'cy-sd4-seedance-2.0-mini', 67, true, 0, 90
FROM (VALUES ('VIDEO'), ('全模型-无claude/gpt')) AS g(grp);

-- models
INSERT INTO models (model_name, description, tags, vendor_id, endpoints, status, sync_official, video_profile_id, created_time, updated_time)
SELECT v.model_name, v.description, v.tags, 4, v.endpoints, 1, 0, v.video_profile_id,
    EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    (
        'cy-sd4-seedance-2.0-mini',
        'Seedance 2.0 Mini。轻量版，文生/图生/多模态/首尾帧，480p / HD 720p，4–15 秒。',
        'video,seedance,subscription,mini',
        '{"openai-video":{"path":"/v1/videos","method":"POST"}}',
        'video-tpl-cy-sd4-seedance-async'
    )
) AS v(model_name, description, tags, endpoints, video_profile_id)
WHERE NOT EXISTS (
    SELECT 1 FROM models m WHERE m.model_name = v.model_name AND m.deleted_at IS NULL
);

UPDATE models AS m SET
    description = v.description,
    tags = v.tags,
    endpoints = v.endpoints,
    video_profile_id = v.video_profile_id,
    status = 1,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    (
        'cy-sd4-seedance-2.0-mini',
        'Seedance 2.0 Mini。轻量版，文生/图生/多模态/首尾帧，480p / HD 720p，4–15 秒。',
        'video,seedance,subscription,mini',
        '{"openai-video":{"path":"/v1/videos","method":"POST"}}',
        'video-tpl-cy-sd4-seedance-async'
    )
) AS v(model_name, description, tags, endpoints, video_profile_id)
WHERE m.model_name = v.model_name AND m.deleted_at IS NULL;

COMMIT;
