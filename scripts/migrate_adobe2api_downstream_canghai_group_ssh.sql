-- Allow the dedicated Canghai downstream service account to use Adobe2API video models.
-- Safe to re-run on the production PostgreSQL database.
--
-- Run on contabo:
--   docker exec -i newapi-postgres psql -v ON_ERROR_STOP=1 -U root -d new-api \
--     < scripts/migrate_adobe2api_downstream_canghai_group_ssh.sql

BEGIN;

-- Register the internal service-account group in GroupRatio so Root can see
-- and manage it. It intentionally stays out of UserUsableGroups.
INSERT INTO options (key, value)
VALUES ('GroupRatio', '{"downstream-canghai":1}')
ON CONFLICT (key) DO UPDATE
SET value = (
    COALESCE(NULLIF(BTRIM(options.value), ''), '{}')::jsonb
    || '{"downstream-canghai":1}'::jsonb
)::text;

UPDATE channels
SET "group" = concat_ws(',', NULLIF(BTRIM("group"), ''), 'downstream-canghai')
WHERE id = 75
  AND NOT ('downstream-canghai' = ANY(string_to_array("group", ',')));

INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT 'downstream-canghai', v.model, 75, TRUE, 90, 90
FROM (VALUES
    ('adobe-sora2'),
    ('adobe-sora2-pro'),
    ('adobe-veo31'),
    ('adobe-veo31-ref'),
    ('adobe-veo31-fast')
) AS v(model)
WHERE NOT EXISTS (
    SELECT 1
    FROM abilities a
    WHERE a.channel_id = 75
      AND a."group" = 'downstream-canghai'
      AND a.model = v.model
);

UPDATE abilities
SET enabled = TRUE
WHERE channel_id = 75
  AND "group" = 'downstream-canghai'
  AND model IN (
      'adobe-sora2',
      'adobe-sora2-pro',
      'adobe-veo31',
      'adobe-veo31-ref',
      'adobe-veo31-fast'
  );

COMMIT;

SELECT key, value
FROM options
WHERE key IN ('GroupRatio', 'UserUsableGroups')
ORDER BY key;

SELECT id, name, status, "group"
FROM channels
WHERE id = 75;

SELECT channel_id, "group", model, enabled, priority, weight
FROM abilities
WHERE channel_id = 75
  AND "group" = 'downstream-canghai'
  AND model IN (
      'adobe-sora2',
      'adobe-sora2-pro',
      'adobe-veo31',
      'adobe-veo31-ref',
      'adobe-veo31-fast'
  )
ORDER BY model;
