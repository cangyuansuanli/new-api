-- 纠正已上架 Leonardo Seedance 2.5 的 Profile 与客户 API 文档素材上限。
-- cy-origin: docker exec -i newapi-postgres psql -U root -d new-api < migrate_leonardo_seedance_25_reference_limits_ssh.sql

BEGIN;

UPDATE model_ui_param_profiles
SET reference_limits = '{"images":30,"videos":10,"audios":10,"imageMaxBytes":26214400,"videoMaxBytes":209715200,"audioMaxBytes":15728640,"video":{"maxDurationMs":30200,"totalMaxDurationMs":30200},"audio":{"maxDurationMs":30200,"totalMaxDurationMs":30200},"fullReferenceMode":{"label":"多模态","descriptionWithImages":"多模态：图 + 可选视频/音频"},"validationHint":"参考图 png/jpg/webp ≤25MB（最多 30）；参考视频 mp4/mov ≤200MB，最多 10 条且单条/总时长 ≤30.2 秒，宽高各 720–2160px、24–60 FPS；参考音频 mp3/wav ≤15MB、最多 10 条且单条/总时长 ≤30.2 秒。","showTempMediaHint":true,"prependReferenceGuide":true}',
    hints = '[{"text":"模型名决定固定清晰度，按秒计费；480p 最长 30 秒；720p 无参考视频最长 29 秒，带参考视频最长 18 秒。"},{"text":"最多 30 图 / 10 视频 / 10 音频；参考视频和音频单条及合计均不超过 30.2 秒。"}]',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video'
  AND profile_id = 'video-tpl-seedance-2.5-subscription-async'
  AND deleted_at IS NULL;

UPDATE models AS m
SET api_doc = jsonb_build_object(
        'dispatch_mode', 'async',
        'intro', format(
            'Seedance 2.5 %s 视频，模型名锁定清晰度，按成片秒数计费。输出时长 4–%s 秒。%s',
            v.resolution, v.max_seconds, v.pool_limit
        ),
        'endpoints', jsonb_build_array(
            jsonb_build_object('method', 'POST', 'path', '{{base}}/videos', 'description', '创建异步视频任务。'),
            jsonb_build_object('method', 'GET', 'path', '{{base}}/videos/{task_id}', 'description', '查询任务状态和成片地址。'),
            jsonb_build_object('method', 'GET', 'path', '{{base}}/videos/{task_id}/content', 'description', '下载已完成任务的成片。')
        ),
        'params', jsonb_build_array(
            jsonb_build_object('name', 'model', 'description', format('必填，传模型广场名称 %s。', v.public_name)),
            jsonb_build_object('name', 'prompt', 'description', '必填，视频内容描述，最多 5000 个 Unicode 字符。'),
            jsonb_build_object('name', 'duration', 'description', format('整数 4–%s 秒，默认 8。%s', v.max_seconds, v.duration_note)),
            jsonb_build_object('name', 'aspect_ratio', 'description', '21:9、16:9、4:3、1:1、3:4 或 9:16。'),
            jsonb_build_object('name', 'resolution', 'description', format('由当前模型固定为 %s，请求值不会跨档。', v.resolution)),
            jsonb_build_object('name', 'generate_audio', 'description', '是否生成原生音频；首尾帧模式不可用。'),
            jsonb_build_object('name', 'reference_image_urls', 'description', '参考图 URL 数组，最多 30 张，PNG/JPG/WEBP，单张最大 25MB。'),
            jsonb_build_object('name', 'reference_videos', 'description', '参考视频 URL 数组，最多 10 条；MP4/MOV，单条及合计均不超过 30.2 秒。'),
            jsonb_build_object('name', 'reference_audios', 'description', '参考音频 URL 数组，最多 10 条；MP3/WAV，单条及合计均不超过 30.2 秒。'),
            jsonb_build_object('name', 'first_image_url', 'description', '首帧 URL；必须与 last_image_url 成对提供，并与多模态参考素材互斥。'),
            jsonb_build_object('name', 'last_image_url', 'description', '尾帧 URL；必须与 first_image_url 成对提供，并与多模态参考素材互斥。')
        ),
        'basic_request_json', jsonb_build_object(
            'model', v.public_name, 'prompt', '电影感城市夜景，镜头缓慢推进',
            'duration', 8, 'aspect_ratio', '16:9', 'generate_audio', true
        ),
        'request_json', jsonb_build_object(
            'model', v.public_name, 'prompt', '使用参考主体生成一段自然运动的视频',
            'duration', 8, 'aspect_ratio', '16:9',
            'reference_image_urls', jsonb_build_array('https://cdn.example.com/reference.png')
        ),
        'create_response_json', jsonb_build_object('id', 'task_video_01HZX8A2...', 'status', 'queued', 'model', v.public_name),
        'query_response_json', jsonb_build_object('id', 'task_video_01HZX8A2...', 'status', 'completed', 'data', jsonb_build_array(jsonb_build_object('url', 'https://example.com/video.mp4')))
    )::text,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('cy-sd4-seedance-2.5-480p', 'sd4-seedance-2.5-480p', '480p', 30,
     '当前 8500 积分号池可覆盖带参考视频的 30 秒模型上限，账号已有消耗时可用时长会降低。',
     '带参考视频仍可到模型自身 30 秒上限，但受账号实时余额约束。'),
    ('cy-sd4-seedance-2.5-720p', 'sd4-seedance-2.5-720p', '720p', 29,
     '当前 8500 积分号池中，无参考视频最多 29 秒；有参考视频时最多 18 秒，账号已有消耗时上限会进一步降低。',
     '带参考视频时当前号池最多 18 秒。')
) AS v(model_name, public_name, resolution, max_seconds, pool_limit, duration_note)
WHERE m.model_name = v.model_name AND m.deleted_at IS NULL;

COMMIT;

SELECT capability, profile_id, reference_limits, hints
FROM model_ui_param_profiles
WHERE profile_id = 'video-tpl-seedance-2.5-subscription-async';

SELECT model_name, api_doc::jsonb -> 'params' AS params
FROM models
WHERE model_name IN ('cy-sd4-seedance-2.5-480p', 'cy-sd4-seedance-2.5-720p')
ORDER BY model_name;
