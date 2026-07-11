-- Adobe2API image SKUs, expand phase (safe to run before code rollout).
-- Creates 12 dedicated Adobe internal models and disabled abilities.
-- Public names are firefly-*, so existing gpt-image-2-{1k,2k,4k} products do not collide.
--
-- Prerequisite:
--   go run ./scripts/seed_model_ui_params/main.go -force
-- Run:
--   docker exec -i newapi-postgres psql -v ON_ERROR_STOP=1 -U root -d new-api \
--     < scripts/migrate_adobe2api_image_skus_expand_ssh.sql

BEGIN;

INSERT INTO model_channel_prefixes (prefix, note, enabled, sort_order, created_time, updated_time)
VALUES ('adobe-', 'Adobe Firefly', TRUE, 41, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (prefix) DO UPDATE SET
    note = EXCLUDED.note,
    enabled = TRUE,
    sort_order = EXCLUDED.sort_order,
    updated_time = EXCLUDED.updated_time,
    deleted_at = NULL;

DO $$
DECLARE
    missing_profiles text;
BEGIN
    IF EXISTS (
        SELECT 1 FROM models
        WHERE model_name LIKE 'adobe-firefly-%'
          AND status = 1
          AND deleted_at IS NULL
    ) OR EXISTS (
        SELECT 1 FROM abilities
        WHERE channel_id = 75
          AND model LIKE 'adobe-firefly-%'
          AND enabled = TRUE
    ) THEN
        RAISE EXCEPTION 'Adobe Firefly SKUs are already active; refusing to rerun expand and disable live products';
    END IF;
    SELECT string_agg(required.profile_id, ', ' ORDER BY required.profile_id)
    INTO missing_profiles
    FROM (VALUES
        ('image-tpl-adobe2api-1k'),
        ('image-tpl-adobe2api-2k'),
        ('image-tpl-adobe2api-4k')
    ) AS required(profile_id)
    WHERE NOT EXISTS (
        SELECT 1
        FROM model_ui_param_profiles p
        WHERE p.capability = 'image'
          AND p.profile_id = required.profile_id
          AND p.deleted_at IS NULL
    );
    IF missing_profiles IS NOT NULL THEN
        RAISE EXCEPTION 'seed Adobe image profiles before expand migration; missing: %', missing_profiles;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM channels WHERE id = 75) THEN
        RAISE EXCEPTION 'Adobe2API channel 75 does not exist';
    END IF;
END $$;

WITH existing_models AS (
    SELECT btrim(item.model) AS model, item.ord
    FROM channels c
    CROSS JOIN LATERAL unnest(string_to_array(COALESCE(c.models, ''), ',')) WITH ORDINALITY AS item(model, ord)
    WHERE c.id = 75 AND btrim(item.model) <> ''
),
wanted_models AS (
    SELECT * FROM (VALUES
        ('adobe-firefly-nano-banana-pro-1k', 20001),
        ('adobe-firefly-nano-banana-pro-2k', 20002),
        ('adobe-firefly-nano-banana-pro-4k', 20003),
        ('adobe-firefly-nano-banana-1k', 20004),
        ('adobe-firefly-nano-banana-2k', 20005),
        ('adobe-firefly-nano-banana-4k', 20006),
        ('adobe-firefly-nano-banana2-1k', 20007),
        ('adobe-firefly-nano-banana2-2k', 20008),
        ('adobe-firefly-nano-banana2-4k', 20009),
        ('adobe-firefly-gpt-image-2-1k', 20010),
        ('adobe-firefly-gpt-image-2-2k', 20011),
        ('adobe-firefly-gpt-image-2-4k', 20012)
    ) AS v(model, ord)
),
merged_models AS (
    SELECT model, min(ord) AS ord
    FROM (
        SELECT model, ord FROM existing_models
        UNION ALL
        SELECT model, ord FROM wanted_models
    ) all_models
    GROUP BY model
)
UPDATE channels
SET models = (SELECT string_agg(model, ',' ORDER BY ord) FROM merged_models),
    model_mapping = (
        COALESCE(NULLIF(model_mapping, '')::jsonb, '{}'::jsonb)
        || '{
          "adobe-firefly-nano-banana-pro-1k": "nano-banana-pro",
          "adobe-firefly-nano-banana-pro-2k": "nano-banana-pro",
          "adobe-firefly-nano-banana-pro-4k": "nano-banana-pro",
          "adobe-firefly-nano-banana-1k": "nano-banana",
          "adobe-firefly-nano-banana-2k": "nano-banana",
          "adobe-firefly-nano-banana-4k": "nano-banana",
          "adobe-firefly-nano-banana2-1k": "nano-banana2",
          "adobe-firefly-nano-banana2-2k": "nano-banana2",
          "adobe-firefly-nano-banana2-4k": "nano-banana2",
          "adobe-firefly-gpt-image-2-1k": "gpt-image",
          "adobe-firefly-gpt-image-2-2k": "gpt-image",
          "adobe-firefly-gpt-image-2-4k": "gpt-image"
        }'::jsonb
    )::text
WHERE id = 75;

INSERT INTO models (
    model_name, description, tags, vendor_id, endpoints, status,
    sync_official, image_profile_id, created_time, updated_time
)
SELECT
    v.model_name,
    v.description,
    v.tags,
    2,
    '["openai"]',
    0,
    0,
    v.image_profile_id,
    EXTRACT(EPOCH FROM NOW())::BIGINT,
    EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('adobe-firefly-nano-banana-pro-1k', 'Adobe Firefly Nano Banana Pro 1K 固定档位。', 'image,adobe,firefly,nano-banana,pro,1k', 'image-tpl-adobe2api-1k'),
    ('adobe-firefly-nano-banana-pro-2k', 'Adobe Firefly Nano Banana Pro 2K 固定档位。', 'image,adobe,firefly,nano-banana,pro,2k', 'image-tpl-adobe2api-2k'),
    ('adobe-firefly-nano-banana-pro-4k', 'Adobe Firefly Nano Banana Pro 4K 固定档位。', 'image,adobe,firefly,nano-banana,pro,4k', 'image-tpl-adobe2api-4k'),
    ('adobe-firefly-nano-banana-1k', 'Adobe Firefly Nano Banana 1K 固定档位。', 'image,adobe,firefly,nano-banana,1k', 'image-tpl-adobe2api-1k'),
    ('adobe-firefly-nano-banana-2k', 'Adobe Firefly Nano Banana 2K 固定档位。', 'image,adobe,firefly,nano-banana,2k', 'image-tpl-adobe2api-2k'),
    ('adobe-firefly-nano-banana-4k', 'Adobe Firefly Nano Banana 4K 固定档位。', 'image,adobe,firefly,nano-banana,4k', 'image-tpl-adobe2api-4k'),
    ('adobe-firefly-nano-banana2-1k', 'Adobe Firefly Nano Banana 2 1K 固定档位。', 'image,adobe,firefly,nano-banana2,1k', 'image-tpl-adobe2api-1k'),
    ('adobe-firefly-nano-banana2-2k', 'Adobe Firefly Nano Banana 2 2K 固定档位。', 'image,adobe,firefly,nano-banana2,2k', 'image-tpl-adobe2api-2k'),
    ('adobe-firefly-nano-banana2-4k', 'Adobe Firefly Nano Banana 2 4K 固定档位。', 'image,adobe,firefly,nano-banana2,4k', 'image-tpl-adobe2api-4k'),
    ('adobe-firefly-gpt-image-2-1k', 'Adobe Firefly GPT Image 2 1K 固定档位。', 'image,adobe,firefly,gpt-image,1k', 'image-tpl-adobe2api-1k'),
    ('adobe-firefly-gpt-image-2-2k', 'Adobe Firefly GPT Image 2 2K 固定档位。', 'image,adobe,firefly,gpt-image,2k', 'image-tpl-adobe2api-2k'),
    ('adobe-firefly-gpt-image-2-4k', 'Adobe Firefly GPT Image 2 4K 固定档位。', 'image,adobe,firefly,gpt-image,4k', 'image-tpl-adobe2api-4k')
) AS v(model_name, description, tags, image_profile_id)
WHERE NOT EXISTS (
    SELECT 1 FROM models m WHERE m.model_name = v.model_name AND m.deleted_at IS NULL
);

UPDATE models AS m
SET description = v.description,
    tags = v.tags,
    vendor_id = 2,
    endpoints = '["openai"]',
    status = 0,
    sync_official = 0,
    image_profile_id = v.image_profile_id,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('adobe-firefly-nano-banana-pro-1k', 'Adobe Firefly Nano Banana Pro 1K 固定档位。', 'image,adobe,firefly,nano-banana,pro,1k', 'image-tpl-adobe2api-1k'),
    ('adobe-firefly-nano-banana-pro-2k', 'Adobe Firefly Nano Banana Pro 2K 固定档位。', 'image,adobe,firefly,nano-banana,pro,2k', 'image-tpl-adobe2api-2k'),
    ('adobe-firefly-nano-banana-pro-4k', 'Adobe Firefly Nano Banana Pro 4K 固定档位。', 'image,adobe,firefly,nano-banana,pro,4k', 'image-tpl-adobe2api-4k'),
    ('adobe-firefly-nano-banana-1k', 'Adobe Firefly Nano Banana 1K 固定档位。', 'image,adobe,firefly,nano-banana,1k', 'image-tpl-adobe2api-1k'),
    ('adobe-firefly-nano-banana-2k', 'Adobe Firefly Nano Banana 2K 固定档位。', 'image,adobe,firefly,nano-banana,2k', 'image-tpl-adobe2api-2k'),
    ('adobe-firefly-nano-banana-4k', 'Adobe Firefly Nano Banana 4K 固定档位。', 'image,adobe,firefly,nano-banana,4k', 'image-tpl-adobe2api-4k'),
    ('adobe-firefly-nano-banana2-1k', 'Adobe Firefly Nano Banana 2 1K 固定档位。', 'image,adobe,firefly,nano-banana2,1k', 'image-tpl-adobe2api-1k'),
    ('adobe-firefly-nano-banana2-2k', 'Adobe Firefly Nano Banana 2 2K 固定档位。', 'image,adobe,firefly,nano-banana2,2k', 'image-tpl-adobe2api-2k'),
    ('adobe-firefly-nano-banana2-4k', 'Adobe Firefly Nano Banana 2 4K 固定档位。', 'image,adobe,firefly,nano-banana2,4k', 'image-tpl-adobe2api-4k'),
    ('adobe-firefly-gpt-image-2-1k', 'Adobe Firefly GPT Image 2 1K 固定档位。', 'image,adobe,firefly,gpt-image,1k', 'image-tpl-adobe2api-1k'),
    ('adobe-firefly-gpt-image-2-2k', 'Adobe Firefly GPT Image 2 2K 固定档位。', 'image,adobe,firefly,gpt-image,2k', 'image-tpl-adobe2api-2k'),
    ('adobe-firefly-gpt-image-2-4k', 'Adobe Firefly GPT Image 2 4K 固定档位。', 'image,adobe,firefly,gpt-image,4k', 'image-tpl-adobe2api-4k')
) AS v(model_name, description, tags, image_profile_id)
WHERE m.model_name = v.model_name AND m.deleted_at IS NULL;

DELETE FROM abilities
WHERE channel_id = 75
  AND model LIKE 'adobe-firefly-%';

INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT g.grp, m.model, 75, FALSE, 0, 90
FROM (VALUES
    ('adobe-firefly-nano-banana-pro-1k'), ('adobe-firefly-nano-banana-pro-2k'), ('adobe-firefly-nano-banana-pro-4k'),
    ('adobe-firefly-nano-banana-1k'), ('adobe-firefly-nano-banana-2k'), ('adobe-firefly-nano-banana-4k'),
    ('adobe-firefly-nano-banana2-1k'), ('adobe-firefly-nano-banana2-2k'), ('adobe-firefly-nano-banana2-4k'),
    ('adobe-firefly-gpt-image-2-1k'), ('adobe-firefly-gpt-image-2-2k'), ('adobe-firefly-gpt-image-2-4k')
) AS m(model)
CROSS JOIN (VALUES ('IMAGE'), ('全模型-无claude/gpt'), ('对接专用')) AS g(grp);

COMMIT;

SELECT model_name, status, image_profile_id
FROM models
WHERE model_name LIKE 'adobe-firefly-%'
ORDER BY model_name;

SELECT channel_id, "group", model, enabled
FROM abilities
WHERE channel_id = 75 AND model LIKE 'adobe-firefly-%'
ORDER BY model, "group";
