-- 渠道 106/107：分别接入 grok-imagine-video 1.0/1.5，避免上游分组串路由。
BEGIN;

INSERT INTO model_channel_prefixes (prefix, note, enabled, sort_order, created_time, updated_time)
VALUES ('cy-gv2-', '渠道 106/107 Grok Imagine Video', TRUE, 126, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (prefix) DO UPDATE SET note = EXCLUDED.note, enabled = TRUE, updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT;

UPDATE channels SET
    models = 'cy-gv2-grok-video',
    model_mapping = '{"cy-gv2-grok-video":"grok-imagine-video"}',
    "group" = 'VIDEO,全模型-无claude/gpt',
    status = 1
WHERE id = 106;

UPDATE channels SET
    models = 'cy-gv2-grok-video-1.5',
    model_mapping = '{"cy-gv2-grok-video-1.5":"grok-imagine-video-1.5"}',
    "group" = 'VIDEO,全模型-无claude/gpt',
    status = 1
WHERE id = 107;

DELETE FROM abilities
WHERE channel_id IN (106, 107)
  AND model IN ('cy-gv2-grok-video', 'cy-gv2-grok-video-1.5');

INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT g.grp, m.model, m.channel_id, TRUE, 0, 90
FROM (VALUES
    ('cy-gv2-grok-video', 106),
    ('cy-gv2-grok-video-1.5', 107)
) AS m(model, channel_id)
CROSS JOIN (VALUES ('VIDEO'), ('全模型-无claude/gpt')) AS g(grp);

INSERT INTO models (
    model_name, description, tags, vendor_id, endpoints, status, sync_official,
    video_profile_id, created_time, updated_time
)
SELECT
    v.model_name, v.description, 'video,grok', 1,
    '{"openai-video":{"path":"/v1/videos","method":"POST"}}', 1, 0,
    'video-tpl-gen-ratio-ref7',
    EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
FROM (VALUES
    ('cy-gv2-grok-video', 'Grok 视频 1.0：支持文生/单图，480p/720p，1～15 秒。'),
    ('cy-gv2-grok-video-1.5', 'Grok 视频 1.5：支持文生/单图/最多 7 图；1080p 仅限文生或单图，1～15 秒。')
) AS v(model_name, description)
WHERE NOT EXISTS (
    SELECT 1 FROM models m
    WHERE m.model_name = v.model_name AND m.deleted_at IS NULL
);

COMMIT;
