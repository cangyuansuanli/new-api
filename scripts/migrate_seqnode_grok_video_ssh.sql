-- 渠道 106：独立接入 grok-imagine-video 协议。
BEGIN;
INSERT INTO model_channel_prefixes (prefix, note, enabled, sort_order, created_time, updated_time)
VALUES ('cy-gv2-', '渠道 106 Grok Imagine Video', TRUE, 126, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (prefix) DO UPDATE SET note = EXCLUDED.note, enabled = TRUE, updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT;
UPDATE channels SET models = 'cy-gv2-grok-video,cy-gv2-grok-video-1.5', model_mapping = '{"cy-gv2-grok-video":"grok-imagine-video","cy-gv2-grok-video-1.5":"grok-imagine-video-1.5"}', "group" = 'VIDEO,全模型-无claude/gpt', status = 1 WHERE id = 106;
DELETE FROM abilities WHERE channel_id = 106 AND model IN ('cy-gv2-grok-video', 'cy-gv2-grok-video-1.5');
INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT g.grp, m.model, 106, true, 0, 90 FROM (VALUES ('VIDEO'), ('全模型-无claude/gpt')) g(grp) CROSS JOIN (VALUES ('cy-gv2-grok-video'), ('cy-gv2-grok-video-1.5')) m(model);
INSERT INTO models (model_name, description, tags, vendor_id, endpoints, status, sync_official, video_profile_id, created_time, updated_time)
SELECT v.model_name, v.description, 'video,grok', 1, '{"openai-video":{"path":"/v1/videos","method":"POST"}}', 1, 0, 'video-tpl-gen-ratio-ref7', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT FROM (VALUES ('cy-gv2-grok-video','Grok Video 2'), ('cy-gv2-grok-video-1.5','Grok Video 2 1.5')) v(model_name, description) WHERE NOT EXISTS (SELECT 1 FROM models m WHERE m.model_name = v.model_name AND m.deleted_at IS NULL);
COMMIT;
