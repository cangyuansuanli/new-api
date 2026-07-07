-- Manju Sora2 渠道 70：注册 manju-openai-sora2 模型、abilities、前缀说明（源站 SSH 执行）
-- contabo: docker exec -i newapi-postgres psql -U root -d new-api < migrate_manju_sora2_ssh.sql

BEGIN;

-- 1. 渠道前缀说明（生图 + 视频）
UPDATE model_channel_prefixes SET
    note = 'Gemini 生图 / Sora2 视频',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE prefix = 'manju-';

-- 2. 渠道 70：加入 VIDEO 分组以便 /v1/videos 路由
UPDATE channels SET
    "group" = 'IMAGE,VIDEO,全模型-无claude/gpt',
    status = 1
WHERE id = 70;

-- 3. abilities：VIDEO 分组（移除误配的 IMAGE）
DELETE FROM abilities WHERE channel_id = 70 AND model = 'manju-openai-sora2';

INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT g.grp, 'manju-openai-sora2', 70, true, 0, 90
FROM (VALUES ('VIDEO'), ('全模型-无claude/gpt')) AS g(grp);

-- 4. models 元数据
INSERT INTO models (model_name, description, tags, vendor_id, endpoints, status, sync_official, video_profile_id, created_time, updated_time)
SELECT v.model_name, v.description, v.tags, 1, v.endpoints, 1, 0, v.video_profile_id,
    EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    (
        'manju-openai-sora2',
        'OpenAI Sora2 视频生成。POST /v1/videos 创建任务，GET /v1/videos/{task_id} 轮询取片；支持文生视频与单张参考图。',
        'video,sora,openai,video-sora',
        '{"openai-video":{"path":"/v1/videos","method":"POST"}}',
        'video-tpl-manju-sora-async'
    )
) AS v(model_name, description, tags, endpoints, video_profile_id)
WHERE NOT EXISTS (
    SELECT 1 FROM models m WHERE m.model_name = v.model_name AND m.deleted_at IS NULL
);

UPDATE models AS m SET
    description = v.description,
    tags = v.tags,
    vendor_id = 1,
    endpoints = v.endpoints,
    video_profile_id = v.video_profile_id,
    status = 1,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    (
        'manju-openai-sora2',
        'OpenAI Sora2 视频生成。POST /v1/videos 创建任务，GET /v1/videos/{task_id} 轮询取片；支持文生视频与单张参考图。',
        'video,sora,openai,video-sora',
        '{"openai-video":{"path":"/v1/videos","method":"POST"}}',
        'video-tpl-manju-sora-async'
    )
) AS v(model_name, description, tags, endpoints, video_profile_id)
WHERE m.model_name = v.model_name AND m.deleted_at IS NULL;

COMMIT;

-- api_doc 由 seed_manju_sora2_api_doc.py 写入（不修改 ModelPrice）

SELECT 'channels' AS section, id, name, models FROM channels WHERE id = 70;
SELECT 'models' AS section, model_name, video_profile_id, tags FROM models WHERE model_name = 'manju-openai-sora2' AND deleted_at IS NULL;
SELECT 'abilities' AS section, "group", model FROM abilities WHERE channel_id = 70 AND model = 'manju-openai-sora2';
