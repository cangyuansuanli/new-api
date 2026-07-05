-- Leonardo Seedance UI v3：去掉 internal 名 modelIncludes，改由画布按 public 名 seedance-2.0 精确识别
-- （substring 规则会误伤 seedance-2.0-480p 等经济档，故不在 DB 写 seedance-2.0 前缀规则）
-- docker exec -i newapi-postgres psql -U root -d new-api < migrate_leonardo_seedance_ui_v3_public.sql

BEGIN;

UPDATE model_ui_param_profiles SET
    option_rules = (
        SELECT COALESCE(jsonb_agg(elem ORDER BY elem), '[]'::jsonb)::text
        FROM (
            SELECT elem
            FROM jsonb_array_elements(COALESCE(option_rules::jsonb, '[]'::jsonb)) AS elem
            WHERE COALESCE(elem->'disabledWhen'->>'modelIncludes', '') !~ '(leonardo-seedance|cy-sd4-seedance)'
        ) kept(elem)
    ),
    hints = (
        SELECT COALESCE(jsonb_agg(elem ORDER BY elem), '[]'::jsonb)::text
        FROM (
            SELECT elem
            FROM jsonb_array_elements(COALESCE(hints::jsonb, '[]'::jsonb)) AS elem
            WHERE COALESCE(elem->'when'->>'modelIncludes', '') !~ '(leonardo-seedance|cy-sd4-seedance)'
        ) kept(elem)
    ),
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video' AND profile_id = 'video-tpl-seedance-async';

COMMIT;

SELECT jsonb_array_length(option_rules::jsonb) AS rule_count,
       jsonb_array_length(hints::jsonb) AS hint_count
FROM model_ui_param_profiles
WHERE capability = 'video' AND profile_id = 'video-tpl-seedance-async';
