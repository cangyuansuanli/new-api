-- Seed model naming config for PostgreSQL (production new-api).
-- Safe to re-run: uses IF NOT EXISTS / ON CONFLICT DO NOTHING.

CREATE TABLE IF NOT EXISTS model_channel_prefixes (
    id BIGSERIAL PRIMARY KEY,
    prefix VARCHAR(64) NOT NULL,
    note VARCHAR(255) DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order BIGINT NOT NULL DEFAULT 0,
    created_time BIGINT,
    updated_time BIGINT,
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_model_channel_prefix ON model_channel_prefixes (prefix);
CREATE INDEX IF NOT EXISTS idx_model_channel_prefixes_deleted_at ON model_channel_prefixes (deleted_at);

INSERT INTO model_channel_prefixes (prefix, note, enabled, sort_order, created_time, updated_time)
VALUES
  ('119337-', 'api.119337.xyz Grok Video', TRUE, 10, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
  ('aini-', 'Aini 生图', TRUE, 20, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
  ('byte-', '字节 Gemini 生图', TRUE, 30, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
  ('ctlove-', 'CTLove 字节火山 Seedance', TRUE, 40, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
  ('czeq-', 'Czeq 生图', TRUE, 50, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
  ('go2api-', 'Go2API 生图', TRUE, 60, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
  ('gz-', 'GZ / 冠臻 Seedance', TRUE, 70, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
  ('happyhorse-', 'HappyHorse 视频', TRUE, 80, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
  ('niming-', '匿名 Gemini 生图', TRUE, 90, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
  ('oairegbox-', 'OAIREGBox', TRUE, 100, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
  ('yunwu-', '云雾 / Apifox 聚合', TRUE, 110, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
  ('zeabur-', 'Zeabur 托管 Gemini/GLM', TRUE, 120, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (prefix) DO NOTHING;

-- OAIREGBox Grok Chat 视频 public 名与 119337 grok-video 碰撞时，alias 为 grok-imagine-video。
INSERT INTO model_public_aliases (internal_name, public_name, created_time, updated_time)
VALUES ('oairegbox-grok-video', 'grok-imagine-video', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (internal_name) DO UPDATE SET public_name = EXCLUDED.public_name, updated_time = EXCLUDED.updated_time;

-- Remove incorrect hotfix alias (go2api-gpt-image-2-1k → gpt-image-2).
-- After prefix seed, public name auto-resolves to gpt-image-2-1k.
DELETE FROM model_public_aliases
WHERE internal_name = 'go2api-gpt-image-2-1k'
  AND public_name = 'gpt-image-2';
