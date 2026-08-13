-- Retire veo-clean from customer-facing model plaza (internal upstream may remain on channel).
-- Run on production after deploy; adjust model_name if your register name differs.

UPDATE models
SET status = 2,
    video_profile_id = '',
    updated_time = extract(epoch from now())::bigint
WHERE model_name IN ('oairegbox-veo-clean', 'veo-clean')
  AND deleted_at IS NULL;

DELETE FROM model_ui_param_profiles
WHERE profile_id = 'video-tpl-async-v2v-clean';

SELECT model_name, status, video_profile_id
FROM models
WHERE model_name LIKE '%veo-clean%'
  AND deleted_at IS NULL;
