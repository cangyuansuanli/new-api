-- flux-pro-2：绑定 image profile + BFL 厂商（源站 SSH 执行）
-- contabo: docker exec -i newapi-postgres psql -U root -d new-api < migrate_flux_pro_2_ssh.sql
-- 完整 profile + api_doc 请再执行 seed_flux_pro_2_api_doc.py

BEGIN;

INSERT INTO vendors (name, description, icon, status, created_time, updated_time)
SELECT
    'Black Forest Labs',
    'Black Forest Labs（BFL），FLUX 系列文生图/图生图模型厂商。',
    'Flux',
    1,
    EXTRACT(EPOCH FROM NOW())::BIGINT,
    EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE NOT EXISTS (
    SELECT 1 FROM vendors WHERE name = 'Black Forest Labs' AND deleted_at IS NULL
);

UPDATE vendors SET
    description = 'Black Forest Labs（BFL），FLUX 系列文生图/图生图模型厂商。',
    icon = 'Flux',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE name = 'Black Forest Labs' AND deleted_at IS NULL;

UPDATE models SET
    image_profile_id = 'image-tpl-flux-pro-2',
    vendor_id = (SELECT id FROM vendors WHERE name = 'Black Forest Labs' AND deleted_at IS NULL LIMIT 1),
    icon = 'Flux',
    description = 'Black Forest Labs FLUX.2 Pro 文生图/图生图。写实摄影与电影感、细节丰富，适合产品图、场景与人像；支持多参考图编辑。OpenAI 兼容 /v1/images/generations，单边 256–1920px，按次 ¥0.08。',
    tags = 'image,flux,bfl,photorealistic,image-edit',
    endpoints = '{"openai-image":{"path":"/v1/images/generations","method":"POST"}}',
    sync_official = 0,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE model_name = 'flux-pro-2' AND deleted_at IS NULL;

COMMIT;

SELECT m.model_name, m.vendor_id, v.name AS vendor_name, m.icon, m.description, m.tags
FROM models m
LEFT JOIN vendors v ON v.id = m.vendor_id
WHERE m.model_name = 'flux-pro-2' AND m.deleted_at IS NULL;
