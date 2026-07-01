-- 腾达 Seedance 2.0：绑定统一 seedance profile（源站 SSH 执行）
-- contabo: docker exec -i newapi-postgres psql -U root -d new-api < migrate_tengda_seedance_2.0_ssh.sql

BEGIN;

-- 1. 绑定统一 profile（与 oairegbox / ctlove Seedance 2.0 相同）
UPDATE models SET
    video_profile_id = 'video-tpl-seedance-async',
    description = 'Seedance 2.0 特惠。文生/图生/933 全能参考/首尾帧，480P/720P，4–15 秒。',
    tags = 'video,seedance,tengd,geeknow,special-offer',
    vendor_id = 6,
    endpoints = '{"openai-video":{"path":"/v1/videos","method":"POST"}}',
    sync_official = 0,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE model_name = 'tengd-Seedance-2.0' AND deleted_at IS NULL;

-- 2. 统一 profile：tengd 仅 480P/720P
UPDATE model_ui_param_profiles SET
    option_rules = (
        SELECT COALESCE(jsonb_agg(DISTINCT elem ORDER BY elem), '[]'::jsonb)::text
        FROM (
            SELECT jsonb_array_elements(COALESCE(option_rules::jsonb, '[]'::jsonb)) AS elem
            UNION ALL
            SELECT * FROM jsonb_array_elements('[
                {"param":"resolution","value":"1080p","disabledWhen":{"modelIncludes":"tengd-"}},
                {"param":"resolution","value":"4k","disabledWhen":{"modelIncludes":"tengd-"}}
            ]'::jsonb)
        ) merged(elem)
    ),
    hints = (
        SELECT COALESCE(jsonb_agg(DISTINCT elem ORDER BY elem), '[]'::jsonb)::text
        FROM (
            SELECT jsonb_array_elements(COALESCE(hints::jsonb, '[]'::jsonb)) AS elem
            UNION ALL
            SELECT * FROM jsonb_array_elements('[
                {"text":"Seedance 2.0 特惠：480P/720P，4–15 秒；933 全能参考与首尾帧；模式由素材字段自动判定。","when":{"modelIncludes":"tengd-"}}
            ]'::jsonb)
        ) merged(elem)
    ),
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video' AND profile_id = 'video-tpl-seedance-async';

-- 3. 清理已废弃的独立 profile
DELETE FROM model_ui_param_profiles
WHERE capability = 'video' AND profile_id = 'video-tpl-tengda-seedance-2.0-async';

COMMIT;

SELECT model_name, video_profile_id, tags
FROM models
WHERE model_name = 'tengd-Seedance-2.0' AND deleted_at IS NULL;
