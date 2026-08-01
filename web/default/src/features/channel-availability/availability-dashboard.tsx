import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Pause, Play, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { getChannelMonitor, getChannelMonitors } from './api'
import { statusColor } from './constants'
import { StatusBadge } from './status'
import type { ChannelMonitorView } from './types'

function formatPercent(value: number | null): string {
  return value == null ? '-' : `${value.toFixed(2)}%`
}

function formatLatency(value: number | null): string {
  return value == null ? '-' : `${value} ms`
}

function MonitorTimeline(props: { monitor: ChannelMonitorView }) {
  const { t } = useTranslation()
  const timeline = props.monitor.primary.timeline ?? []
  if (timeline.length === 0) {
    return (
      <div className='text-muted-foreground text-xs'>
        {t('No probe samples yet')}
      </div>
    )
  }
  return (
    <div
      className='flex h-7 items-stretch gap-1'
      aria-label={t('Recent probe timeline')}
    >
      {timeline.map((result, index) => (
        <Tooltip key={`${result.checked_at}-${result.status}-${index}`}>
          <TooltipTrigger
            className={cn(
              'min-w-1 flex-1 rounded-sm',
              statusColor[result.status]
            )}
            aria-label={`${result.status} ${new Date(result.checked_at * 1000).toLocaleString()}`}
          />
          <TooltipContent>
            {new Date(result.checked_at * 1000).toLocaleString()} ·{' '}
            {formatLatency(result.latency_ms)}
          </TooltipContent>
        </Tooltip>
      ))}
    </div>
  )
}

function MonitorDetail(props: {
  monitor: ChannelMonitorView | null
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useTranslation()
  return (
    <Dialog open={props.monitor != null} onOpenChange={props.onOpenChange}>
      <DialogContent className='sm:max-w-xl'>
        {props.monitor && (
          <>
            <DialogHeader>
              <DialogTitle>{props.monitor.name}</DialogTitle>
              <DialogDescription>
                {props.monitor.provider} · {props.monitor.primary_model}
              </DialogDescription>
            </DialogHeader>
            <div className='grid grid-cols-2 gap-3 sm:grid-cols-4'>
              <Metric
                label={t('Status')}
                value={
                  <StatusBadge status={props.monitor.primary.latest_status} />
                }
              />
              <Metric
                label={t('Availability')}
                value={formatPercent(props.monitor.primary.availability)}
              />
              <Metric
                label={t('Average latency')}
                value={formatLatency(props.monitor.primary.average_latency_ms)}
              />
              <Metric
                label={t('Observed checks')}
                value={String(props.monitor.primary.observed_checks)}
              />
            </div>
            <div className='space-y-2'>
              <div className='text-sm font-medium'>
                {t('Recent probe timeline')}
              </div>
              <MonitorTimeline monitor={props.monitor} />
            </div>
            {props.monitor.extra_models.length > 0 && (
              <div className='divide-y rounded-lg border'>
                {props.monitor.extra_models.map((item) => (
                  <div
                    key={item.model}
                    className='flex items-center justify-between gap-3 p-3'
                  >
                    <span className='min-w-0 truncate text-sm'>
                      {item.model}
                    </span>
                    <div className='flex items-center gap-3'>
                      <span className='text-muted-foreground text-xs'>
                        {formatPercent(item.availability)}
                      </span>
                      <StatusBadge status={item.latest_status} />
                    </div>
                  </div>
                ))}
              </div>
            )}
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}

function Metric(props: { label: string; value: React.ReactNode }) {
  return (
    <div className='min-w-0 rounded-lg border p-3'>
      <div className='text-muted-foreground mb-1 text-xs'>{props.label}</div>
      <div className='truncate text-sm font-semibold'>{props.value}</div>
    </div>
  )
}

function InlineMetric(props: { label: string; value: React.ReactNode }) {
  return (
    <div className='min-w-0 px-3 first:pl-0 last:pr-0 [&+&]:border-l'>
      <div className='text-muted-foreground mb-1 text-xs'>{props.label}</div>
      <div className='truncate text-sm font-semibold'>{props.value}</div>
    </div>
  )
}

export function AvailabilityDashboard(props: {
  windowDays: number
  onWindowDaysChange: (days: number) => void
}) {
  const { t } = useTranslation()
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [selected, setSelected] = useState<ChannelMonitorView | null>(null)
  const listQuery = useQuery({
    queryKey: ['channel-monitors', props.windowDays],
    queryFn: () => getChannelMonitors(props.windowDays),
    refetchInterval: autoRefresh ? 60_000 : false,
  })
  const detailQuery = useQuery({
    queryKey: ['channel-monitor', selected?.id, props.windowDays],
    queryFn: () => getChannelMonitor(selected!.id, props.windowDays),
    enabled: selected != null,
  })
  const data = listQuery.data
  const coverage = data?.summary.visible_monitors
    ? (data.summary.observed_monitors * 100) / data.summary.visible_monitors
    : 0

  return (
    <div className='space-y-4'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <Tabs
          value={String(props.windowDays)}
          onValueChange={(value) => props.onWindowDaysChange(Number(value))}
        >
          <TabsList>
            {[7, 15, 30].map((days) => (
              <TabsTrigger key={days} value={String(days)}>
                {t('{{count}} days', { count: days })}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
        <div className='flex items-center gap-1'>
          <Button
            variant='outline'
            size='icon'
            onClick={() => setAutoRefresh((value) => !value)}
            aria-label={
              autoRefresh
                ? t('Pause automatic refresh')
                : t('Resume automatic refresh')
            }
          >
            {autoRefresh ? <Pause /> : <Play />}
          </Button>
          <Button
            variant='outline'
            size='icon'
            onClick={() => listQuery.refetch()}
            disabled={listQuery.isFetching}
            aria-label={t('Refresh')}
          >
            <RefreshCw className={cn(listQuery.isFetching && 'animate-spin')} />
          </Button>
        </div>
      </div>

      {listQuery.isLoading ? (
        <div className='grid gap-3 md:grid-cols-3'>
          {[1, 2, 3].map((item) => (
            <Skeleton key={item} className='h-28' />
          ))}
        </div>
      ) : listQuery.isError ? (
        <div className='border-destructive/30 text-destructive rounded-lg border p-4 text-sm'>
          {t('Failed to load channel availability')}
        </div>
      ) : !data?.summary.enabled ? (
        <div className='text-muted-foreground rounded-lg border p-6 text-center text-sm'>
          {t('Channel availability monitoring is disabled')}
        </div>
      ) : (
        <>
          <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-5'>
            <Metric
              label={t('Observed coverage')}
              value={`${coverage.toFixed(0)}%`}
            />
            <Metric
              label={t('Operational')}
              value={String(data.summary.operational)}
            />
            <Metric
              label={t('Degraded')}
              value={String(data.summary.degraded)}
            />
            <Metric
              label={t('Unavailable')}
              value={String(data.summary.unavailable ?? 0)}
            />
            <Metric
              label={t('Unknown')}
              value={String(data.summary.unknown ?? 0)}
            />
          </div>
          {data.items.length === 0 ? (
            <div className='text-muted-foreground rounded-lg border p-8 text-center text-sm'>
              {t('No visible channel monitors')}
            </div>
          ) : (
            <div className='grid gap-3 lg:grid-cols-2 2xl:grid-cols-3'>
              {data.items.map((monitor) => (
                <Card key={monitor.id} className='rounded-lg'>
                  <CardHeader>
                    <CardTitle className='truncate'>{monitor.name}</CardTitle>
                    <CardDescription className='truncate'>
                      {monitor.provider} · {monitor.primary_model}
                    </CardDescription>
                    <CardAction>
                      <StatusBadge status={monitor.primary.latest_status} />
                    </CardAction>
                  </CardHeader>
                  <CardContent className='space-y-4'>
                    <div className='grid grid-cols-3 gap-3'>
                      <InlineMetric
                        label={t('Availability')}
                        value={formatPercent(monitor.primary.availability)}
                      />
                      <InlineMetric
                        label={t('Average latency')}
                        value={formatLatency(
                          monitor.primary.average_latency_ms
                        )}
                      />
                      <InlineMetric
                        label={t('Samples')}
                        value={String(monitor.primary.observed_checks)}
                      />
                    </div>
                    <MonitorTimeline monitor={monitor} />
                    <Button
                      variant='outline'
                      className='w-full'
                      onClick={() => setSelected(monitor)}
                    >
                      {t('View details')}
                    </Button>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </>
      )}
      <MonitorDetail
        monitor={detailQuery.data ?? selected}
        onOpenChange={(open) => !open && setSelected(null)}
      />
    </div>
  )
}
