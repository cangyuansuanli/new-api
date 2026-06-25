/*
Copyright (C) 2023-2026 QuantumNous
*/
import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ArrowDown, ArrowUp, Pencil, Plus, Search, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { Dialog } from '@/components/dialog'
import { TagInput } from '@/components/tag-input'
import { StatusBadge } from '@/components/status-badge'
import { getModelPublicNameRegistryStatus } from '../model-naming-api'
import { modelNamingQueryKeys } from '../model-naming-types'
import {
  createModelUiParamProfile,
  deleteModelUiParamProfile,
  getModelUiParamRegistry,
  listModelUiParamProfiles,
  previewModelUiParamMatch,
  reorderModelUiParamProfiles,
  updateModelUiParamProfile,
  updateModelUiParamRegistry,
} from '../model-params-api'
import {
  IMAGE_PARAM_KEYS,
  modelParamsQueryKeys,
  VIDEO_API_MODES,
  VIDEO_PARAM_KEYS,
  type ModelUiParamCapability,
  type ModelUiParamProfile,
} from '../model-params-types'

const CHANNEL_PREFIXES = [
  '119337-',
  'aini-',
  'byte-',
  'ctlove-',
  'czeq-',
  'go2api-',
  'gz-',
  'happyhorse-',
  'niming-',
  'oairegbox-',
  'yunwu-',
  'zeabur-',
]

function parseJsonArray(raw: string): string[] {
  try {
    const parsed = JSON.parse(raw || '[]')
    return Array.isArray(parsed) ? parsed.map(String) : []
  } catch {
    return []
  }
}

function parseJsonObject(raw: string): Record<string, unknown> {
  try {
    const parsed = JSON.parse(raw || '{}')
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed)
      ? (parsed as Record<string, unknown>)
      : {}
  } catch {
    return {}
  }
}

function stringifyJson(value: unknown, fallback: string) {
  try {
    return JSON.stringify(value ?? JSON.parse(fallback), null, 2)
  } catch {
    return fallback
  }
}

function validateMatchTokens(tokens: string[]) {
  for (const token of tokens) {
    const lower = token.toLowerCase()
    for (const prefix of CHANNEL_PREFIXES) {
      if (lower.startsWith(prefix)) {
        return `match 须为 public 名，不可含渠道前缀：${token}`
      }
    }
  }
  return ''
}

type ProfileFormState = {
  profile_id: string
  match: string[]
  sort_order: string
  api_mode: string
  requires_reference_media: boolean
  poll_status: string
  poll: string
  reference_limits: string
  params: string
  option_rules: string
  hints: string
  note: string
}

const emptyProfileForm = (): ProfileFormState => ({
  profile_id: '',
  match: [],
  sort_order: '0',
  api_mode: '',
  requires_reference_media: false,
  poll_status: '',
  poll: '{}',
  reference_limits: '{"images":0,"videos":0,"audios":0}',
  params: '{}',
  option_rules: '[]',
  hints: '[]',
  note: '',
})

function profileToForm(profile: ModelUiParamProfile): ProfileFormState {
  return {
    profile_id: profile.profile_id,
    match: parseJsonArray(profile.match),
    sort_order: String(profile.sort_order),
    api_mode: profile.api_mode || '',
    requires_reference_media: profile.requires_reference_media,
    poll_status: profile.poll_status || '',
    poll: stringifyJson(JSON.parse(profile.poll || '{}'), '{}'),
    reference_limits: stringifyJson(
      JSON.parse(profile.reference_limits || '{}'),
      '{"images":0,"videos":0,"audios":0}'
    ),
    params: stringifyJson(JSON.parse(profile.params || '{}'), '{}'),
    option_rules: stringifyJson(JSON.parse(profile.option_rules || '[]'), '[]'),
    hints: stringifyJson(JSON.parse(profile.hints || '[]'), '[]'),
    note: profile.note || '',
  }
}

function setParamEnabled(
  paramsJson: string,
  key: string,
  enabled: boolean
): string {
  const params = parseJsonObject(paramsJson)
  const current = (params[key] as Record<string, unknown>) || {}
  params[key] = { ...current, enabled }
  return JSON.stringify(params, null, 2)
}

function readParamEnabled(paramsJson: string, key: string) {
  const params = parseJsonObject(paramsJson)
  const current = params[key] as { enabled?: boolean } | undefined
  return Boolean(current?.enabled)
}

export function ModelParamsSection() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [capability, setCapability] = useState<ModelUiParamCapability>('video')
  const [previewModel, setPreviewModel] = useState('')
  const [profileDialogOpen, setProfileDialogOpen] = useState(false)
  const [editingProfile, setEditingProfile] = useState<ModelUiParamProfile | null>(
    null
  )
  const [profileForm, setProfileForm] = useState<ProfileFormState>(emptyProfileForm)
  const [deleteProfileId, setDeleteProfileId] = useState<number | null>(null)

  const invalidate = async () => {
    await Promise.all([
      queryClient.invalidateQueries({
        queryKey: modelParamsQueryKeys.registry(capability),
      }),
      queryClient.invalidateQueries({
        queryKey: modelParamsQueryKeys.profiles(capability),
      }),
    ])
  }

  const { data: registry, isLoading: registryLoading } = useQuery({
    queryKey: modelParamsQueryKeys.registry(capability),
    queryFn: () => getModelUiParamRegistry(capability),
  })

  const { data: profiles = [], isLoading: profilesLoading } = useQuery({
    queryKey: modelParamsQueryKeys.profiles(capability),
    queryFn: () => listModelUiParamProfiles(capability),
  })

  const { data: namingStatus } = useQuery({
    queryKey: modelNamingQueryKeys.status(),
    queryFn: getModelPublicNameRegistryStatus,
  })

  const { data: matchPreview } = useQuery({
    queryKey: modelParamsQueryKeys.preview(capability, previewModel.trim()),
    queryFn: () => previewModelUiParamMatch(capability, previewModel.trim()),
    enabled: previewModel.trim().length > 0,
  })

  const registryForm = useMemo(
    () => ({
      default_profile_id: registry?.default_profile_id ?? '',
      capability_fallback: registry?.capability_fallback ?? '[]',
      poll_defaults: registry?.poll_defaults ?? '{}',
    }),
    [registry]
  )

  const [registryDraft, setRegistryDraft] = useState(registryForm)

  useEffect(() => {
    setRegistryDraft(registryForm)
  }, [registryForm])

  const saveRegistryMutation = useMutation({
    mutationFn: () =>
      updateModelUiParamRegistry(capability, {
        default_profile_id: registryDraft.default_profile_id.trim(),
        capability_fallback: registryDraft.capability_fallback,
        poll_defaults:
          capability === 'video' ? registryDraft.poll_defaults : undefined,
      }),
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t('Operation failed'))
        return
      }
      toast.success(t('Registry settings saved'))
      toast.message(t('Canvas users may need to refresh the page'))
      await invalidate()
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const saveProfileMutation = useMutation({
    mutationFn: async () => {
      const matchError = validateMatchTokens(profileForm.match)
      if (matchError) throw new Error(matchError)
      JSON.parse(profileForm.params)
      JSON.parse(profileForm.reference_limits)
      if (capability === 'video') {
        JSON.parse(profileForm.poll)
        JSON.parse(profileForm.option_rules)
        JSON.parse(profileForm.hints)
      }
      const payload = {
        capability,
        profile_id: profileForm.profile_id.trim(),
        match: JSON.stringify(profileForm.match),
        sort_order: Number(profileForm.sort_order) || 0,
        params: profileForm.params,
        note: profileForm.note.trim(),
        ...(capability === 'video'
          ? {
              api_mode: profileForm.api_mode,
              requires_reference_media: profileForm.requires_reference_media,
              poll_status: profileForm.poll_status,
              poll: profileForm.poll,
              reference_limits: profileForm.reference_limits,
              option_rules: profileForm.option_rules,
              hints: profileForm.hints,
            }
          : {}),
      }
      if (editingProfile) {
        return updateModelUiParamProfile({ id: editingProfile.id, ...payload })
      }
      return createModelUiParamProfile(payload)
    },
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t('Operation failed'))
        return
      }
      toast.success(
        editingProfile ? t('Profile updated') : t('Profile created')
      )
      toast.message(t('Canvas users may need to refresh the page'))
      setProfileDialogOpen(false)
      setEditingProfile(null)
      setProfileForm(emptyProfileForm())
      await invalidate()
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const deleteProfileMutation = useMutation({
    mutationFn: (id: number) => deleteModelUiParamProfile(id),
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t('Operation failed'))
        return
      }
      toast.success(t('Profile deleted'))
      setDeleteProfileId(null)
      await invalidate()
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const reorderMutation = useMutation({
    mutationFn: (items: Array<{ id: number; sort_order: number }>) =>
      reorderModelUiParamProfiles(capability, items),
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t('Operation failed'))
        return
      }
      await invalidate()
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const moveProfile = (index: number, direction: -1 | 1) => {
    const target = index + direction
    if (target < 0 || target >= profiles.length) return
    const next = [...profiles]
    const tmp = next[index]!
    next[index] = next[target]!
    next[target] = tmp
    void reorderMutation.mutateAsync(
      next.map((item, sortIndex) => ({
        id: item.id,
        sort_order: sortIndex * 10,
      }))
    )
  }

  const paramKeys =
    capability === 'video' ? VIDEO_PARAM_KEYS : IMAGE_PARAM_KEYS

  return (
    <div className='flex h-full min-h-0 flex-col gap-4'>
      <Tabs
        value={capability}
        onValueChange={(value) => setCapability(value as ModelUiParamCapability)}
      >
        <TabsList>
          <TabsTrigger value='video'>{t('Video parameters')}</TabsTrigger>
          <TabsTrigger value='image'>{t('Image parameters')}</TabsTrigger>
        </TabsList>
      </Tabs>

      <Card>
        <CardHeader className='pb-3'>
          <CardTitle className='text-base'>{t('Registry settings')}</CardTitle>
        </CardHeader>
        <CardContent className='grid gap-3 md:grid-cols-2'>
          <div className='space-y-2'>
            <Label>{t('Default profile id')}</Label>
            <Input
              value={registryDraft.default_profile_id}
              onChange={(event) =>
                setRegistryDraft((prev) => ({
                  ...prev,
                  default_profile_id: event.target.value,
                }))
              }
            />
          </div>
          <div className='space-y-2 md:col-span-2'>
            <Label>{t('Capability fallback tokens')}</Label>
            <Textarea
              rows={3}
              value={registryDraft.capability_fallback}
              onChange={(event) =>
                setRegistryDraft((prev) => ({
                  ...prev,
                  capability_fallback: event.target.value,
                }))
              }
            />
          </div>
          {capability === 'video' ? (
            <div className='space-y-2 md:col-span-2'>
              <Label>{t('Poll defaults (JSON)')}</Label>
              <Textarea
                rows={4}
                value={registryDraft.poll_defaults}
                onChange={(event) =>
                  setRegistryDraft((prev) => ({
                    ...prev,
                    poll_defaults: event.target.value,
                  }))
                }
              />
            </div>
          ) : null}
          <div>
            <Button
              size='sm'
              disabled={registryLoading || saveRegistryMutation.isPending}
              onClick={() => saveRegistryMutation.mutate()}
            >
              {t('Save registry')}
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className='flex flex-row items-center justify-between pb-3'>
          <CardTitle className='text-base'>{t('Match preview')}</CardTitle>
        </CardHeader>
        <CardContent className='space-y-3'>
          <div className='flex gap-2'>
            <Input
              placeholder={t('Public model name')}
              value={previewModel}
              onChange={(event) => setPreviewModel(event.target.value)}
            />
            <Button variant='outline' size='icon' type='button'>
              <Search className='h-4 w-4' />
            </Button>
          </div>
          {previewModel.trim() ? (
            <div className='text-sm'>
              {matchPreview?.matched_profile ? (
                <p>
                  {t('Matched profile')}:{' '}
                  <code>{matchPreview.matched_profile}</code>
                  {matchPreview.collision ? (
                    <StatusBadge variant='warning' className='ml-2'>
                      {t('Collision')}
                    </StatusBadge>
                  ) : null}
                </p>
              ) : (
                <p className='text-muted-foreground'>
                  {t('No profile matched; will use default')}
                </p>
              )}
            </div>
          ) : null}
          {Object.keys(namingStatus?.collisions ?? {}).length > 0 ? (
            <Alert variant='warning'>
              <AlertTitle>{t('Naming collisions detected')}</AlertTitle>
              <AlertDescription className='text-xs'>
                {Object.entries(namingStatus?.collisions ?? {})
                  .slice(0, 5)
                  .map(([name, internals]) => (
                    <div key={name}>
                      {name}: {internals.join(', ')}
                    </div>
                  ))}
              </AlertDescription>
            </Alert>
          ) : null}
        </CardContent>
      </Card>

      <div className='flex items-center justify-between'>
        <h3 className='text-sm font-medium'>{t('Parameter profiles')}</h3>
        <Button
          size='sm'
          onClick={() => {
            setEditingProfile(null)
            setProfileForm(emptyProfileForm())
            setProfileDialogOpen(true)
          }}
        >
          <Plus className='h-4 w-4' />
          {t('Add profile')}
        </Button>
      </div>

      <div className='min-h-0 flex-1 overflow-auto rounded-md border'>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('Profile id')}</TableHead>
              <TableHead>{t('Match')}</TableHead>
              {capability === 'video' ? (
                <TableHead>{t('API mode')}</TableHead>
              ) : null}
              <TableHead>{t('Order')}</TableHead>
              <TableHead className='text-right'>{t('Actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {profilesLoading ? (
              <TableRow>
                <TableCell colSpan={5}>{t('Loading...')}</TableCell>
              </TableRow>
            ) : profiles.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} className='text-muted-foreground'>
                  {t('No profiles yet')}
                </TableCell>
              </TableRow>
            ) : (
              profiles.map((profile, index) => (
                <TableRow key={profile.id}>
                  <TableCell className='font-mono text-xs'>
                    {profile.profile_id}
                  </TableCell>
                  <TableCell className='max-w-xs truncate text-xs'>
                    {parseJsonArray(profile.match).join(', ') || '—'}
                  </TableCell>
                  {capability === 'video' ? (
                    <TableCell>{profile.api_mode || '—'}</TableCell>
                  ) : null}
                  <TableCell>{profile.sort_order}</TableCell>
                  <TableCell className='text-right'>
                    <div className='flex justify-end gap-1'>
                      <Button
                        variant='ghost'
                        size='icon'
                        onClick={() => moveProfile(index, -1)}
                        disabled={index === 0 || reorderMutation.isPending}
                      >
                        <ArrowUp className='h-4 w-4' />
                      </Button>
                      <Button
                        variant='ghost'
                        size='icon'
                        onClick={() => moveProfile(index, 1)}
                        disabled={
                          index === profiles.length - 1 ||
                          reorderMutation.isPending
                        }
                      >
                        <ArrowDown className='h-4 w-4' />
                      </Button>
                      <Button
                        variant='ghost'
                        size='icon'
                        onClick={() => {
                          setEditingProfile(profile)
                          setProfileForm(profileToForm(profile))
                          setProfileDialogOpen(true)
                        }}
                      >
                        <Pencil className='h-4 w-4' />
                      </Button>
                      <Button
                        variant='ghost'
                        size='icon'
                        onClick={() => setDeleteProfileId(profile.id)}
                      >
                        <Trash2 className='h-4 w-4' />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      <Dialog
        open={profileDialogOpen}
        onOpenChange={(open) => {
          setProfileDialogOpen(open)
          if (!open) {
            setEditingProfile(null)
            setProfileForm(emptyProfileForm())
          }
        }}
        title={
          editingProfile ? t('Edit parameter profile') : t('Add parameter profile')
        }
        className='max-w-3xl'
      >
        <div className='max-h-[70vh] space-y-4 overflow-y-auto pr-1'>
          <div className='grid gap-3 md:grid-cols-2'>
            <div className='space-y-2'>
              <Label>{t('Profile id')}</Label>
              <Input
                value={profileForm.profile_id}
                onChange={(event) =>
                  setProfileForm((prev) => ({
                    ...prev,
                    profile_id: event.target.value,
                  }))
                }
              />
            </div>
            <div className='space-y-2'>
              <Label>{t('Sort order')}</Label>
              <Input
                value={profileForm.sort_order}
                onChange={(event) =>
                  setProfileForm((prev) => ({
                    ...prev,
                    sort_order: event.target.value,
                  }))
                }
              />
            </div>
          </div>
          <div className='space-y-2'>
            <Label>{t('Match tokens (public names)')}</Label>
            <TagInput
              value={profileForm.match}
              onChange={(match) => setProfileForm((prev) => ({ ...prev, match }))}
              placeholder={t('Add match token')}
            />
          </div>
          {capability === 'video' ? (
            <>
              <div className='grid gap-3 md:grid-cols-2'>
                <div className='space-y-2'>
                  <Label>{t('API mode')}</Label>
                  <Select
                    value={profileForm.api_mode || '__none__'}
                    onValueChange={(value) =>
                      setProfileForm((prev) => ({
                        ...prev,
                        api_mode: value === '__none__' ? '' : value,
                      }))
                    }
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value='__none__'>{t('None')}</SelectItem>
                      {VIDEO_API_MODES.map((mode) => (
                        <SelectItem key={mode} value={mode}>
                          {mode}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className='space-y-2'>
                  <Label>{t('Poll status')}</Label>
                  <Select
                    value={profileForm.poll_status || '__none__'}
                    onValueChange={(value) =>
                      setProfileForm((prev) => ({
                        ...prev,
                        poll_status: value === '__none__' ? '' : value,
                      }))
                    }
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value='__none__'>{t('None')}</SelectItem>
                      <SelectItem value='strict'>strict</SelectItem>
                      <SelectItem value='relaxed'>relaxed</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <div className='flex items-center gap-2'>
                <Checkbox
                  checked={profileForm.requires_reference_media}
                  onCheckedChange={(checked) =>
                    setProfileForm((prev) => ({
                      ...prev,
                      requires_reference_media: checked === true,
                    }))
                  }
                />
                <Label>{t('Requires reference media')}</Label>
              </div>
              <div className='space-y-2'>
                <Label>{t('Reference limits (JSON)')}</Label>
                <Textarea
                  rows={3}
                  value={profileForm.reference_limits}
                  onChange={(event) =>
                    setProfileForm((prev) => ({
                      ...prev,
                      reference_limits: event.target.value,
                    }))
                  }
                />
              </div>
              <div className='space-y-2'>
                <Label>{t('Poll override (JSON)')}</Label>
                <Textarea
                  rows={3}
                  value={profileForm.poll}
                  onChange={(event) =>
                    setProfileForm((prev) => ({
                      ...prev,
                      poll: event.target.value,
                    }))
                  }
                />
              </div>
            </>
          ) : null}
          <div className='space-y-2'>
            <Label>{t('Parameter toggles')}</Label>
            <div className='grid gap-2 sm:grid-cols-2'>
              {paramKeys.map((key) => (
                <div key={key} className='flex items-center gap-2'>
                  <Checkbox
                    checked={readParamEnabled(profileForm.params, key)}
                    onCheckedChange={(checked) =>
                      setProfileForm((prev) => ({
                        ...prev,
                        params: setParamEnabled(
                          prev.params,
                          key,
                          checked === true
                        ),
                      }))
                    }
                  />
                  <Label className='font-mono text-xs'>{key}</Label>
                </div>
              ))}
            </div>
          </div>
          <div className='space-y-2'>
            <Label>{t('Params (JSON)')}</Label>
            <Textarea
              rows={10}
              className='font-mono text-xs'
              value={profileForm.params}
              onChange={(event) =>
                setProfileForm((prev) => ({
                  ...prev,
                  params: event.target.value,
                }))
              }
            />
          </div>
          {capability === 'video' ? (
            <>
              <div className='space-y-2'>
                <Label>{t('Option rules (JSON)')}</Label>
                <Textarea
                  rows={4}
                  className='font-mono text-xs'
                  value={profileForm.option_rules}
                  onChange={(event) =>
                    setProfileForm((prev) => ({
                      ...prev,
                      option_rules: event.target.value,
                    }))
                  }
                />
              </div>
              <div className='space-y-2'>
                <Label>{t('Hints (JSON)')}</Label>
                <Textarea
                  rows={4}
                  className='font-mono text-xs'
                  value={profileForm.hints}
                  onChange={(event) =>
                    setProfileForm((prev) => ({
                      ...prev,
                      hints: event.target.value,
                    }))
                  }
                />
              </div>
            </>
          ) : null}
          <div className='space-y-2'>
            <Label>{t('Note')}</Label>
            <Input
              value={profileForm.note}
              onChange={(event) =>
                setProfileForm((prev) => ({ ...prev, note: event.target.value }))
              }
            />
          </div>
        </div>
        <div className='mt-4 flex justify-end gap-2'>
          <Button variant='outline' onClick={() => setProfileDialogOpen(false)}>
            {t('Cancel')}
          </Button>
          <Button
            disabled={saveProfileMutation.isPending}
            onClick={() => saveProfileMutation.mutate()}
          >
            {t('Save')}
          </Button>
        </div>
      </Dialog>

      <ConfirmDialog
        open={deleteProfileId !== null}
        onOpenChange={(open) => !open && setDeleteProfileId(null)}
        title={t('Delete profile')}
        description={t('This action cannot be undone.')}
        confirmLabel={t('Delete')}
        onConfirm={() => {
          if (deleteProfileId) deleteProfileMutation.mutate(deleteProfileId)
        }}
      />
    </div>
  )
}
