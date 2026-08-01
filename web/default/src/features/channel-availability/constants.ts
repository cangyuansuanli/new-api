import type { MonitorStatus } from './types'

export const statusBarClass: Record<MonitorStatus, string> = {
  operational: 'bg-emerald-500',
  degraded: 'bg-amber-500',
  unavailable: 'bg-red-500',
  unknown: 'bg-zinc-400',
  not_monitored: 'bg-sky-500',
}
