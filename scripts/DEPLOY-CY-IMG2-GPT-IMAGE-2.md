# Greek2API GPT Image 2 fallback rollout

Production model names stay provider-neutral:

- internal: `cy-img2-gpt-image-2-{1k,2k,4k}`
- public: `gpt-image-2-{1k,2k,4k}`
- upstream: `gpt-image-2`

The provider-specific outbound contract lives in `relay/imagevendor/vendor_greek2api.go` and only matches channel 48.

## Order

1. Deploy the matching New API image through `.github/workflows/cangyuan-prod.yml`.
2. Copy `scripts/seed_cy_img2_gpt_image_2_tiers.py` to the source host and run it once to create `image-tpl-gpt-image-2-tiered`.
3. Execute `scripts/migrate_cy_img2_gpt_image_2_tiers_ssh.sql` against `newapi-postgres`.
4. Run the seed script again to populate API docs for the newly created/bound model rows.
5. Wait at least one channel/options cache sync interval. Do not restart production containers.

## Canary

Verify one 1K generation first, then 2K/4K and multipart edits:

- public requests use only `gpt-image-2-{1k,2k,4k}`;
- a client `size` ratio remains unchanged until New API selects channel 48;
- channel 48 outbound `16:9` becomes `1280x720`, `2560x1440`, or `3840x2160`;
- upstream receives `model: gpt-image-2`;
- completed URLs are rehosted to R2;
- billing uses the selected public SKU price.

The seed preserves existing `cy-img2` prices. At the audited production state this means 2K `$0.075` and 4K `$0.15`; the new 1K SKU falls back to `$0.025`.
