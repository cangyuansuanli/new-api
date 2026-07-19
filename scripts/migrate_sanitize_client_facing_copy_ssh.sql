-- 清理模型广场 / API 文档中对上游渠道、Adobe2API、Leonardo 等的暴露。
-- contabo: docker exec -i newapi-postgres psql -U root -d new-api < migrate_sanitize_client_facing_copy_ssh.sql

BEGIN;

CREATE OR REPLACE FUNCTION pg_temp.sanitize_client_copy(input text)
RETURNS text
LANGUAGE plpgsql
IMMUTABLE
AS $$
DECLARE
    out text := COALESCE(input, '');
BEGIN
    IF out = '' THEN
        RETURN out;
    END IF;

    out := replace(out, 'Adobe2API Firefly 视频：', 'OpenAI Video 兼容接口：');
    out := replace(out, 'Adobe Firefly ', '');
    out := replace(out, 'Adobe2API ', '');
    out := replace(out, 'Leonardo 订阅号 1300 积分号池，', '');
    out := replace(out, 'Leonardo Seedance', 'Seedance');
    out := replace(out, 'Leonardo 1300 积分号池专用', 'Mini 8 秒特惠专用');
    out := replace(out, 'cy-img1-gpt-image-2', 'gpt-image-2');
    out := replace(out, '固定传模型广场展示名 cy-img1-gpt-image-2。', '必填，传模型广场展示名（{{model}}）。');
    out := replace(out, '勿传上游名 omni-fast-v2v-no-water', '请传 public 名 omni-v2v-no-water');
    out := replace(out, '勿传上游名 omni-fast-v2v', '请传 public 名 omni-v2v');
    out := replace(out, '（Gemini Veo）', '');
    out := replace(out, 'OAIREGBox ', '');
    out := replace(out, '请求由上游网页生成能力执行，不等同于 OpenAI 官方 GPT Image API；仅保证下列基础参数生效。', '支持文生图和上传参考图后的图生图/编辑；下列参数为平台保证生效的基础项。');
    out := replace(out, 'video-tpl-cy-sd4-seedance-async', 'video-tpl-seedance-subscription-async');
    out := replace(out, 'video-tpl-cy-sd5-seedance-933-async', 'video-tpl-seedance-fullref-async');
    out := replace(out, 'video-tpl-cy-sd4-seedance-mini-8s', 'video-tpl-seedance-mini-8s-async');
    out := replace(out, 'seedance-cy-sd4-mini-8s', 'seedance-mini-8s');
    out := replace(out, 'image-tpl-adobe2api-nano-banana-pro-', 'image-tpl-nano-banana-pro-');
    out := replace(out, 'image-tpl-adobe2api-nano-banana2-', 'image-tpl-nano-banana2-');
    out := replace(out, 'image-tpl-adobe2api-gpt-image-2-', 'image-tpl-gpt-image-2-');
    out := replace(out, 'image-tpl-adobe2api-1k', 'image-tpl-nano-banana-tier-1k');
    out := replace(out, 'image-tpl-adobe2api-2k', 'image-tpl-nano-banana-tier-2k');
    out := replace(out, 'image-tpl-adobe2api-4k', 'image-tpl-nano-banana-tier-4k');

    RETURN out;
END;
$$;

-- 1) UI profile id 重命名（对外 pricing 会透出 id）
UPDATE model_ui_param_profiles
SET profile_id = pg_temp.sanitize_client_copy(profile_id),
    validation_key = pg_temp.sanitize_client_copy(validation_key),
    hints = pg_temp.sanitize_client_copy(hints),
    note = pg_temp.sanitize_client_copy(note),
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE profile_id LIKE 'video-tpl-cy-sd%'
   OR profile_id LIKE 'image-tpl-adobe2api%'
   OR validation_key LIKE '%cy-sd4%'
   OR hints ~* 'Adobe2API|Leonardo|Adobe Firefly|上游|Tengda|cy-sd[0-9]|relay 转'
   OR note ~* 'Leonardo|Adobe2API';

UPDATE models
SET video_profile_id = pg_temp.sanitize_client_copy(video_profile_id),
    image_profile_id = pg_temp.sanitize_client_copy(image_profile_id),
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE video_profile_id LIKE 'video-tpl-cy-sd%'
   OR image_profile_id LIKE 'image-tpl-adobe2api%';

-- 2) 模型广场 description
UPDATE models
SET description = pg_temp.sanitize_client_copy(description),
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE deleted_at IS NULL
  AND description ~* 'Adobe Firefly|Adobe2API|Leonardo|OAIREGBox|cy-img1-|omni-fast-v2v';

-- 3) api_doc JSON 文本
UPDATE models
SET api_doc = pg_temp.sanitize_client_copy(api_doc),
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE deleted_at IS NULL
  AND api_doc ~* 'Adobe Firefly|Adobe2API|Leonardo|OAIREGBox|cy-img1-|omni-fast-v2v|cy-sd|adobe2api|上游';

COMMIT;

-- 验收：公开 alias 对应模型不应再含上游关键词
SELECT m.model_name,
       left(m.description, 80) AS description_preview
FROM models m
JOIN model_public_aliases a ON a.internal_name = m.model_name AND a.deleted_at IS NULL
WHERE m.deleted_at IS NULL
  AND (
    m.description ~* 'Adobe2API|Leonardo|OAIREGBox|cy-img1-|omni-fast-v2v|Adobe Firefly'
    OR m.api_doc ~* 'Adobe2API|Leonardo|OAIREGBox|cy-img1-|omni-fast-v2v|cy-sd[0-9]|adobe2api'
  )
ORDER BY m.model_name;
