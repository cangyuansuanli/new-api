import { api } from '@/lib/api'
import type {
  AdminChannelMonitorView,
  ChannelMonitorInput,
  ChannelMonitorList,
  ChannelMonitorView,
} from './types'

type ApiEnvelope<T> = {
  success: boolean
  message?: string
  data: T
}

function unwrap<T>(response: ApiEnvelope<T>): T {
  if (!response.success) {
    throw new Error(response.message || 'Channel monitor request failed')
  }
  return response.data
}

async function assertSuccess(request: Promise<{ data: ApiEnvelope<unknown> }>) {
  const response = await request
  unwrap(response.data)
}

export async function getChannelMonitors(
  windowDays: number
): Promise<ChannelMonitorList> {
  const response = await api.get<ApiEnvelope<ChannelMonitorList>>(
    '/api/channel-monitors',
    { params: { window_days: windowDays } }
  )
  return unwrap(response.data)
}

export async function getChannelMonitor(
  id: number,
  windowDays: number
): Promise<ChannelMonitorView> {
  const response = await api.get<ApiEnvelope<ChannelMonitorView>>(
    `/api/channel-monitors/${id}/status`,
    { params: { window_days: windowDays } }
  )
  return unwrap(response.data)
}

export async function getAdminChannelMonitors(
  windowDays: number
): Promise<ChannelMonitorList<AdminChannelMonitorView>> {
  const response = await api.get<
    ApiEnvelope<ChannelMonitorList<AdminChannelMonitorView>>
  >('/api/channel-monitors/admin', { params: { window_days: windowDays } })
  return unwrap(response.data)
}

export async function createChannelMonitor(
  input: ChannelMonitorInput
): Promise<void> {
  await assertSuccess(api.post('/api/channel-monitors/admin', input))
}

export async function updateChannelMonitor(
  id: number,
  input: ChannelMonitorInput
): Promise<void> {
  await assertSuccess(api.put(`/api/channel-monitors/admin/${id}`, input))
}

export async function deleteChannelMonitor(id: number): Promise<void> {
  await assertSuccess(api.delete(`/api/channel-monitors/admin/${id}`))
}

export async function runChannelMonitor(id: number): Promise<void> {
  await assertSuccess(api.post(`/api/channel-monitors/admin/${id}/run`))
}

export async function updateChannelMonitorSettings(
  enabled: boolean
): Promise<void> {
  await assertSuccess(
    api.put('/api/channel-monitors/admin/settings', { enabled })
  )
}
