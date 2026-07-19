# 客户端错误归一化覆盖表

单入口：`service/clienterror/normalize.go` → `NormalizeClientErrorMessage`

新增 vendor 规则：在 `service/clienterror/<vendor>.go` 实现 `normalize<Vendor>`，并在 `normalize.go` 的 `init()` 里 `Register(...)`。

## common.go（跨渠道）

| 上游来源 | 典型 raw | 状态 |
|---------|---------|------|
| 各渠道内容审查 | `content moderation`, `unsafe content`, 谷粒中文政策 | ✅ |
| Manju 审核 | `审核失败`, `平台内容审核`, `参考图未通过` | ✅ |
| 代理/HTTP | `status_code=502/503/524`, `do request failed` | ✅ |
| adobe2api/media_limits | `image too large, max 30MB` | ✅ |
| adobe2api/generation | `too many images, max 9` | ✅ |
| adobe2api/app | `Upstream is temporarily unavailable` | ✅ |
| adobe2api | `Failed to fetch image_url`, `image_url is empty` | ✅ |
| new-api/adapt_adobe2api | `mask supports exactly one` | ✅（adobe.go） |
| 通用 | `prompt length exceeds`, `prompt exceeds 4096 characters` | ✅ |
| 真人脸 | `real human face`, `reference image rejected`, `可识别真人肖像` | ✅ |

## leonardo.go（cy-sd4 / leonardo-web2api）

| 上游来源 | 典型 raw | 状态 |
|---------|---------|------|
| video_multimodal.go | `reference images exceed Leonardo limit (5/4)` | ✅ |
| video_multimodal.go | `multimodal references cannot be combined...` | ✅ |
| public_message.go | `All cookies failed. cookie#N: ...` | ✅ → 按失败类型汇总（不暴露 vendor / 账号编号） |
| public_message.go | `depleted`, `no active cookie` | ✅ → 号池耗尽 + 缩短秒数/经济模型引导 |
| public_message.go | `insufficient credits (need/have)` | ✅ → 本次积分不足 + 换短时长/经济模型 |
| public_message.go | `busy`, `cooldown`, `proxy`, `auth_expired` | ✅ → 各自具体文案 |
| generation_failure.go | `leonardo: video generation failed (FAILED): ...` | ✅ |
| generation_failure.go | `upstream returned no detail` | ✅ |
| leonardo API | `DURATION_TOO_LONG` | ✅ |

## adobe.go（adobe2api / cy-sd5 / adobe-direct）

| 上游来源 | 典型 raw | 状态 |
|---------|---------|------|
| generation.py | `entity not found`, `Unsupported parameters for video model` | ✅（common invalid_request） |
| generation.py | `mask is only supported for gpt-image` | ✅（common） |
| generation.py | `entities in one prompt must belong to the same Adobe account` | ✅ |
| adapt_adobe2api.go | `mask is only supported for Adobe GPT Image 2` | ✅ |
| entity.py | `type must be one of: character, object, location` | ❌ 透传 |
| Adobe 上游原始错误 | 各类 Firefly moderation 原文 | 部分（走 common content policy） |
| adobe2api video poll | `451 prompt_unsafe`, `provided prompt is considered unsafe` | ✅（adobe.go + common；画布需 relay 带 `X-Cangyuan-Client` 才返回中文） |

## grok.go（cy-gv1 / geeknowgrok）

| 上游来源 | 典型 raw | 状态 |
|---------|---------|------|
| grok/adaptor.go | `Grok video supports at most 7 reference images` | ✅ |
| grok/adaptor.go | `grok-video-1.5 requires exactly one reference image` | ✅ |
| grok/adaptor.go | `does not support video references` | ✅ |
| 119337 轮询 | `fail_reason: reference image rejected` | ✅（common） |

## manju.go（manju-openai-sora*）

| 上游来源 | 典型 raw | 状态 |
|---------|---------|------|
| convert_test.go | `审核失败` | ✅ |
| convert_test.go | `某张上传的参考图未通过平台内容审核...` | ✅ |
| convert.go | `task failed` | ✅ |

## chatvideo.go（cy-vid2-sora-2 等 chat 线路）

| 上游来源 | 典型 raw | 状态 |
|---------|---------|------|
| chatvideo/adaptor.go | `chat video response does not contain a video url` | ✅ |
| chatvideo/adaptor.go | `empty chat video response` | ✅ |
| SSE error.message | 内容审查类 | ✅（common，透传后 normalize） |

## defaultvideo.go（sora-2 等标准 OpenAI Video 聚合）

| 上游来源 | 典型 raw | 状态 |
|---------|---------|------|
| adaptor_test.go | `content policy violation` | ✅（common） |
| adaptor_test.go | `Generated video rejected by content moderation.` | ✅（common） |
| adaptor.go | `task failed` | ✅ |
| 上游 envelope | `Client specified an invalid argument` | ✅ |

## 调用链 normalize 覆盖

| 路径 | 状态 |
|------|------|
| controller/relay.go defer | ✅ |
| relay_task TaskModel2DtoForClient | ✅ |
| relay_task GET /v1/videos/{id} | ✅ |
| relay/image/fetch.go | ✅ |
| controller/image_sync_queue.go 失败等待 | ✅ |

## 待补

| Vendor | 说明 |
|--------|------|
| seedancetengda (cy-sd2) | 错误多来自 Geeknow 上游，common 兜底 |
| entity.py 细粒度 | Adobe 主体创建参数 |
| Midjourney / Suno | 非 OpenAI Video 族，未纳入 clienterror |
