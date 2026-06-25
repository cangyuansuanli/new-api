-- Model UI params: per-model profile binding (video_profile_id / image_profile_id on models table).
-- Profile templates no longer use match tokens; bindings are explicit on models.

CREATE TABLE IF NOT EXISTS model_ui_param_registries (
    id SERIAL PRIMARY KEY,
    capability VARCHAR(16) NOT NULL,
    default_profile_id VARCHAR(128) NOT NULL,
    poll_defaults TEXT NOT NULL DEFAULT '{}',
    updated_time BIGINT,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_model_ui_param_registry_capability ON model_ui_param_registries (capability);
CREATE INDEX IF NOT EXISTS idx_model_ui_param_registries_deleted_at ON model_ui_param_registries (deleted_at);

CREATE TABLE IF NOT EXISTS model_ui_param_profiles (
    id SERIAL PRIMARY KEY,
    capability VARCHAR(16) NOT NULL,
    profile_id VARCHAR(128) NOT NULL,
    api_mode VARCHAR(32),
    requires_reference_media BOOLEAN NOT NULL DEFAULT FALSE,
    poll TEXT NOT NULL DEFAULT '{}',
    poll_status VARCHAR(16),
    reference_limits TEXT NOT NULL DEFAULT '{}',
    params TEXT NOT NULL DEFAULT '{}',
    option_rules TEXT NOT NULL DEFAULT '[]',
    hints TEXT NOT NULL DEFAULT '[]',
    note VARCHAR(512),
    created_time BIGINT,
    updated_time BIGINT,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_model_ui_param_profile_cap_id ON model_ui_param_profiles (capability, profile_id);
CREATE INDEX IF NOT EXISTS idx_model_ui_param_profiles_deleted_at ON model_ui_param_profiles (deleted_at);

ALTER TABLE models ADD COLUMN IF NOT EXISTS video_profile_id VARCHAR(128) DEFAULT '';
ALTER TABLE models ADD COLUMN IF NOT EXISTS image_profile_id VARCHAR(128) DEFAULT '';

-- Seed profiles + auto-bind models from seed JSON:
--   go run ./scripts/seed_model_ui_params/main.go -force
