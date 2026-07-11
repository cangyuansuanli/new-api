-- Adobe2API image SKUs, contract/price gate + activation phase.
-- Run only after NewAPI code rollout and real candidate requests succeed.

BEGIN;

DO $$
DECLARE
    prices jsonb;
    family text;
    model_count integer;
    ability_count integer;
BEGIN
    SELECT count(*) INTO model_count
    FROM models
    WHERE model_name LIKE 'adobe-firefly-%'
      AND deleted_at IS NULL;
    IF model_count <> 12 THEN
        RAISE EXCEPTION 'expected 12 Adobe Firefly image models, found %', model_count;
    END IF;

    SELECT count(*) INTO ability_count
    FROM abilities
    WHERE channel_id = 75
      AND model LIKE 'adobe-firefly-%';
    IF ability_count <> 36 THEN
        RAISE EXCEPTION 'expected 36 disabled Adobe abilities, found %', ability_count;
    END IF;

    SELECT value::jsonb INTO prices FROM options WHERE key = 'ModelPrice';
    IF prices IS NULL OR NOT prices ?& ARRAY[
        'adobe-firefly-nano-banana-pro-1k', 'adobe-firefly-nano-banana-pro-2k', 'adobe-firefly-nano-banana-pro-4k',
        'adobe-firefly-nano-banana-1k', 'adobe-firefly-nano-banana-2k', 'adobe-firefly-nano-banana-4k',
        'adobe-firefly-nano-banana2-1k', 'adobe-firefly-nano-banana2-2k', 'adobe-firefly-nano-banana2-4k',
        'adobe-firefly-gpt-image-2-1k', 'adobe-firefly-gpt-image-2-2k', 'adobe-firefly-gpt-image-2-4k'
    ] THEN
        RAISE EXCEPTION 'configure all 12 Adobe Firefly ModelPrice keys before activation';
    END IF;

    FOREACH family IN ARRAY ARRAY[
        'adobe-firefly-nano-banana-pro',
        'adobe-firefly-nano-banana',
        'adobe-firefly-nano-banana2',
        'adobe-firefly-gpt-image-2'
    ] LOOP
        IF (prices->>(family || '-1k'))::numeric <= 0
           OR (prices->>(family || '-1k'))::numeric >= (prices->>(family || '-2k'))::numeric
           OR (prices->>(family || '-2k'))::numeric >= (prices->>(family || '-4k'))::numeric THEN
            RAISE EXCEPTION 'ModelPrice must satisfy 0 < %-1k < %-2k < %-4k', family, family, family;
        END IF;
    END LOOP;
END $$;

UPDATE models
SET status = 1,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE model_name LIKE 'adobe-firefly-%'
  AND deleted_at IS NULL;

UPDATE abilities
SET enabled = TRUE
WHERE channel_id = 75
  AND model LIKE 'adobe-firefly-%';

COMMIT;

-- Rolling-restart NewAPI after this transaction so the public-name registry is refreshed.
SELECT model_name, status, image_profile_id
FROM models
WHERE model_name LIKE 'adobe-firefly-%'
ORDER BY model_name;

SELECT "group", count(*) AS enabled_abilities
FROM abilities
WHERE channel_id = 75
  AND model LIKE 'adobe-firefly-%'
  AND enabled = TRUE
GROUP BY "group"
ORDER BY "group";
