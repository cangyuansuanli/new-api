/*
Copyright (C) 2023-2026 QuantumNous
*/
import type { TFunction } from 'i18next'
import { VIDEO_API_MODES } from './model-params-types'

export type VideoApiMode = (typeof VIDEO_API_MODES)[number]

const VIDEO_API_MODE_META: Record<
  VideoApiMode,
  { labelKey: string; descKey: string }
> = {
  'videos-json-gz': {
    labelKey: 'API mode: videos-json-gz',
    descKey:
      'POST /v1/videos with JSON body (GZ payload: multi-ref image/video/audio, frame inputs).',
  },
  'videos-form': {
    labelKey: 'API mode: videos-form',
    descKey:
      'POST /v1/videos as multipart FormData (OpenAI Sora-style: seconds, size, input_reference[]).',
  },
  'videos-json-async': {
    labelKey: 'API mode: videos-json-async',
    descKey:
      'POST /v1/videos with JSON body, poll GET /v1/videos/{id} (Omni, Seedance, veo-clean, etc.).',
  },
  'chat-completions': {
    labelKey: 'API mode: chat-completions',
    descKey:
      'POST /v1/chat/completions streaming; video URL returned inline (oairegbox-grok-video, etc.).',
  },
  'video-generations': {
    labelKey: 'API mode: video-generations',
    descKey:
      'POST /v1/video/generations, poll GET /v1/video/generations/{id} (119337-grok-video, etc.).',
  },
}

export function getVideoApiModeLabel(t: TFunction, mode: string) {
  const meta = VIDEO_API_MODE_META[mode as VideoApiMode]
  return meta ? t(meta.labelKey) : mode
}

export function getVideoApiModeDescription(t: TFunction, mode: string) {
  const meta = VIDEO_API_MODE_META[mode as VideoApiMode]
  return meta ? t(meta.descKey) : ''
}

export function getVideoApiModeOptions(t: TFunction) {
  return VIDEO_API_MODES.map((mode) => ({
    value: mode,
    label: getVideoApiModeLabel(t, mode),
    description: getVideoApiModeDescription(t, mode),
  }))
}
