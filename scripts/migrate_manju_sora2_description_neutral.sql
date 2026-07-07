-- manju-openai-sora2 模型广场 description / api 端点中性化（不暴露上游；源站 SSH 执行）
-- docker exec -i newapi-postgres psql -U root -d new-api < migrate_manju_sora2_description_neutral.sql

BEGIN;

UPDATE model_channel_prefixes SET
    note = 'Gemini 生图 / Sora2 视频',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE prefix = 'manju-';

UPDATE models SET
    description = 'OpenAI Sora2 视频生成。POST /v1/videos 创建任务，GET /v1/videos/{task_id} 轮询取片；支持文生视频与单张参考图。',
    tags = 'video,sora,openai,video-sora',
    endpoints = '{"openai-video":{"path":"/v1/videos","method":"POST"}}',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE model_name = 'manju-openai-sora2' AND deleted_at IS NULL;

COMMIT;

SELECT model_name, description, tags, endpoints
FROM models
WHERE model_name = 'manju-openai-sora2' AND deleted_at IS NULL;
