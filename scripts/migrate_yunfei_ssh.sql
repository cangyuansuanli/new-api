-- Yunfei 生图：gpt-image-2 4K（OpenAI 渠道）+ Gemini Banana（Gemini 渠道）
-- 源站: docker exec -i newapi-postgres psql -v ON_ERROR_STOP=1 -U root -d new-api < migrate_yunfei_ssh.sql
--
-- 前置：在管理后台创建两条渠道（同一 base_url，不同 type）：
--   1) type=1  OpenAI，name 建议 yunfei-gpt-image-2
--   2) type=24 Gemini，name 建议 yunfei-gemini-banana
-- 创建后将下方 channel id 占位符替换为实际 ID，或按 name 自动匹配。

BEGIN;

INSERT INTO model_channel_prefixes (prefix, note, enabled, sort_order, created_time, updated_time)
VALUES ('cy-yf-', '生图线路 YF', TRUE, 133, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (prefix) DO UPDATE SET
    note = EXCLUDED.note,
    enabled = TRUE,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT;

INSERT INTO model_public_aliases (internal_name, public_name, created_time, updated_time)
VALUES
    ('cy-yf-gpt-image-2-4k', 'gpt-image-2-4k', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-yf-gemini-banana-pro', 'gemini-banana-pro', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-yf-gemini-banana-flash', 'gemini-banana-flash', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (internal_name) DO UPDATE SET
    public_name = EXCLUDED.public_name,
    updated_time = EXCLUDED.updated_time;

INSERT INTO models (
    model_name, description, tags, vendor_id, endpoints, status,
    sync_official, image_profile_id, created_time, updated_time
)
SELECT
    v.model_name, v.description, v.tags, v.vendor_id, '["openai"]', 1,
    0, v.image_profile_id,
    EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('cy-yf-gpt-image-2-4k', 'GPT Image 2 4K 固定档位。', 'image,gpt-image,4k', 2, 'image-tpl-cy-yf-gpt-image-2-4k'),
    ('cy-yf-gemini-banana-pro', 'Gemini Banana Pro。同步出图，支持 1K/2K/4K。', 'image,gemini,banana,pro', 6, 'image-tpl-cy-yf-banana-pro'),
    ('cy-yf-gemini-banana-flash', 'Gemini Banana Flash。同步出图，支持 1K/2K/4K 与超宽比例。', 'image,gemini,banana,flash', 6, 'image-tpl-cy-yf-banana-flash')
) AS v(model_name, description, tags, vendor_id, image_profile_id)
WHERE NOT EXISTS (
    SELECT 1 FROM models m WHERE m.model_name = v.model_name AND m.deleted_at IS NULL
);

UPDATE models AS m SET
    description = v.description,
    tags = v.tags,
    vendor_id = v.vendor_id,
    endpoints = '["openai"]',
    image_profile_id = v.image_profile_id,
    status = 1,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('cy-yf-gpt-image-2-4k', 'GPT Image 2 4K 固定档位。', 'image,gpt-image,4k', 2, 'image-tpl-cy-yf-gpt-image-2-4k'),
    ('cy-yf-gemini-banana-pro', 'Gemini Banana Pro。同步出图，支持 1K/2K/4K。', 'image,gemini,banana,pro', 6, 'image-tpl-cy-yf-banana-pro'),
    ('cy-yf-gemini-banana-flash', 'Gemini Banana Flash。同步出图，支持 1K/2K/4K 与超宽比例。', 'image,gemini,banana,flash', 6, 'image-tpl-cy-yf-banana-flash')
) AS v(model_name, description, tags, vendor_id, image_profile_id)
WHERE m.model_name = v.model_name AND m.deleted_at IS NULL;

UPDATE model_ui_param_profiles AS p
SET deleted_at = NOW(),
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE p.capability = 'image'
  AND p.profile_id IN (
      'image-tpl-yf-gpt-image-2-4k',
      'image-tpl-yf-banana-pro',
      'image-tpl-yf-banana-flash'
  )
  AND p.deleted_at IS NULL;

DO $$
DECLARE
    openai_channel_id INT;
    gemini_channel_id INT;
BEGIN
    SELECT id INTO openai_channel_id
    FROM channels
    WHERE type = 1
      AND (
          name ILIKE '%yunfei%gpt%'
          OR base_url ILIKE '%yunfei.best%'
      )
    ORDER BY id
    LIMIT 1;

    SELECT id INTO gemini_channel_id
    FROM channels
    WHERE type = 24
      AND (
          name ILIKE '%yunfei%banana%'
          OR base_url ILIKE '%yunfei.best%'
      )
    ORDER BY id
    LIMIT 1;

    IF openai_channel_id IS NULL THEN
        RAISE NOTICE 'skip OpenAI channel update: create type=1 channel with yunfei base_url first';
    ELSE
        UPDATE channels SET
            models = 'cy-yf-gpt-image-2-4k',
            model_mapping = '{"cy-yf-gpt-image-2-4k":"gpt-image-2"}'::text,
            "group" = trim(both ',' from concat_ws(',', NULLIF("group", ''), 'IMAGE')),
            status = 1
        WHERE id = openai_channel_id;

        DELETE FROM abilities
        WHERE channel_id = openai_channel_id
           OR model = 'cy-yf-gpt-image-2-4k';

        INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
        SELECT g.grp, 'cy-yf-gpt-image-2-4k', openai_channel_id, TRUE, 0, 90
        FROM (VALUES ('IMAGE'), ('全模型-无claude/gpt')) AS g(grp);
    END IF;

    IF gemini_channel_id IS NULL THEN
        RAISE NOTICE 'skip Gemini channel update: create type=24 channel with yunfei base_url first';
    ELSE
        UPDATE channels SET
            models = 'cy-yf-gemini-banana-pro,cy-yf-gemini-banana-flash',
            model_mapping = '{
  "cy-yf-gemini-banana-pro": "gemini-3-pro-image-preview",
  "cy-yf-gemini-banana-flash": "gemini-3.1-flash-image-preview"
}'::text,
            "group" = trim(both ',' from concat_ws(',', NULLIF("group", ''), 'IMAGE')),
            status = 1
        WHERE id = gemini_channel_id;

        DELETE FROM abilities
        WHERE channel_id = gemini_channel_id
           OR model IN ('cy-yf-gemini-banana-pro', 'cy-yf-gemini-banana-flash');

        INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
        SELECT g.grp, m.model, gemini_channel_id, TRUE, 0, 90
        FROM (VALUES
            ('cy-yf-gemini-banana-pro'),
            ('cy-yf-gemini-banana-flash')
        ) AS m(model)
        CROSS JOIN (VALUES ('IMAGE'), ('全模型-无claude/gpt')) AS g(grp);
    END IF;
END $$;

COMMIT;

SELECT model_name, image_profile_id, status FROM models
WHERE model_name LIKE 'cy-yf-%' AND deleted_at IS NULL
ORDER BY 1;
