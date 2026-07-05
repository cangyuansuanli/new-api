-- 更新 Leonardo hint：参考视频总时长 ≤15s
UPDATE model_ui_param_profiles SET
    hints = replace(
        hints,
        '多模态 4图/3视频（总时长≤15s）/1音频',
        '多模态 4图/3视频（总时长≤15s）/1音频（≤15s）'
    ),
    updated_time = EXTRACT(EPOCH FROM NOW())::BIGINT
WHERE capability = 'video' AND profile_id = 'video-tpl-seedance-async';
