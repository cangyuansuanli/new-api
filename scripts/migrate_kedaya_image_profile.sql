-- Kedaya 生图：绑定 image-tpl-kedaya-1k profile（需先导入 seed_data/model_ui_params_image.json 中的新 profile）
UPDATE models
SET image_profile_id = 'image-tpl-kedaya-1k', updated_time = EXTRACT(EPOCH FROM NOW())::bigint
WHERE model_name LIKE 'kedaya-%' AND deleted_at IS NULL;
