/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { api } from '@/lib/api'
import type {
  ModelPublicAlias,
  ModelPublicNameRegistryStatus,
  ModelRoutingAlias,
} from './model-naming-types'

type ApiListResponse<T> = {
  success: boolean
  message?: string
  data?: T
}

export async function listModelPublicAliases(): Promise<ModelPublicAlias[]> {
  const res = await api.get<ApiListResponse<ModelPublicAlias[]>>(
    '/api/model_public_aliases/'
  )
  return res.data.data ?? []
}

export async function createModelPublicAlias(
  data: Pick<
    ModelPublicAlias,
    'internal_name' | 'public_name' | 'hidden_from_marketplace'
  >
): Promise<ApiListResponse<ModelPublicAlias>> {
  const res = await api.post('/api/model_public_aliases/', data)
  return res.data
}

export async function updateModelPublicAlias(
  data: Pick<
    ModelPublicAlias,
    'id' | 'internal_name' | 'public_name' | 'hidden_from_marketplace'
  >
): Promise<ApiListResponse<ModelPublicAlias>> {
  const res = await api.put('/api/model_public_aliases/', data)
  return res.data
}

export async function deleteModelPublicAlias(
  id: number
): Promise<ApiListResponse<null>> {
  const res = await api.delete(`/api/model_public_aliases/${id}`)
  return res.data
}

export async function listModelRoutingAliases(): Promise<ModelRoutingAlias[]> {
  const res = await api.get<ApiListResponse<ModelRoutingAlias[]>>(
    '/api/model_routing_aliases/'
  )
  return res.data.data ?? []
}

export async function createModelRoutingAlias(
  data: Pick<ModelRoutingAlias, 'public_name' | 'internal_name' | 'note'>
): Promise<ApiListResponse<ModelRoutingAlias>> {
  const res = await api.post('/api/model_routing_aliases/', data)
  return res.data
}

export async function updateModelRoutingAlias(
  data: Pick<ModelRoutingAlias, 'id' | 'public_name' | 'internal_name' | 'note'>
): Promise<ApiListResponse<ModelRoutingAlias>> {
  const res = await api.put('/api/model_routing_aliases/', data)
  return res.data
}

export async function deleteModelRoutingAlias(
  id: number
): Promise<ApiListResponse<null>> {
  const res = await api.delete(`/api/model_routing_aliases/${id}`)
  return res.data
}

export async function getModelPublicNameRegistryStatus(): Promise<ModelPublicNameRegistryStatus> {
  const res = await api.get<ApiListResponse<ModelPublicNameRegistryStatus>>(
    '/api/model_public_name_registry/status'
  )
  return res.data.data ?? { ready: false, collisions: {}, missing_aliases: [] }
}
