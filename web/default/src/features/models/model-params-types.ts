/*
Copyright (C) 2023-2026 QuantumNous
*/
export type ModelUiParamCapability = 'video' | 'image'

export interface ModelUiParamRegistry {
  id: number
  capability: ModelUiParamCapability
  default_profile_id: string
  capability_fallback: string
  poll_defaults: string
  updated_time: number
}

export interface ModelUiParamProfile {
  id: number
  capability: ModelUiParamCapability
  profile_id: string
  match: string
  sort_order: number
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

export interface ModelUiParamMatchPreview {
  model_name: string
  matched_profile?: string
  matched_profiles?: string[]
  collision: boolean
}

export const modelParamsQueryKeys = {
  all: ['model-params'] as const,
  registry: (capability: ModelUiParamCapability) =>
    [...modelParamsQueryKeys.all, 'registry', capability] as const,
  profiles: (capability: ModelUiParamCapability) =>
    [...modelParamsQueryKeys.all, 'profiles', capability] as const,
  preview: (capability: ModelUiParamCapability, modelName: string) =>
    [...modelParamsQueryKeys.all, 'preview', capability, modelName] as const,
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
  'registry',
  'openai',
  'newapi-async',
  'newapi-chat',
  'newapi-grok',
] as const
