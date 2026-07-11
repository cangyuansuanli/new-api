-- Deprecated: Adobe image products must not reuse Manju internal models.
-- Reuse prevents independent ModelPrice/UI semantics and can mix channel 70/75 routing.
-- Use the two-phase dedicated SKU migrations instead:
--   1. migrate_adobe2api_image_skus_expand_ssh.sql
--   2. migrate_adobe2api_image_skus_activate_ssh.sql

DO $$
BEGIN
    RAISE EXCEPTION 'deprecated unsafe migration; use migrate_adobe2api_image_skus_expand_ssh.sql';
END $$;
