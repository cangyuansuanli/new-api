-- RETIRED: cy-img2-gpt-image-2-2k no longer belongs to Gulie channel 72.
-- Use migrate_cy_img2_gpt_image_2_tiers_ssh.sql after deploying the
-- Greek2API channel-aware image vendor adapter.

DO $$
BEGIN
    RAISE EXCEPTION 'retired migration: use migrate_cy_img2_gpt_image_2_tiers_ssh.sql';
END $$;
