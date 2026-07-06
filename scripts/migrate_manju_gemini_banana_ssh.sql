-- Manju Gemini Banana 渠道 70：注册模型、映射、abilities、前缀与定价（源站 SSH 执行）
-- contabo: docker exec -i newapi-postgres psql -U root -d new-api < migrate_manju_gemini_banana_ssh.sql

BEGIN;

-- 1. 渠道前缀
INSERT INTO model_channel_prefixes (prefix, note, enabled, sort_order, created_time, updated_time)
VALUES ('manju-', 'Manju Gemini 生图', TRUE, 125, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (prefix) DO UPDATE SET
    note = EXCLUDED.note,
    enabled = TRUE,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT;

-- 2. 渠道 70：internal 名 + model_mapping → 上游名
UPDATE channels SET
    models = 'manju-gemini-banana-pro-4k,manju-gemini-banana-flash-lite,manju-gemini-banana-pro-1/2k,manju-gemini-banana-2.0-1/2k,manju-gemini-banana-2.0-4k',
    model_mapping = '{
  "manju-gemini-banana-pro-4k": "gemini-3.0-pro-image 4K",
  "manju-gemini-banana-flash-lite": "gemini-3.1-flash-lite-image",
  "manju-gemini-banana-pro-1/2k": "gemini-3.0-pro-image",
  "manju-gemini-banana-2.0-1/2k": "Nano Banana 2",
  "manju-gemini-banana-2.0-4k": "Nano Banana 2 4K"
}'::text,
    status = 1
WHERE id = 70;

-- 3. abilities
DELETE FROM abilities WHERE channel_id = 70;

INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT g.grp, m.model, 70, true, 0, 0
FROM (VALUES
    ('manju-gemini-banana-pro-4k'),
    ('manju-gemini-banana-flash-lite'),
    ('manju-gemini-banana-pro-1/2k'),
    ('manju-gemini-banana-2.0-1/2k'),
    ('manju-gemini-banana-2.0-4k')
) AS m(model)
CROSS JOIN (VALUES ('IMAGE'), ('全模型-无claude/gpt')) AS g(grp);

-- 4. models 元数据（OpenAI Image API 为主；chat/completions 兼容）
INSERT INTO models (model_name, description, tags, vendor_id, endpoints, status, sync_official, image_profile_id, created_time, updated_time)
SELECT v.model_name, v.description, v.tags, 6, '["openai"]', 1, 0, v.image_profile_id,
    EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('manju-gemini-banana-pro-4k', 'Manju Gemini Banana Pro 4K。同步/异步出图，支持 4K。', 'image,gemini,banana,pro,4k', 'image-tpl-banana-chat'),
    ('manju-gemini-banana-flash-lite', 'Manju Gemini Banana Flash Lite。同步/异步出图，仅 1K。', 'image,gemini,banana,flash,lite', 'image-tpl-banana-chat-flash-lite'),
    ('manju-gemini-banana-pro-1/2k', 'Manju Gemini Banana Pro 1K/2K。同步/异步出图。', 'image,gemini,banana,pro', 'image-tpl-banana-chat'),
    ('manju-gemini-banana-2.0-1/2k', 'Manju Nano Banana 2.0 1K/2K。同步/异步出图。', 'image,gemini,banana,2.0', 'image-tpl-banana-chat'),
    ('manju-gemini-banana-2.0-4k', 'Manju Nano Banana 2.0 4K。同步/异步出图。', 'image,gemini,banana,2.0,4k', 'image-tpl-banana-chat')
) AS v(model_name, description, tags, image_profile_id)
WHERE NOT EXISTS (
    SELECT 1 FROM models m WHERE m.model_name = v.model_name AND m.deleted_at IS NULL
);

UPDATE models AS m SET
    description = v.description,
    tags = v.tags,
    vendor_id = 6,
    endpoints = '["openai"]',
    image_profile_id = v.image_profile_id,
    status = 1,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('manju-gemini-banana-pro-4k', 'Manju Gemini Banana Pro 4K。同步/异步出图，支持 4K。', 'image,gemini,banana,pro,4k', 'image-tpl-banana-chat'),
    ('manju-gemini-banana-flash-lite', 'Manju Gemini Banana Flash Lite。同步/异步出图，仅 1K。', 'image,gemini,banana,flash,lite', 'image-tpl-banana-chat-flash-lite'),
    ('manju-gemini-banana-pro-1/2k', 'Manju Gemini Banana Pro 1K/2K。同步/异步出图。', 'image,gemini,banana,pro', 'image-tpl-banana-chat'),
    ('manju-gemini-banana-2.0-1/2k', 'Manju Nano Banana 2.0 1K/2K。同步/异步出图。', 'image,gemini,banana,2.0', 'image-tpl-banana-chat'),
    ('manju-gemini-banana-2.0-4k', 'Manju Nano Banana 2.0 4K。同步/异步出图。', 'image,gemini,banana,2.0,4k', 'image-tpl-banana-chat')
) AS v(model_name, description, tags, image_profile_id)
WHERE m.model_name = v.model_name AND m.deleted_at IS NULL;

COMMIT;

-- 5. ModelPrice（USD/次）
-- 由 seed_manju_gemini_banana_api_doc.py 写入 api_doc 后一并更新定价

SELECT 'channels' AS section, id, name, models FROM channels WHERE id = 70;
SELECT 'models' AS section, model_name, image_profile_id FROM models WHERE model_name LIKE 'manju-gemini-banana%' AND deleted_at IS NULL ORDER BY 1;
