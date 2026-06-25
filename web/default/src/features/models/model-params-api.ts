/*
Copyright (C) 2023-2026 QuantumNous
*/
import { api } from '@/lib/api'
import type {
  ModelUiParamCapability,
  ModelUiParamMatchPreview,
  ModelUiParamProfile,
  ModelUiParamRegistry,
} from './model-params-types'

type ApiListResponse<T> = {
  success: boolean
  message?: string
  data?: T
}

export async function getModelUiParamRegistry(
  capability: ModelUiParamCapability
): Promise<ModelUiParamRegistry> {
  const res = await api.get<ApiListResponse<ModelUiParamRegistry>>(
    `/api/model_ui_param_registries/${capability}`
  )
  if (!res.data.data) throw new Error(res.data.message || 'registry not found')
  return res.data.data
}

export async function updateModelUiParamRegistry(
  capability: ModelUiParamCapability,
  data: Partial<
    Pick<
      ModelUiParamRegistry,
      'default_profile_id' | 'capability_fallback' | 'poll_defaults'
    >
  >
): Promise<ApiListResponse<ModelUiParamRegistry>> {
  const res = await api.put(`/api/model_ui_param_registries/${capability}`, data)
  return res.data
}

export async function listModelUiParamProfiles(
  capability: ModelUiParamCapability
): Promise<ModelUiParamProfile[]> {
  const res = await api.get<ApiListResponse<ModelUiParamProfile[]>>(
    '/api/model_ui_param_profiles/',
    { params: { capability } }
  )
  return res.data.data ?? []
}

export async function createModelUiParamProfile(
  data: Partial<ModelUiParamProfile> & {
    capability: ModelUiParamCapability
    profile_id: string
  }
): Promise<ApiListResponse<ModelUiParamProfile>> {
  const res = await api.post('/api/model_ui_param_profiles/', data)
  return res.data
}

export async function updateModelUiParamProfile(
  data: Partial<ModelUiParamProfile> & { id: number }
): Promise<ApiListResponse<ModelUiParamProfile>> {
  const res = await api.put('/api/model_ui_param_profiles/', data)
  return res.data
}

export async function deleteModelUiParamProfile(
  id: number
): Promise<ApiListResponse<null>> {
  const res = await api.delete(`/api/model_ui_param_profiles/${id}`)
  return res.data
}

export async function reorderModelUiParamProfiles(
  capability: ModelUiParamCapability,
  items: Array<{ id: number; sort_order: number }>
): Promise<ApiListResponse<null>> {
  const res = await api.post('/api/model_ui_param_profiles/reorder', {
    capability,
    items,
  })
  return res.data
}

export async function previewModelUiParamMatch(
  capability: ModelUiParamCapability,
  modelName: string
): Promise<ModelUiParamMatchPreview> {
  const res = await api.get<ApiListResponse<ModelUiParamMatchPreview>>(
    '/api/model_ui_param_profiles/preview-match',
    { params: { capability, model_name: modelName } }
  )
  return res.data.data ?? { model_name: modelName, collision: false }
}
