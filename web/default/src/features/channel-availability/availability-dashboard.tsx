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
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { getChannelMonitors } from './api'
import { statusBarClass } from './constants'
import { StatusBadge } from './status'
import type { PublicChannelMonitorItem, PublicMonitorCategory } from './types'

const categories: PublicMonitorCategory[] = ['text', 'image', 'video']

function formatPercent(value: number | null): string {
  return value == null ? '-' : `${value.toFixed(2)}%`
}

function formatLatency(value: number | null): string {
  return value == null ? '-' : `${value} ms`
}

function Metric(props: { label: string; value: React.ReactNode }) {
  return (
    <div className='min-w-0 px-3 first:pl-0 last:pr-0 [&+&]:border-l'>
      <div className='text-muted-foreground mb-1 text-xs'>{props.label}</div>
      <div className='truncate text-sm font-semibold'>{props.value}</div>
    </div>
  )
}

function StatusTimeline(props: { item: PublicChannelMonitorItem }) {
  const { t } = useTranslation()
  if (props.item.timeline.length === 0) return null

  return (
    <div
      className='mt-4 flex h-8 items-stretch gap-0.5 border-t pt-3'
      aria-label={t('Recent probe timeline')}
    >
      {props.item.timeline.map((point, index) => (
        <span
          key={index}
          className={cn(
            'min-w-0 flex-1 rounded-[1px]',
            statusBarClass[point.status],
            point.carried && 'opacity-30'
          )}
          aria-hidden='true'
        />
      ))}
    </div>
  )
}

function AvailabilityCards(props: { items: PublicChannelMonitorItem[] }) {
  const { t } = useTranslation()
  if (props.items.length === 0) {
    return (
      <div className='text-muted-foreground rounded-lg border p-8 text-center text-sm'>
        {t('No availability data in this category')}
      </div>
    )
  }

  return (
    <div className='grid gap-3 lg:grid-cols-2 2xl:grid-cols-3'>
      {props.items.map((item) => (
        <Card key={`${item.category}-${item.name}`} className='rounded-lg'>
          <CardHeader>
            <CardTitle className='truncate'>{item.name}</CardTitle>
            <CardAction>
              <StatusBadge status={item.latest_status} />
            </CardAction>
          </CardHeader>
          <CardContent>
            <div className='grid grid-cols-2 gap-y-4'>
              <Metric
                label={t('Availability')}
                value={formatPercent(item.availability)}
              />
              <Metric
                label={t('Average latency')}
                value={formatLatency(item.average_latency_ms)}
              />
            </div>
            <StatusTimeline item={item} />
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

export function AvailabilityDashboard(props: {
  windowDays: number
  onWindowDaysChange: (days: number) => void
}) {
  const { t } = useTranslation()
  const [autoRefresh, setAutoRefresh] = useState(true)
  const listQuery = useQuery({
    queryKey: ['channel-monitors', props.windowDays],
    queryFn: () => getChannelMonitors(props.windowDays),
    refetchInterval: autoRefresh ? 60_000 : false,
  })
  const data = listQuery.data

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
      ) : data.items.length === 0 ? (
        <div className='text-muted-foreground rounded-lg border p-8 text-center text-sm'>
          {t('No visible channel monitors')}
        </div>
      ) : (
        <Tabs defaultValue='text' className='space-y-4'>
          <TabsList>
            <TabsTrigger value='text'>{t('Text')}</TabsTrigger>
            <TabsTrigger value='image'>{t('Image')}</TabsTrigger>
            <TabsTrigger value='video'>{t('Video')}</TabsTrigger>
          </TabsList>
          {categories.map((category) => (
            <TabsContent key={category} value={category}>
              <AvailabilityCards
                items={data.items.filter((item) => item.category === category)}
              />
            </TabsContent>
          ))}
        </Tabs>
      )}
    </div>
  )
}
