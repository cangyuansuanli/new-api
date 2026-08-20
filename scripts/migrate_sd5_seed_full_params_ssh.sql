-- contabo: docker exec -i newapi-postgres psql -v ON_ERROR_STOP=1 -U root -d new-api < migrate_sd5_seed_full_params_ssh.sql
-- Align the existing SD5 UI profile and API docs with Adobe2API's deployed contract.

BEGIN;

UPDATE model_ui_param_profiles
SET params = jsonb_set(
        COALESCE(NULLIF(params, ''), '{}')::jsonb,
        '{seed}',
        '{"enabled":true,"min":0,"max":2147483647,"hint":"可选整数种子；相同输入可用于复现，显式 0 也会透传。"}'::jsonb,
        TRUE
    )::text,
    reference_limits = jsonb_set(
        jsonb_set(
            jsonb_set(
                COALESCE(NULLIF(reference_limits, ''), '{}')::jsonb,
                '{total}',
                '12'::jsonb,
                TRUE
            ),
            '{fullReferenceMode,descriptionWithImages}',
            to_jsonb('最多 9 图 / 3 视频 / 3 音频，三类合计不超过 12'::text),
            TRUE
        ),
        '{validationHint}',
        to_jsonb('全能参考最多 9 图、3 视频、3 音频，三类合计不超过 12；使用参考音频时至少搭配 1 张参考图；首尾帧与全能参考互斥。'::text),
        TRUE
    )::text,
    hints = '[{"text":"Seedance 2.0：480p / 720p、4–15 秒任意整数、可选整数 seed；支持 9 图 / 3 视频 / 3 音频且合计不超过 12，也支持首尾帧。"}]',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE profile_id = 'video-tpl-cy-sd5-seedance-933-async'
  AND capability = 'video'
  AND deleted_at IS NULL;

DO $migration$
DECLARE
    item RECORD;
    doc JSONB;
BEGIN
    FOR item IN
        SELECT id, api_doc
        FROM models
        WHERE model_name IN ('cy-sd5-seedance-2.0', 'cy-sd5-seedance-2.0-fast')
          AND deleted_at IS NULL
    LOOP
        doc := item.api_doc::jsonb;
        doc := jsonb_set(
            doc,
            '{intro}',
            to_jsonb(replace(
                replace(
                    doc->>'intro',
                    '全能参考最多 9 图、3 视频、3 音频',
                    '全能参考最多 9 图、3 视频、3 音频且合计不超过 12'
                ),
                '不支持 seed、n 或 response_format。',
                '支持可选整数 seed；不支持 n 或 response_format。'
            )),
            TRUE
        );
        doc := jsonb_set(doc, '{basic_request_json,seed}', '0'::jsonb, TRUE);
        doc := jsonb_set(doc, '{request_json,seed}', '0'::jsonb, TRUE);
        doc := jsonb_set(
            doc,
            '{request_json,reference_videos}',
            jsonb_build_array(
                doc#>'{request_json,reference_videos,0}',
                doc#>'{request_json,reference_videos,1}'
            ),
            TRUE
        );
        doc := jsonb_set(
            doc,
            '{request_json,reference_audios}',
            jsonb_build_array(doc#>'{request_json,reference_audios,0}'),
            TRUE
        );
        doc := jsonb_set(doc, '{examples,0,request_json,seed}', '0'::jsonb, TRUE);
        doc := jsonb_set(doc, '{examples,1,request_json,seed}', '0'::jsonb, TRUE);
        doc := jsonb_set(doc, '{examples,2,request_json,seed}', '0'::jsonb, TRUE);
        doc := jsonb_set(
            doc,
            '{examples,2,title}',
            to_jsonb('9 图 + 2 视频 + 1 音频全能参考（总计 12）'::text),
            TRUE
        );
        doc := jsonb_set(
            doc,
            '{examples,2,request_json,reference_videos}',
            jsonb_build_array(
                doc#>'{examples,2,request_json,reference_videos,0}',
                doc#>'{examples,2,request_json,reference_videos,1}'
            ),
            TRUE
        );
        doc := jsonb_set(
            doc,
            '{examples,2,request_json,reference_audios}',
            jsonb_build_array(doc#>'{examples,2,request_json,reference_audios,0}'),
            TRUE
        );
        doc := jsonb_set(
            doc,
            '{generation_modes,2,notes}',
            to_jsonb('最多 9 图、3 视频、3 音频，三类合计不超过 12；URL 素材需公网可访问。'::text),
            TRUE
        );
        IF NOT EXISTS (
            SELECT 1
            FROM jsonb_array_elements(doc->'params') AS parameter
            WHERE parameter->>'name' = 'seed'
        ) THEN
            doc := jsonb_set(
                doc,
                '{params}',
                (doc->'params') || jsonb_build_array(jsonb_build_object(
                    'name', 'seed',
                    'description', '可选整数种子；相同输入可用于复现，显式 0 也会透传。'
                )),
                TRUE
            );
        END IF;

        UPDATE models
        SET api_doc = doc::text,
            updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
        WHERE id = item.id;
    END LOOP;
END
$migration$;

COMMIT;
