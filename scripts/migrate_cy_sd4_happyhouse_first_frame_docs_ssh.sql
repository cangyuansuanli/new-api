-- Happy House 1.0 / 1.1：补齐线上 profile 与 models.api_doc 的首帧能力。
-- 源站执行：docker exec -i newapi-postgres psql -U root -d new-api -v ON_ERROR_STOP=1 < migrate_cy_sd4_happyhouse_first_frame_docs_ssh.sql

BEGIN;

UPDATE model_ui_param_profiles
SET
    reference_limits = CASE profile_id
        WHEN 'video-tpl-happyhouse-1.1-async' THEN '{"images":9,"videos":0,"audios":0,"imageMaxBytes":26214400,"fullReferenceMode":{"label":"多图参考","descriptionWithImages":"最多 9 张参考图"},"validationHint":"支持单张首帧或参考图；参考图支持 png/jpg/webp，单张不超过 25MB，最多 9 张。不支持参考视频和参考音频。","showTempMediaHint":true,"prependReferenceGuide":true}'
        WHEN 'video-tpl-happyhouse-1.0-async' THEN '{"images":9,"videos":1,"audios":0,"imageMaxBytes":26214400,"videoMaxBytes":209715200,"video":{"minDurationMs":3000,"maxDurationMs":10000,"totalMaxDurationMs":10000},"fullReferenceMode":{"label":"多模态","descriptionWithImages":"参考图 + 可选 1 个参考视频"},"validationHint":"支持单张首帧或参考素材；参考图支持 png/jpg/webp，单张不超过 25MB，最多 9 张；带参考视频时最多 5 张图。参考视频支持 mp4/mov，不超过 200MB，宽高各 720–2160px、24–60 FPS、3–10 秒，最多 1 个。不支持参考音频。","showTempMediaHint":true,"prependReferenceGuide":true}'
    END,
    params = '{"resolution":{"enabled":true,"options":[{"value":"720p","label":"HD 720p"},{"value":"1080p","label":"Full HD 1080p"}]},"ratio":{"enabled":true,"options":[{"value":"16:9","label":"横屏"},{"value":"9:16","label":"竖屏"},{"value":"1:1","label":"方形"},{"value":"3:4","label":"3:4"},{"value":"4:3","label":"4:3"}]},"duration":{"enabled":true,"numericOptions":[3,4,5,6,7,8,9,10,11,12,13,14,15],"min":3,"max":15},"generateAudio":{"enabled":true,"hint":"是否生成原生音频，默认开启"},"watermark":{"enabled":false},"seed":{"enabled":false},"widthHeight":{"enabled":false},"frameInputs":{"enabled":true,"hint":"支持首帧；与多图/视频参考素材二选一"}}',
    hints = CASE profile_id
        WHEN 'video-tpl-happyhouse-1.1-async' THEN '[{"text":"720p / 1080p，3–15 秒；支持单张首帧或最多 9 张参考图，不支持参考音频。"}]'
        WHEN 'video-tpl-happyhouse-1.0-async' THEN '[{"text":"720p / 1080p，3–15 秒；支持单张首帧、最多 9 张参考图或 1 个参考视频，带视频时最多 5 张图。"}]'
    END,
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT,
    deleted_at = NULL
WHERE capability = 'video'
  AND profile_id IN ('video-tpl-happyhouse-1.0-async', 'video-tpl-happyhouse-1.1-async');

WITH docs(model_name, public_name, intro, image_help, video_help, supports_video) AS (
    VALUES
    ('cy-sd4-happyhouse-1.1', 'happyhouse-1.1', 'Happy House 1.1 异步视频。按条计费；失败不计费。', '参考图 URL 数组，最多 9 张；支持 PNG/JPG/WEBP，单张不超过 25MB。', '', false),
    ('cy-sd4-happyhouse-1.0', 'happyhouse-1.0', 'Happy House 1.0 异步视频。按条计费；失败不计费。', '参考图 URL 数组，最多 9 张；带参考视频时最多 5 张。支持 PNG/JPG/WEBP，单张不超过 25MB。', '可选 1 个 MP4/MOV 公网 HTTPS 直链，不超过 200MB，3–10 秒。', true)
)
UPDATE models m
SET api_doc = jsonb_build_object(
    'dispatch_mode', 'async',
    'intro', d.intro || ' 仅使用本页列出的字段，未列字段不要发送。',
    'endpoints', jsonb_build_array(
        jsonb_build_object('method', 'POST', 'path', '{{base}}/videos', 'description', '创建视频任务（application/json）。'),
        jsonb_build_object('method', 'GET', 'path', '{{base}}/videos/{task_id}', 'description', '查询任务状态与结果。'),
        jsonb_build_object('method', 'GET', 'path', '{{base}}/videos/{task_id}/content', 'description', '下载已完成任务。')
    ),
    'params', jsonb_build_array(
        jsonb_build_object('name', 'model', 'description', '必填，传模型广场显示的模型名。'),
        jsonb_build_object('name', 'prompt', 'description', '必填，视频描述。'),
        jsonb_build_object('name', 'duration', 'description', '成片时长，3–15 秒整数。'),
        jsonb_build_object('name', 'resolution', 'description', '720p 或 1080p。'),
        jsonb_build_object('name', 'aspect_ratio', 'description', '16:9、9:16、1:1、3:4 或 4:3。'),
        jsonb_build_object('name', 'generate_audio', 'description', '是否生成原生音频，默认 true。'),
        jsonb_build_object('name', 'first_image_url', 'description', '首帧 HTTPS URL；可单独使用，与 reference_image_urls/reference_videos 互斥；Happy House 不支持尾帧。'),
        jsonb_build_object('name', 'reference_image_urls', 'description', d.image_help)
    ) || CASE WHEN d.supports_video THEN jsonb_build_array(
        jsonb_build_object('name', 'reference_videos', 'description', d.video_help)
    ) ELSE '[]'::jsonb END,
    'basic_request_json', jsonb_build_object(
        'model', d.public_name, 'prompt', '电影感城市夜景', 'duration', 8,
        'resolution', '720p', 'aspect_ratio', '16:9', 'generate_audio', true
    ),
    'request_json', jsonb_build_object(
        'model', d.public_name, 'prompt', '电影感城市夜景', 'duration', 8,
        'resolution', '720p', 'aspect_ratio', '16:9', 'generate_audio', true,
        'reference_image_urls', jsonb_build_array('https://cdn.example.com/reference.png')
    ) || CASE WHEN d.supports_video THEN jsonb_build_object(
        'reference_videos', jsonb_build_array('https://cdn.example.com/reference.mp4')
    ) ELSE '{}'::jsonb END,
    'examples', jsonb_build_array(
        jsonb_build_object(
            'title', '参考素材',
            'request_json', jsonb_build_object(
                'model', d.public_name, 'prompt', '电影感城市夜景', 'duration', 8,
                'resolution', '720p', 'aspect_ratio', '16:9', 'generate_audio', true,
                'reference_image_urls', jsonb_build_array('https://cdn.example.com/reference.png')
            ) || CASE WHEN d.supports_video THEN jsonb_build_object(
                'reference_videos', jsonb_build_array('https://cdn.example.com/reference.mp4')
            ) ELSE '{}'::jsonb END
        ),
        jsonb_build_object(
            'title', '首帧',
            'request_json', jsonb_build_object(
                'model', d.public_name, 'prompt', '电影感城市夜景', 'duration', 8,
                'resolution', '720p', 'aspect_ratio', '16:9', 'generate_audio', true,
                'first_image_url', 'https://cdn.example.com/first.png'
            )
        )
    ),
    'create_response_json', jsonb_build_object('id', 'task_video_01HZX8A2...', 'status', 'queued', 'model', d.public_name),
    'query_response_json', jsonb_build_object('id', 'task_video_01HZX8A2...', 'status', 'completed', 'data', jsonb_build_array(jsonb_build_object('url', 'https://example.com/video.mp4')))
)::text,
updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
FROM docs d
WHERE m.model_name = d.model_name
  AND m.status = 1
  AND m.deleted_at IS NULL;

SELECT model_name, video_profile_id, api_doc::jsonb -> 'params' AS params
FROM models
WHERE model_name IN ('cy-sd4-happyhouse-1.0', 'cy-sd4-happyhouse-1.1')
  AND deleted_at IS NULL
ORDER BY model_name;

COMMIT;
