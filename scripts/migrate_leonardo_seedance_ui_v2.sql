-- Leonardo Seedance UI 参数 v2（源站执行）
-- 多模态已支持；成片仅 480p/720p；移除旧的参考音视频禁用规则
-- docker exec -i newapi-postgres psql -U root -d new-api < migrate_leonardo_seedance_ui_v2.sql

BEGIN;

UPDATE models AS m SET
    description = v.description,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('leonardo-seedance-2.0', 'Leonardo 订阅号 Seedance 2.0。文生/图生/多模态/首尾帧，标准 480p / HD 720p，4–15 秒。'),
    ('leonardo-seedance-2.0-fast', 'Leonardo 订阅号 Seedance 2.0 Fast。更快出片，参数同标准版。'),
    ('cy-sd4-seedance-2.0', 'Leonardo 订阅号 Seedance 2.0。文生/图生/多模态/首尾帧，标准 480p / HD 720p，4–15 秒。'),
    ('cy-sd4-seedance-2.0-fast', 'Leonardo 订阅号 Seedance 2.0 Fast。更快出片，参数同标准版。')
) AS v(model_name, description)
WHERE m.model_name = v.model_name AND m.deleted_at IS NULL;

UPDATE model_ui_param_profiles SET
    option_rules = (
        SELECT COALESCE(jsonb_agg(elem ORDER BY elem), '[]'::jsonb)::text
        FROM (
            SELECT elem
            FROM jsonb_array_elements(COALESCE(option_rules::jsonb, '[]'::jsonb)) AS elem
            WHERE NOT (
                elem->>'param' IN ('reference_videos', 'reference_audios')
                AND COALESCE(elem->'disabledWhen'->>'modelIncludes', '') ~ '(leonardo-seedance|cy-sd4-seedance)'
            )
        ) kept(elem)
    ),
    hints = (
        SELECT COALESCE(jsonb_agg(elem ORDER BY elem), '[]'::jsonb)::text
        FROM (
            SELECT elem
            FROM jsonb_array_elements(COALESCE(hints::jsonb, '[]'::jsonb)) AS elem
            WHERE COALESCE(elem->'when'->>'modelIncludes', '') !~ '(leonardo-seedance|cy-sd4-seedance)'
               OR (
                 COALESCE(elem->>'text', '') NOT LIKE '%933%'
                 AND COALESCE(elem->>'text', '') NOT LIKE '%不支持参考视频%'
                 AND COALESCE(elem->>'text', '') NOT LIKE '%不支持参考音频%'
               )
        ) kept(elem)
    ),
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video' AND profile_id = 'video-tpl-seedance-async';

UPDATE model_ui_param_profiles SET
    option_rules = (
        SELECT COALESCE(jsonb_agg(DISTINCT elem ORDER BY elem), '[]'::jsonb)::text
        FROM (
            SELECT jsonb_array_elements(COALESCE(option_rules::jsonb, '[]'::jsonb)) AS elem
            UNION ALL
            SELECT * FROM jsonb_array_elements('[
                {"param":"resolution","value":"1080p","disabledWhen":{"modelIncludes":"leonardo-seedance"}},
                {"param":"resolution","value":"1080p","disabledWhen":{"modelIncludes":"cy-sd4-seedance"}},
                {"param":"resolution","value":"4k","disabledWhen":{"modelIncludes":"leonardo-seedance"}},
                {"param":"resolution","value":"4k","disabledWhen":{"modelIncludes":"cy-sd4-seedance"}}
            ]'::jsonb)
        ) merged(elem)
    ),
    hints = (
        SELECT COALESCE(jsonb_agg(DISTINCT elem ORDER BY elem), '[]'::jsonb)::text
        FROM (
            SELECT jsonb_array_elements(COALESCE(hints::jsonb, '[]'::jsonb)) AS elem
            UNION ALL
            SELECT * FROM jsonb_array_elements('[
                {"text":"Leonardo 订阅号：标准 480p（16:9=864×496）/ HD 720p（1280×720）；多模态 4图/3视频（总时长≤15s）/1音频；与首尾帧互斥。","when":{"modelIncludes":"leonardo-seedance"}},
                {"text":"Leonardo 订阅号：标准 480p（16:9=864×496）/ HD 720p（1280×720）；多模态 4图/3视频（总时长≤15s）/1音频；与首尾帧互斥。","when":{"modelIncludes":"cy-sd4-seedance"}}
            ]'::jsonb)
        ) merged(elem)
    ),
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video' AND profile_id = 'video-tpl-seedance-async';

COMMIT;

SELECT jsonb_array_length(option_rules::jsonb) AS rule_count
FROM model_ui_param_profiles
WHERE capability = 'video' AND profile_id = 'video-tpl-seedance-async';
