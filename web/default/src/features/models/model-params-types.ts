/*
Copyright (C) 2023-2026 QuantumNous
*/
export type ModelUiParamCapability = 'video' | 'image'

export interface ModelUiParamRegistry {
  id: number
  capability: ModelUiParamCapability
  default_profile_id: string
  poll_defaults: string
  updated_time: number
}

export interface ModelUiParamProfile {
  id: number
  capability: ModelUiParamCapability
  profile_id: string
  api_mode?: string
  requires_reference_media: boolean
  poll: string
  poll_status?: string
  reference_limits: string
  params: string
  option_rules: string
  hints: string
  note?: string
  created_time: number
  updated_time: number
}

export const modelParamsQueryKeys = {
  all: ['model-params'] as const,
  registry: (capability: ModelUiParamCapability) =>
    [...modelParamsQueryKeys.all, 'registry', capability] as const,
  profiles: (capability: ModelUiParamCapability) =>
    [...modelParamsQueryKeys.all, 'profiles', capability] as const,
  modelBindings: (keyword: string) =>
    [...modelParamsQueryKeys.all, 'model-bindings', keyword] as const,
}

export const VIDEO_PARAM_KEYS = [
  'resolution',
  'ratio',
  'duration',
  'generateAudio',
  'watermark',
  'seed',
  'widthHeight',
  'frameInputs',
] as const

export const IMAGE_PARAM_KEYS = [
  'quality',
  'aspectRatio',
  'customDimensions',
  'count',
  'background',
  'outputFormat',
  'outputCompression',
  'moderation',
] as const

export const VIDEO_API_MODES = [
  'videos-json-gz',
  'videos-form',
  'videos-json-async',
  'chat-completions',
  'video-generations',
] as const

export const DEFAULT_VIDEO_PROFILE_ID = 'default-video'
export const DEFAULT_IMAGE_PROFILE_ID = 'default-image'
