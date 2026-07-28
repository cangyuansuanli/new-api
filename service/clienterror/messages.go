package clienterror

const (
	ContentPolicyMessageZH = "您的提示词或参考素材未通过内容审查，请修改后重新提交。"
	ContentPolicyMessageEN = "Your prompt or reference material was rejected by content moderation. Please revise it and submit again."

	UpstreamUnavailableMessageZH = "服务暂时不可用，请稍后重试。"
	UpstreamUnavailableMessageEN = "Service temporarily unavailable, please retry later."
	UpstreamSaturatedMessageZH   = "当前分组上游负载已饱和，请稍后再试。"
	UpstreamSaturatedMessageEN   = "Upstream capacity for the current group is saturated. Please retry later."

	TimeoutMessageZH = "生成超时，请稍后重试。"
	TimeoutMessageEN = "Generation timed out, please retry later."

	MissingReferenceMessageZH = "参考图未正确传递，请重新上传后重试。"
	MissingReferenceMessageEN = "Reference image was not delivered correctly, please re-upload and retry."

	ReferenceMaterialMessageZH = "参考素材处理失败，请重新上传后重试。"
	ReferenceMaterialMessageEN = "Reference material could not be processed, please re-upload and retry."
	ReferenceDurationTooLongZH = "参考视频或音频超过模型时长限制，请缩短素材后重试。"
	ReferenceDurationTooLongEN = "Reference video or audio exceeds the model's duration limit. Shorten the source media and retry."

	ReferenceRealFaceMessageZH = "参考图或参考视频包含真实人脸，当前模型不支持，请更换不含真人的素材后重试。"
	ReferenceRealFaceMessageEN = "Reference images or videos contain real human faces, which this model does not support. Please use source media without real faces."

	GenerationFailedMessageZH = "视频生成失败，请稍后重试。"
	GenerationFailedMessageEN = "Video generation failed, please retry later."

	GenerationFailedNoDetailZH = "视频生成失败，未返回具体原因。如有参考素材，它们已通过上传和基础格式校验；生成阶段仍可能因内容审核、提示词与素材组合过于复杂或模型暂时不稳定而失败。请简化提示词、减少或更换参考素材后重试。"
	GenerationFailedNoDetailEN = "Video generation failed without a specific reason. Any submitted references passed upload and basic format checks; generation-stage moderation, complex prompt/reference combinations, or temporary model instability may still cause failure. Try a simpler prompt, fewer or different references, then retry."

	InvalidRequestMessageZH = "请求参数不符合要求，请检查后重试。"
	InvalidRequestMessageEN = "Request parameters are invalid, please check and retry."

	PoolUnavailableMessageZH = "视频服务暂时不可用，请稍后重试。"
	PoolUnavailableMessageEN = "Video service is temporarily unavailable, please retry later."

	PoolDepletedMessageZH = "号池可用额度已耗尽，请联系管理员补充；如需大批量使用请提前预约。在此之前可先缩短视频秒数、降低分辨率，或改用 480p/经济档模型再试，以充分利用剩余额度。"
	PoolDepletedMessageEN = "Video pool credits are depleted. Contact an administrator to refill the pool. Until then, try a shorter duration, lower resolution, or an economy model (e.g. 480p) to make the most of any remaining credits."

	InsufficientCreditsForJobMessageZH = "本次生成所需积分超过号池当前可用余额（各账号剩余额度不足以支付该请求）。请缩短视频秒数、减少参考素材，或改用 480p/经济档等更省积分的模型后重试。"
	InsufficientCreditsForJobMessageEN = "This job requires more credits than any account currently has available. Shorten the video duration, use fewer references, or switch to an economy model (e.g. 480p) and retry."
)
