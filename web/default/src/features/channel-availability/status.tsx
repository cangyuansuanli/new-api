import {
  CircleAlert,
  CircleCheck,
  CircleHelp,
  CircleOff,
  CircleX,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import type { MonitorStatus } from './types'

const statusBadgeClass: Record<MonitorStatus, string> = {
  operational:
    'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900 dark:bg-emerald-950 dark:text-emerald-300',
  degraded:
    'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-900 dark:bg-amber-950 dark:text-amber-300',
  unavailable:
    'border-red-200 bg-red-50 text-red-700 dark:border-red-900 dark:bg-red-950 dark:text-red-300',
  unknown:
    'border-zinc-200 bg-zinc-50 text-zinc-700 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-300',
  not_monitored:
    'border-sky-200 bg-sky-50 text-sky-700 dark:border-sky-900 dark:bg-sky-950 dark:text-sky-300',
}

export function StatusBadge(props: { status: MonitorStatus }) {
  const { t } = useTranslation()
  const statusConfig = {
    operational: { icon: CircleCheck, label: t('Operational') },
    degraded: { icon: CircleAlert, label: t('Degraded') },
    unavailable: { icon: CircleX, label: t('Unavailable') },
    unknown: { icon: CircleHelp, label: t('Unknown') },
    not_monitored: { icon: CircleOff, label: t('Not monitored') },
  }[props.status]
  const Icon = statusConfig.icon

  return (
    <Badge variant='outline' className={cn(statusBadgeClass[props.status])}>
      <Icon aria-hidden='true' />
      {statusConfig.label}
    </Badge>
  )
}
