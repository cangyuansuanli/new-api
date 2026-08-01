export type MonitorStatus =
  | 'operational'
  | 'degraded'
  | 'unavailable'
  | 'unknown'
  | 'not_monitored'

export type ChannelMonitorResult = {
  status: MonitorStatus
  latency_ms: number | null
  checked_at: number
}

export type ChannelMonitorModelStat = {
  model: string
  latest_status: MonitorStatus
  latest_latency_ms: number | null
  availability: number | null
  average_latency_ms: number | null
  observed_checks: number
  operational_checks: number
  latest_checked_at: number | null
  timeline?: ChannelMonitorResult[]
}

export type AdminChannelMonitorView = {
  id: number
  name: string
  provider: string
  probe_kind: 'text_active' | 'media_passive' | 'media_disabled'
  scope: 'text' | 'image' | 'video' | 'media'
  enabled: boolean
  visible: boolean
  interval_seconds: number
  primary_model: string
  primary: ChannelMonitorModelStat
  extra_models: ChannelMonitorModelStat[]
  window_days: number
  channel_id: number
  channel_name: string
  group: string
  target: string
  jitter_seconds: number
  extra_model_names: string[]
}

export type ChannelMonitorSummary = {
  enabled: boolean
  visible_monitors: number
  observed_monitors: number
  operational: number
  degraded: number
  unavailable: number
  unknown: number
}

export type AdminChannelMonitorList = {
  items: AdminChannelMonitorView[]
  summary: ChannelMonitorSummary
  text_targets: Array<{ group: string; models: string[] }>
}

export type PublicMonitorCategory = 'text' | 'image' | 'video'

export type PublicChannelMonitorItem = {
  name: string
  category: PublicMonitorCategory
  latest_status: MonitorStatus
  availability: number | null
  average_latency_ms: number | null
  timeline: Array<{ status: MonitorStatus; carried?: boolean }>
}

export type PublicChannelMonitorSummary = {
  enabled: boolean
  total: number
  operational: number
  degraded: number
  unavailable: number
  unknown: number
}

export type PublicChannelMonitorList = {
  items: PublicChannelMonitorItem[]
  summary: PublicChannelMonitorSummary
}

export type ChannelMonitorInput = {
  channel_id: number
  scope: 'text' | 'image' | 'video'
  target: string
  name: string
  primary_model: string
  extra_models: string[]
  interval_seconds: number
  jitter_seconds: number
  enabled: boolean
  visible: boolean
}
