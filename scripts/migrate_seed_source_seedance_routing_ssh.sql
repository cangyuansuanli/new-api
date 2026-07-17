-- Route the existing public Seedance products through channel 75 without
-- creating another public/internal model pair. The source container exposes
-- cy-sd5-*; NewAPI continues to use cy-sd4-* internally so public names stay
-- seedance-2.0 and seedance-2.0-fast.
--
-- Run on contabo:
--   docker exec -i newapi-postgres psql -v ON_ERROR_STOP=1 -U root -d new-api \
--     < scripts/migrate_seed_source_seedance_routing_ssh.sql

BEGIN;

UPDATE channels
SET models = array_to_string(
        ARRAY(
            SELECT item
            FROM unnest(string_to_array(models, ',')) AS item
            WHERE btrim(item) NOT IN (
                'cy-sd4-seedance-2.0',
                'cy-sd4-seedance-2.0-fast',
                'cy-sd5-seedance-2.0',
                'cy-sd5-seedance-2.0-fast'
            )
        ) || ARRAY[
            'cy-sd4-seedance-2.0',
            'cy-sd4-seedance-2.0-fast'
        ],
        ','
    ),
    model_mapping = (
        (COALESCE(NULLIF(model_mapping, ''), '{}')::jsonb
            - 'cy-sd5-seedance-2.0'
            - 'cy-sd5-seedance-2.0-fast')
        || jsonb_build_object(
            'cy-sd4-seedance-2.0', 'cy-sd5-seedance-2.0',
            'cy-sd4-seedance-2.0-fast', 'cy-sd5-seedance-2.0-fast'
        )
    )::text
WHERE id = 75;

DELETE FROM abilities
WHERE channel_id = 75
  AND model IN (
      'cy-sd4-seedance-2.0',
      'cy-sd4-seedance-2.0-fast',
      'cy-sd5-seedance-2.0',
      'cy-sd5-seedance-2.0-fast'
  );

INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT groups.name, models.name, 75, TRUE, 100, 100
FROM (VALUES
    ('IMAGE'),
    ('VIDEO'),
    ('全模型-无claude/gpt'),
    ('对接专用')
) AS groups(name)
CROSS JOIN (VALUES
    ('cy-sd4-seedance-2.0'),
    ('cy-sd4-seedance-2.0-fast')
) AS models(name);

COMMIT;

SELECT id, models, model_mapping
FROM channels
WHERE id = 75;

SELECT channel_id, "group", model, enabled, priority, weight
FROM abilities
WHERE channel_id = 75
  AND model IN ('cy-sd4-seedance-2.0', 'cy-sd4-seedance-2.0-fast')
ORDER BY model, "group";
