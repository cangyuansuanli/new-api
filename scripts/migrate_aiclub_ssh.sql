-- Aiclub 生图备用线路：GPT Image 2 + Nano Banana（12 固定档位 SKU）
-- 上游: https://api.aiclub.cv  POST/GET /v1/videos
-- GPT Image 各档位出站映射为上游 gpt-image2-*-high（固定 high 质量）
--
-- 执行:
--   docker exec -i newapi-postgres psql -v ON_ERROR_STOP=1 -U root -d new-api \
--     < scripts/migrate_aiclub_ssh.sql

BEGIN;

INSERT INTO model_channel_prefixes (prefix, note, enabled, sort_order, created_time, updated_time)
VALUES ('cy-ac-', '生图线路 Aiclub', TRUE, 134, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (prefix) DO UPDATE SET
    note = EXCLUDED.note,
    enabled = TRUE,
    sort_order = EXCLUDED.sort_order,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT;

INSERT INTO model_public_aliases (internal_name, public_name, hidden_from_marketplace, created_time, updated_time)
VALUES
    ('cy-ac-gpt-image-2-1k', 'gpt-image-2-1k', TRUE, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-ac-gpt-image-2-2k', 'gpt-image-2-2k', TRUE, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-ac-gpt-image-2-4k', 'gpt-image-2-4k', TRUE, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-ac-nano-banana-pro-1k', 'nano-banana-pro-1k', TRUE, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-ac-nano-banana-pro-2k', 'nano-banana-pro-2k', TRUE, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-ac-nano-banana-pro-4k', 'nano-banana-pro-4k', TRUE, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-ac-nano-banana-1k', 'nano-banana-1k', TRUE, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-ac-nano-banana-2k', 'nano-banana-2k', TRUE, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-ac-nano-banana-4k', 'nano-banana-4k', TRUE, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-ac-nano-banana2-1k', 'nano-banana2-1k', TRUE, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-ac-nano-banana2-2k', 'nano-banana2-2k', TRUE, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('cy-ac-nano-banana2-4k', 'nano-banana2-4k', TRUE, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (internal_name) DO UPDATE SET
    public_name = EXCLUDED.public_name,
    hidden_from_marketplace = EXCLUDED.hidden_from_marketplace,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT;

INSERT INTO models (
    model_name, description, tags, vendor_id, endpoints, status,
    sync_official, image_profile_id, created_time, updated_time
)
SELECT
    v.model_name, v.description, v.tags, 2, '["openai"]', 1,
    0, v.image_profile_id,
    EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('cy-ac-gpt-image-2-1k', 'GPT Image 2 1K 固定档位。', 'image,gpt-image,1k', 'image-tpl-adobe2api-1k'),
    ('cy-ac-gpt-image-2-2k', 'GPT Image 2 2K 固定档位。', 'image,gpt-image,2k', 'image-tpl-adobe2api-2k'),
    ('cy-ac-gpt-image-2-4k', 'GPT Image 2 4K 固定档位。', 'image,gpt-image,4k', 'image-tpl-adobe2api-4k'),
    ('cy-ac-nano-banana-pro-1k', 'Nano Banana Pro 1K 固定档位。', 'image,nano-banana,pro,1k', 'image-tpl-adobe2api-1k'),
    ('cy-ac-nano-banana-pro-2k', 'Nano Banana Pro 2K 固定档位。', 'image,nano-banana,pro,2k', 'image-tpl-adobe2api-2k'),
    ('cy-ac-nano-banana-pro-4k', 'Nano Banana Pro 4K 固定档位。', 'image,nano-banana,pro,4k', 'image-tpl-adobe2api-4k'),
    ('cy-ac-nano-banana-1k', 'Nano Banana 1K 固定档位。', 'image,nano-banana,1k', 'image-tpl-adobe2api-1k'),
    ('cy-ac-nano-banana-2k', 'Nano Banana 2K 固定档位。', 'image,nano-banana,2k', 'image-tpl-adobe2api-2k'),
    ('cy-ac-nano-banana-4k', 'Nano Banana 4K 固定档位。', 'image,nano-banana,4k', 'image-tpl-adobe2api-4k'),
    ('cy-ac-nano-banana2-1k', 'Nano Banana 2 1K 固定档位。', 'image,nano-banana2,1k', 'image-tpl-adobe2api-1k'),
    ('cy-ac-nano-banana2-2k', 'Nano Banana 2 2K 固定档位。', 'image,nano-banana2,2k', 'image-tpl-adobe2api-2k'),
    ('cy-ac-nano-banana2-4k', 'Nano Banana 2 4K 固定档位。', 'image,nano-banana2,4k', 'image-tpl-adobe2api-4k')
) AS v(model_name, description, tags, image_profile_id)
WHERE NOT EXISTS (
    SELECT 1 FROM models m WHERE m.model_name = v.model_name AND m.deleted_at IS NULL
);

UPDATE models AS m SET
    description = v.description,
    tags = v.tags,
    vendor_id = 2,
    endpoints = '["openai"]',
    image_profile_id = v.image_profile_id,
    status = 1,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('cy-ac-gpt-image-2-1k', 'GPT Image 2 1K 固定档位。', 'image,gpt-image,1k', 'image-tpl-adobe2api-1k'),
    ('cy-ac-gpt-image-2-2k', 'GPT Image 2 2K 固定档位。', 'image,gpt-image,2k', 'image-tpl-adobe2api-2k'),
    ('cy-ac-gpt-image-2-4k', 'GPT Image 2 4K 固定档位。', 'image,gpt-image,4k', 'image-tpl-adobe2api-4k'),
    ('cy-ac-nano-banana-pro-1k', 'Nano Banana Pro 1K 固定档位。', 'image,nano-banana,pro,1k', 'image-tpl-adobe2api-1k'),
    ('cy-ac-nano-banana-pro-2k', 'Nano Banana Pro 2K 固定档位。', 'image,nano-banana,pro,2k', 'image-tpl-adobe2api-2k'),
    ('cy-ac-nano-banana-pro-4k', 'Nano Banana Pro 4K 固定档位。', 'image,nano-banana,pro,4k', 'image-tpl-adobe2api-4k'),
    ('cy-ac-nano-banana-1k', 'Nano Banana 1K 固定档位。', 'image,nano-banana,1k', 'image-tpl-adobe2api-1k'),
    ('cy-ac-nano-banana-2k', 'Nano Banana 2K 固定档位。', 'image,nano-banana,2k', 'image-tpl-adobe2api-2k'),
    ('cy-ac-nano-banana-4k', 'Nano Banana 4K 固定档位。', 'image,nano-banana,4k', 'image-tpl-adobe2api-4k'),
    ('cy-ac-nano-banana2-1k', 'Nano Banana 2 1K 固定档位。', 'image,nano-banana2,1k', 'image-tpl-adobe2api-1k'),
    ('cy-ac-nano-banana2-2k', 'Nano Banana 2 2K 固定档位。', 'image,nano-banana2,2k', 'image-tpl-adobe2api-2k'),
    ('cy-ac-nano-banana2-4k', 'Nano Banana 2 4K 固定档位。', 'image,nano-banana2,4k', 'image-tpl-adobe2api-4k')
) AS v(model_name, description, tags, image_profile_id)
WHERE m.model_name = v.model_name AND m.deleted_at IS NULL;

DO $$
DECLARE
    aiclub_channel_id INT;
    model_list TEXT := 'cy-ac-gpt-image-2-1k,cy-ac-gpt-image-2-2k,cy-ac-gpt-image-2-4k,cy-ac-nano-banana-pro-1k,cy-ac-nano-banana-pro-2k,cy-ac-nano-banana-pro-4k,cy-ac-nano-banana-1k,cy-ac-nano-banana-2k,cy-ac-nano-banana-4k,cy-ac-nano-banana2-1k,cy-ac-nano-banana2-2k,cy-ac-nano-banana2-4k';
    mapping JSONB := '{
  "cy-ac-gpt-image-2-1k": "gpt-image2-1k-high",
  "cy-ac-gpt-image-2-2k": "gpt-image2-2k-high",
  "cy-ac-gpt-image-2-4k": "gpt-image2-4k-high",
  "cy-ac-nano-banana-pro-1k": "Nano-Banana-Pro-1k",
  "cy-ac-nano-banana-pro-2k": "Nano-Banana-Pro-2k",
  "cy-ac-nano-banana-pro-4k": "Nano-Banana-Pro-4k",
  "cy-ac-nano-banana-1k": "Nano-Banana-2-1k",
  "cy-ac-nano-banana-2k": "Nano-Banana-2-2k",
  "cy-ac-nano-banana-4k": "Nano-Banana-2-4k",
  "cy-ac-nano-banana2-1k": "Nano-Banana-2-1k",
  "cy-ac-nano-banana2-2k": "Nano-Banana-2-2k",
  "cy-ac-nano-banana2-4k": "Nano-Banana-2-4k"
}'::jsonb;
    aiclub_key TEXT := 'sk-mBZkQxopoW0XOmEDzLYKPOSjDhsEZ9eOeKDVIPFj91ksM1xo';
    aiclub_base_url TEXT := 'https://api.aiclub.cv';
BEGIN
    INSERT INTO channels (
        type, key, status, name, weight, created_time, base_url, models, "group",
        model_mapping, priority, auto_ban, tag, remark
    )
    SELECT
        1,
        aiclub_key,
        1,
        'aiclub-image',
        90,
        EXTRACT(EPOCH FROM NOW())::BIGINT,
        aiclub_base_url,
        model_list,
        'IMAGE,全模型-无claude/gpt',
        mapping::text,
        0,
        1,
        'aiclub-image',
        'Aiclub 生图备用线（GPT high + Nano Banana）'
    WHERE NOT EXISTS (
        SELECT 1 FROM channels ch WHERE ch.name = 'aiclub-image'
    );

    SELECT id INTO aiclub_channel_id
    FROM channels
    WHERE name = 'aiclub-image'
    ORDER BY id
    LIMIT 1;

    IF aiclub_channel_id IS NULL THEN
        SELECT id INTO aiclub_channel_id
        FROM channels
        WHERE type = 1
          AND base_url ILIKE '%aiclub.cv%'
        ORDER BY id
        LIMIT 1;
    END IF;

    IF aiclub_channel_id IS NULL THEN
        RAISE EXCEPTION 'aiclub channel not found after insert';
    END IF;

    UPDATE channels SET
        key = aiclub_key,
        base_url = aiclub_base_url,
        models = model_list,
        model_mapping = mapping::text,
        "group" = trim(both ',' from concat_ws(',', NULLIF("group", ''), 'IMAGE', '全模型-无claude/gpt')),
        status = 1,
        test_model = 'cy-ac-gpt-image-2-2k'
    WHERE id = aiclub_channel_id;

    DELETE FROM abilities
    WHERE channel_id = aiclub_channel_id
       OR model LIKE 'cy-ac-%';

    INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
    SELECT g.grp, m.model, aiclub_channel_id, FALSE, 0, 10
    FROM (VALUES
        ('cy-ac-gpt-image-2-1k'),
        ('cy-ac-gpt-image-2-2k'),
        ('cy-ac-gpt-image-2-4k'),
        ('cy-ac-nano-banana-pro-1k'),
        ('cy-ac-nano-banana-pro-2k'),
        ('cy-ac-nano-banana-pro-4k'),
        ('cy-ac-nano-banana-1k'),
        ('cy-ac-nano-banana-2k'),
        ('cy-ac-nano-banana-4k'),
        ('cy-ac-nano-banana2-1k'),
        ('cy-ac-nano-banana2-2k'),
        ('cy-ac-nano-banana2-4k')
    ) AS m(model)
    CROSS JOIN (VALUES ('IMAGE'), ('全模型-无claude/gpt')) AS g(grp);
END $$;

COMMIT;

SELECT model_name, image_profile_id, status FROM models
WHERE model_name LIKE 'cy-ac-%' AND deleted_at IS NULL
ORDER BY 1;

SELECT id, name, status, base_url, models, model_mapping, test_model
FROM channels
WHERE name = 'aiclub-image' OR base_url ILIKE '%aiclub.cv%'
ORDER BY id;
