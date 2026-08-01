import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Edit, Play, Plus, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
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
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import {
  createChannelMonitor,
  deleteChannelMonitor,
  getAdminChannelMonitors,
  runChannelMonitor,
  updateChannelMonitor,
  updateChannelMonitorSettings,
} from './api'
import { StatusBadge } from './status'
import type { AdminChannelMonitorView, ChannelMonitorInput } from './types'

const emptyInput: ChannelMonitorInput = {
  channel_id: 0,
  scope: 'text',
  target: '',
  name: '',
  primary_model: '',
  extra_models: [],
  interval_seconds: 300,
  jitter_seconds: 30,
  enabled: true,
  visible: true,
}

type MonitorScope = ChannelMonitorInput['scope']

function MonitorFormDialog(props: {
  textTargets: Array<{ group: string; models: string[] }>
  monitor: AdminChannelMonitorView | null
  open: boolean
  pending: boolean
  onOpenChange: (open: boolean) => void
  onSubmit: (input: ChannelMonitorInput) => void
}) {
  const { t } = useTranslation()
  const [form, setForm] = useState<ChannelMonitorInput>(() =>
    props.monitor
      ? {
          channel_id: props.monitor.channel_id,
          scope:
            props.monitor.scope === 'media' ? 'image' : props.monitor.scope,
          target: props.monitor.target || props.monitor.group,
          name: props.monitor.name,
          primary_model: props.monitor.primary_model,
          extra_models: props.monitor.extra_model_names,
          interval_seconds: props.monitor.interval_seconds,
          jitter_seconds: props.monitor.jitter_seconds,
          enabled: props.monitor.enabled,
          visible: props.monitor.visible,
        }
      : emptyInput
  )
  const textModels =
    props.textTargets.find((target) => target.group === form.target)?.models ??
    []

  const submit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    props.onSubmit({
      ...form,
      extra_models: [],
    })
  }

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='max-h-[90vh] overflow-y-auto sm:max-w-lg'>
        <form onSubmit={submit} className='space-y-4'>
          <DialogHeader>
            <DialogTitle>
              {props.monitor ? t('Edit monitor') : t('Create monitor')}
            </DialogTitle>
            <DialogDescription>
              {t('Choose text probing or automatic media aggregation.')}
            </DialogDescription>
          </DialogHeader>
          <div className='grid gap-4 sm:grid-cols-2'>
            <div className='space-y-2 sm:col-span-2'>
              <Label htmlFor='monitor-scope'>{t('Monitor type')}</Label>
              <NativeSelect
                id='monitor-scope'
                className='w-full'
                value={form.scope}
                onChange={(event) =>
                  setForm((value) => ({
                    ...value,
                    scope: event.target.value as MonitorScope,
                    channel_id: 0,
                    target: '',
                    primary_model: '',
                    interval_seconds:
                      event.target.value === 'text' ? 300 : 1800,
                    jitter_seconds: event.target.value === 'text' ? 30 : 0,
                  }))
                }
              >
                <NativeSelectOption value='text'>
                  {t('Text probe')}
                </NativeSelectOption>
                <NativeSelectOption value='image'>
                  {t('Image aggregation')}
                </NativeSelectOption>
                <NativeSelectOption value='video'>
                  {t('Video aggregation')}
                </NativeSelectOption>
              </NativeSelect>
            </div>
            {form.scope === 'text' && (
              <div className='space-y-2 sm:col-span-2'>
                <Label htmlFor='monitor-group'>{t('Group')}</Label>
                <NativeSelect
                  id='monitor-group'
                  className='w-full'
                  value={form.target}
                  onChange={(event) =>
                    setForm((value) => ({
                      ...value,
                      target: event.target.value,
                      primary_model: '',
                    }))
                  }
                  required
                >
                  <NativeSelectOption value='' disabled>
                    {t('Select a group')}
                  </NativeSelectOption>
                  {props.textTargets.map((target) => (
                    <NativeSelectOption key={target.group} value={target.group}>
                      {target.group}
                    </NativeSelectOption>
                  ))}
                </NativeSelect>
              </div>
            )}
            {form.scope === 'text' && (
              <div className='space-y-2 sm:col-span-2'>
                <Label htmlFor='monitor-primary-model'>
                  {t('Probe model')}
                </Label>
                <NativeSelect
                  id='monitor-primary-model'
                  className='w-full'
                  value={form.primary_model}
                  onChange={(event) =>
                    setForm((value) => ({
                      ...value,
                      primary_model: event.target.value,
                    }))
                  }
                  required
                >
                  <NativeSelectOption value='' disabled>
                    {t('Select a model')}
                  </NativeSelectOption>
                  {textModels.map((model) => (
                    <NativeSelectOption key={model} value={model}>
                      {model}
                    </NativeSelectOption>
                  ))}
                </NativeSelect>
              </div>
            )}
            {form.scope !== 'text' && (
              <div className='text-muted-foreground rounded-lg border p-3 text-sm sm:col-span-2'>
                {t(
                  'All currently listed models of this type are aggregated automatically every 30 minutes.'
                )}
              </div>
            )}
          </div>
          <div className='flex items-center justify-between rounded-lg border p-3'>
            <Label htmlFor='monitor-enabled'>{t('Monitoring enabled')}</Label>
            <Switch
              id='monitor-enabled'
              checked={form.enabled}
              onCheckedChange={(checked) =>
                setForm((value) => ({ ...value, enabled: checked }))
              }
            />
          </div>
          <div className='flex items-center justify-between rounded-lg border p-3'>
            <Label htmlFor='monitor-visible'>{t('Visible to users')}</Label>
            <Switch
              id='monitor-visible'
              checked={form.visible}
              onCheckedChange={(checked) =>
                setForm((value) => ({ ...value, visible: checked }))
              }
            />
          </div>
          <DialogFooter>
            <Button
              type='button'
              variant='outline'
              onClick={() => props.onOpenChange(false)}
            >
              {t('Cancel')}
            </Button>
            <Button type='submit' disabled={props.pending}>
              {t('Save')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

export function AdminMonitorPanel() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<AdminChannelMonitorView | null>(null)
  const monitorsQuery = useQuery({
    queryKey: ['channel-monitors', 'admin'],
    queryFn: () => getAdminChannelMonitors(7),
  })
  const invalidate = async () => {
    await queryClient.invalidateQueries({ queryKey: ['channel-monitors'] })
  }
  const saveMutation = useMutation({
    mutationFn: (input: ChannelMonitorInput) =>
      editing
        ? updateChannelMonitor(editing.id, input)
        : createChannelMonitor(input),
    onSuccess: async () => {
      toast.success(t('Monitor saved'))
      setDialogOpen(false)
      setEditing(null)
      await invalidate()
    },
  })
  const runMutation = useMutation({
    mutationFn: runChannelMonitor,
    onSuccess: async () => {
      toast.success(t('Probe completed'))
      await invalidate()
    },
  })
  const deleteMutation = useMutation({
    mutationFn: deleteChannelMonitor,
    onSuccess: async () => {
      toast.success(t('Monitor deleted'))
      await invalidate()
    },
  })
  const settingsMutation = useMutation({
    mutationFn: updateChannelMonitorSettings,
    onSuccess: invalidate,
  })

  return (
    <div className='space-y-4'>
      <div className='flex flex-wrap items-center justify-between gap-3'>
        <div className='text-muted-foreground text-sm'>
          {t(
            'Text models use active probes. Media models use recent real tasks without generation cost.'
          )}
        </div>
        <div className='flex items-center gap-3'>
          <div className='flex items-center gap-2'>
            <Label htmlFor='channel-monitor-global-enabled'>
              {t('Monitoring enabled')}
            </Label>
            <Switch
              id='channel-monitor-global-enabled'
              checked={monitorsQuery.data?.summary.enabled ?? false}
              disabled={settingsMutation.isPending || !monitorsQuery.data}
              onCheckedChange={(checked) => settingsMutation.mutate(checked)}
            />
          </div>
          <Button
            onClick={() => {
              setEditing(null)
              setDialogOpen(true)
            }}
          >
            <Plus />
            {t('Create monitor')}
          </Button>
        </div>
      </div>
      {monitorsQuery.isLoading ? (
        <div className='grid gap-3 lg:grid-cols-2'>
          {[1, 2].map((item) => (
            <Skeleton key={item} className='h-40' />
          ))}
        </div>
      ) : monitorsQuery.data?.items.length ? (
        <div className='grid gap-3 lg:grid-cols-2'>
          {monitorsQuery.data.items.map((monitor) => (
            <Card key={monitor.id} className='rounded-lg'>
              <CardHeader>
                <CardTitle>{monitor.name}</CardTitle>
                <CardDescription>
                  {monitor.scope === 'text'
                    ? `${monitor.group} · ${monitor.primary_model}`
                    : t(
                        monitor.scope === 'video'
                          ? 'Video aggregation'
                          : 'Image aggregation'
                      )}
                </CardDescription>
                {monitor.scope === 'text' && (
                  <CardAction>
                    <StatusBadge status={monitor.primary.latest_status} />
                  </CardAction>
                )}
              </CardHeader>
              <CardContent className='space-y-3'>
                <div className='text-muted-foreground flex flex-wrap gap-x-4 gap-y-1 text-xs'>
                  <span>
                    {t('Refresh cycle')}:{' '}
                    {monitor.scope === 'text'
                      ? t('5 minutes')
                      : t('30 minutes')}
                  </span>
                  <span>{monitor.enabled ? t('Enabled') : t('Disabled')}</span>
                  <span>{monitor.visible ? t('Visible') : t('Hidden')}</span>
                </div>
                {monitor.probe_kind === 'media_passive' && (
                  <div className='rounded-lg border border-sky-200 bg-sky-50 p-2 text-xs text-sky-700 dark:border-sky-900 dark:bg-sky-950 dark:text-sky-300'>
                    {t(
                      'Media status is refreshed from recent real tasks; no generation request is sent.'
                    )}
                  </div>
                )}
                <div className='flex justify-end gap-1'>
                  {monitor.scope === 'text' && (
                    <Button
                      variant='outline'
                      size='icon'
                      aria-label={t('Run now')}
                      disabled={runMutation.isPending}
                      onClick={() => runMutation.mutate(monitor.id)}
                    >
                      <Play />
                    </Button>
                  )}
                  <Button
                    variant='outline'
                    size='icon'
                    aria-label={t('Edit')}
                    onClick={() => {
                      setEditing(monitor)
                      setDialogOpen(true)
                    }}
                  >
                    <Edit />
                  </Button>
                  <Button
                    variant='destructive'
                    size='icon'
                    aria-label={t('Delete')}
                    disabled={deleteMutation.isPending}
                    onClick={() => {
                      if (
                        window.confirm(
                          t('Delete this monitor and its history?')
                        )
                      )
                        deleteMutation.mutate(monitor.id)
                    }}
                  >
                    <Trash2 />
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : (
        <div className='text-muted-foreground rounded-lg border p-8 text-center text-sm'>
          {t('No channel monitors configured')}
        </div>
      )}
      {dialogOpen && (
        <MonitorFormDialog
          key={editing?.id ?? 'new'}
          textTargets={monitorsQuery.data?.text_targets ?? []}
          monitor={editing}
          open
          pending={saveMutation.isPending}
          onOpenChange={setDialogOpen}
          onSubmit={(input) => saveMutation.mutate(input)}
        />
      )}
    </div>
  )
}
