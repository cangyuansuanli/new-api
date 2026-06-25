-- Rename video api_mode values (breaking change, no backward compatibility).
-- Run once on existing deployments before rolling out new-api + infinite-canvas.

UPDATE model_ui_param_profiles
SET api_mode = CASE api_mode
    WHEN 'registry' THEN 'videos-json-gz'
    WHEN 'openai' THEN 'videos-form'
    WHEN 'newapi-async' THEN 'videos-json-async'
    WHEN 'newapi-chat' THEN 'chat-completions'
    WHEN 'newapi-grok' THEN 'video-generations'
    ELSE api_mode
END
WHERE capability = 'video'
  AND api_mode <> '';

UPDATE model_ui_param_registries
SET poll_defaults = replace(
    replace(
        replace(
            replace(
                replace(
                    poll_defaults,
                    '"registry":', '"videos-json-gz":'
                ),
                '"openai":', '"videos-form":'
            ),
            '"newapi-async":', '"videos-json-async":'
        ),
        '"newapi-chat":', '"chat-completions":'
    ),
    '"newapi-grok":', '"video-generations":'
)
WHERE capability = 'video';
