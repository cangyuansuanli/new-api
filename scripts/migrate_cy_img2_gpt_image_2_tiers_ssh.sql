-- Greek2API channel 48: provider-neutral GPT Image 2 fixed-resolution SKUs.
-- Run only after the matching relay image has been deployed.
-- contabo: docker exec -i newapi-postgres psql -v ON_ERROR_STOP=1 -U root -d new-api < migrate_cy_img2_gpt_image_2_tiers_ssh.sql

BEGIN;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM channels
        WHERE id = 48
          AND type = 1
          AND base_url = 'https://www.greek2api.com'
          AND length(COALESCE(key, '')) > 0
    ) THEN
        RAISE EXCEPTION 'channel 48 does not match the audited OpenAI-compatible image upstream';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM model_ui_param_profiles
        WHERE capability = 'image'
          AND profile_id = 'image-tpl-gpt-image-2-tiered'
          AND deleted_at IS NULL
    ) THEN
        RAISE EXCEPTION 'seed image-tpl-gpt-image-2-tiered before activating channel 48';
    END IF;
END $$;

INSERT INTO model_channel_prefixes (prefix, note, enabled, sort_order, created_time, updated_time)
VALUES ('cy-img2-', '生图线路 B', TRUE, 131, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (prefix) DO UPDATE SET
    note = EXCLUDED.note,
    enabled = TRUE,
    sort_order = EXCLUDED.sort_order,
    updated_time = EXCLUDED.updated_time,
    deleted_at = NULL;

UPDATE channels
SET name = 'greek2api-gpt-image-2-fallback',
    models = 'cy-img2-gpt-image-2-1k,cy-img2-gpt-image-2-2k,cy-img2-gpt-image-2-4k',
    model_mapping = '{
      "cy-img2-gpt-image-2-1k": "gpt-image-2",
      "cy-img2-gpt-image-2-2k": "gpt-image-2",
      "cy-img2-gpt-image-2-4k": "gpt-image-2"
    }',
    "group" = 'IMAGE,全模型-无claude/gpt',
    status = 1,
    priority = 90,
    weight = 90,
    param_override = ''
WHERE id = 48;

-- Remove the neutral SKUs from retired providers so re-enabling an old channel
-- cannot recreate a competing route for the same internal model.
UPDATE channels
SET models = array_to_string(
        ARRAY(
            SELECT model_name
            FROM unnest(string_to_array(COALESCE(models, ''), ',')) AS model_name
            WHERE btrim(model_name) NOT IN (
                'cy-img2-gpt-image-2-1k',
                'cy-img2-gpt-image-2-2k',
                'cy-img2-gpt-image-2-4k'
            )
        ),
        ','
    ),
    model_mapping = (
        COALESCE(NULLIF(model_mapping, ''), '{}')::jsonb
        - 'cy-img2-gpt-image-2-1k'
        - 'cy-img2-gpt-image-2-2k'
        - 'cy-img2-gpt-image-2-4k'
    )::text
WHERE id IN (72, 73, 74);

INSERT INTO models (
    model_name, description, tags, vendor_id, endpoints, status,
    sync_official, image_profile_id, created_time, updated_time
)
SELECT
    v.model_name, v.description, v.tags, 2, '["openai"]', 1,
    0, 'image-tpl-gpt-image-2-tiered',
    EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('cy-img2-gpt-image-2-1k', 'GPT Image 2 1K 固定档位。', 'image,gpt-image,1k'),
    ('cy-img2-gpt-image-2-2k', 'GPT Image 2 2K 固定档位。', 'image,gpt-image,2k'),
    ('cy-img2-gpt-image-2-4k', 'GPT Image 2 4K 固定档位。', 'image,gpt-image,4k')
) AS v(model_name, description, tags)
WHERE NOT EXISTS (
    SELECT 1 FROM models m WHERE m.model_name = v.model_name AND m.deleted_at IS NULL
);

UPDATE models AS m
SET description = v.description,
    tags = v.tags,
    vendor_id = 2,
    endpoints = '["openai"]',
    status = 1,
    sync_official = 0,
    image_profile_id = 'image-tpl-gpt-image-2-tiered',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('cy-img2-gpt-image-2-1k', 'GPT Image 2 1K 固定档位。', 'image,gpt-image,1k'),
    ('cy-img2-gpt-image-2-2k', 'GPT Image 2 2K 固定档位。', 'image,gpt-image,2k'),
    ('cy-img2-gpt-image-2-4k', 'GPT Image 2 4K 固定档位。', 'image,gpt-image,4k')
) AS v(model_name, description, tags)
WHERE m.model_name = v.model_name AND m.deleted_at IS NULL;

DELETE FROM model_public_aliases
WHERE internal_name IN (
    'cy-img2-gpt-image-2-1k', 'cy-img2-gpt-image-2-2k', 'cy-img2-gpt-image-2-4k'
);

DELETE FROM abilities
WHERE channel_id = 48
   OR model IN (
       'cy-img2-gpt-image-2-1k',
       'cy-img2-gpt-image-2-2k',
       'cy-img2-gpt-image-2-4k'
   );

INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT g.grp, m.model, 48, TRUE, 90, 90
FROM (VALUES
    ('cy-img2-gpt-image-2-1k'),
    ('cy-img2-gpt-image-2-2k'),
    ('cy-img2-gpt-image-2-4k')
) AS m(model)
CROSS JOIN (VALUES ('IMAGE'), ('全模型-无claude/gpt')) AS g(grp);

UPDATE model_ui_param_profiles AS p
SET deleted_at = NOW(),
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE p.capability = 'image'
  AND p.profile_id IN ('image-tpl-geek2-4k', 'image-tpl-gulie-2k')
  AND p.deleted_at IS NULL
  AND NOT EXISTS (
      SELECT 1
      FROM models AS m
      WHERE m.image_profile_id = p.profile_id
        AND m.deleted_at IS NULL
  );

COMMIT;

SELECT id, name, status, "group", models, model_mapping
FROM channels WHERE id = 48;
SELECT "group", model, channel_id, enabled, priority, weight
FROM abilities WHERE channel_id = 48 ORDER BY model, "group";
