-- Profile 路由字段：payload_builder（提交 payload 组装）、validation_key（提交前校验）
-- 客户端只读 profile 文档，不再维护 profile id 白名单。

ALTER TABLE model_ui_param_profiles
    ADD COLUMN IF NOT EXISTS payload_builder VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS validation_key VARCHAR(64) NOT NULL DEFAULT '';

-- Seedance（oairegbox cy-sd1）
UPDATE model_ui_param_profiles
SET payload_builder = 'seedance-flat',
    validation_key = 'seedance-oairegbox',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video'
  AND profile_id IN (
    'video-tpl-seedance-480p-async',
    'video-tpl-seedance-720p-async',
    'video-tpl-seedance-1080p-async',
    'video-tpl-seedance-4k-async'
  );

-- Seedance（Leonardo cy-sd4）
UPDATE model_ui_param_profiles
SET payload_builder = 'seedance-flat',
    validation_key = 'seedance-cy-sd4',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video'
  AND profile_id = 'video-tpl-cy-sd4-seedance-async';

-- Omni / Veo async
UPDATE model_ui_param_profiles
SET payload_builder = 'omni-frame',
    validation_key = 'omni-frame',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video'
  AND profile_id = 'video-tpl-async-ratio-frame-ref5';

UPDATE model_ui_param_profiles
SET payload_builder = 'omni-v2v',
    validation_key = 'omni-v2v',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video'
  AND profile_id = 'video-tpl-async-v2v-ref1';

UPDATE model_ui_param_profiles
SET payload_builder = 'omni-v2v-clean',
    validation_key = 'omni-v2v',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video'
  AND profile_id = 'video-tpl-async-v2v-clean';

-- Grok CLI async
UPDATE model_ui_param_profiles
SET payload_builder = 'grok-cli',
    validation_key = 'grok-cli',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video'
  AND profile_id = 'video-tpl-async-ratio-duration-ref10v1';

UPDATE model_ui_param_profiles
SET payload_builder = 'grok-cli-i2v',
    validation_key = 'grok-cli-i2v',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video'
  AND profile_id = 'video-tpl-async-ratio-ref1';

-- Tengda Veo
UPDATE model_ui_param_profiles
SET payload_builder = 'tengda-veo',
    validation_key = 'tengda-veo',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video'
  AND profile_id = 'video-tpl-async-ratio-duration-ref1-veo';

-- Grok video-generations / chat
UPDATE model_ui_param_profiles
SET validation_key = 'grok-generations-ref7',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video'
  AND profile_id = 'video-tpl-gen-ratio-ref7';

UPDATE model_ui_param_profiles
SET validation_key = 'grok-generations-ref1',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video'
  AND profile_id = 'video-tpl-gen-ratio-ref1';

UPDATE model_ui_param_profiles
SET validation_key = 'grok-generations-chat-ref7',
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video'
  AND profile_id = 'video-tpl-chat-ref7';
