-- Schema for model UI param tables (PostgreSQL).
-- Safe to re-run: uses IF NOT EXISTS.

CREATE TABLE IF NOT EXISTS model_ui_param_registries (
    id BIGSERIAL PRIMARY KEY,
    capability VARCHAR(16) NOT NULL,
    default_profile_id VARCHAR(128) NOT NULL,
    capability_fallback TEXT NOT NULL DEFAULT '[]',
    poll_defaults TEXT NOT NULL DEFAULT '{}',
    updated_time BIGINT,
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_model_ui_param_registry_capability ON model_ui_param_registries (capability);
CREATE INDEX IF NOT EXISTS idx_model_ui_param_registries_deleted_at ON model_ui_param_registries (deleted_at);

CREATE TABLE IF NOT EXISTS model_ui_param_profiles (
    id BIGSERIAL PRIMARY KEY,
    capability VARCHAR(16) NOT NULL,
    profile_id VARCHAR(128) NOT NULL,
    match TEXT NOT NULL DEFAULT '[]',
    sort_order BIGINT NOT NULL DEFAULT 0,
    api_mode VARCHAR(32) DEFAULT '',
    requires_reference_media BOOLEAN NOT NULL DEFAULT FALSE,
    poll TEXT NOT NULL DEFAULT '{}',
    poll_status VARCHAR(16) DEFAULT '',
    reference_limits TEXT NOT NULL DEFAULT '{}',
    params TEXT NOT NULL DEFAULT '{}',
    option_rules TEXT NOT NULL DEFAULT '[]',
    hints TEXT NOT NULL DEFAULT '[]',
    note VARCHAR(512) DEFAULT '',
    created_time BIGINT,
    updated_time BIGINT,
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_model_ui_param_profile_cap_id ON model_ui_param_profiles (capability, profile_id);
CREATE INDEX IF NOT EXISTS idx_model_ui_param_profiles_deleted_at ON model_ui_param_profiles (deleted_at);
CREATE INDEX IF NOT EXISTS idx_model_ui_param_profiles_sort ON model_ui_param_profiles (capability, sort_order);

-- Seed data: run Go importer after deploy:
--   go run ./scripts/seed_model_ui_params/main.go -force
--
-- Optional alias for OAIREGBox Grok Chat video (public name collision with 119337):
INSERT INTO model_public_aliases (internal_name, public_name, created_time, updated_time)
VALUES ('oairegbox-grok-video', 'grok-imagine-video', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (internal_name) DO UPDATE SET public_name = EXCLUDED.public_name, updated_time = EXCLUDED.updated_time;
