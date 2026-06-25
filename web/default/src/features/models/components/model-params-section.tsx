/*
Copyright (C) 2023-2026 QuantumNous
*/
import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Pencil, Plus, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { Dialog } from '@/components/dialog'
import { searchModels, updateModel } from '../api'
import { modelsQueryKeys } from '../lib'
import {
  createModelUiParamProfile,
  deleteModelUiParamProfile,
  getModelUiParamRegistry,
  listModelUiParamProfiles,
  updateModelUiParamProfile,
  updateModelUiParamRegistry,
} from '../model-params-api'
import {
  getVideoApiModeLabel,
  getVideoApiModeOptions,
} from '../model-params-labels'
import {
  DEFAULT_IMAGE_PROFILE_ID,
  DEFAULT_VIDEO_PROFILE_ID,
  IMAGE_PARAM_KEYS,
  modelParamsQueryKeys,
  VIDEO_PARAM_KEYS,
  type ModelUiParamCapability,
  type ModelUiParamProfile,
} from '../model-params-types'
import type { Model } from '../types'

function FieldHelp({ children }: { children: ReactNode }) {
  return <p className='text-muted-foreground text-xs'>{children}</p>
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

type ProfileFormState = {
  profile_id: string
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

function ProfileEditorDialog({
  open,
  onOpenChange,
  capability,
  editingProfile,
  onSaved,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  capability: ModelUiParamCapability
  editingProfile: ModelUiParamProfile | null
  onSaved: () => Promise<void>
}) {
  const { t } = useTranslation()
  const [form, setForm] = useState<ProfileFormState>(emptyProfileForm())
  const paramKeys =
    capability === 'video' ? VIDEO_PARAM_KEYS : IMAGE_PARAM_KEYS

  useEffect(() => {
    if (!open) return
    setForm(editingProfile ? profileToForm(editingProfile) : emptyProfileForm())
  }, [open, editingProfile])

  const saveMutation = useMutation({
    mutationFn: async () => {
      const payload = {
        capability,
        profile_id: form.profile_id.trim(),
        api_mode: capability === 'video' ? form.api_mode : undefined,
        requires_reference_media:
          capability === 'video' ? form.requires_reference_media : false,
        poll_status: capability === 'video' ? form.poll_status : undefined,
        poll: capability === 'video' ? form.poll : '{}',
        reference_limits: capability === 'video' ? form.reference_limits : '{}',
        params: form.params,
        option_rules: capability === 'video' ? form.option_rules : '[]',
        hints: capability === 'video' ? form.hints : '[]',
        note: form.note,
      }
      if (editingProfile) {
        return updateModelUiParamProfile({ id: editingProfile.id, ...payload })
      }
      return createModelUiParamProfile(payload)
    },
    onSuccess: async (res) => {
      if (!res.success) {
        toast.error(res.message || t('Save failed'))
        return
      }
      toast.success(t('Saved'))
      await onSaved()
      onOpenChange(false)
    },
    onError: (error: Error) => toast.error(error.message),
  })

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={
        editingProfile ? t('Edit parameter profile') : t('Add parameter profile')
      }
      contentClassName='max-w-3xl'
      footer={
        <>
          <Button variant='outline' onClick={() => onOpenChange(false)}>
            {t('Cancel')}
          </Button>
          <Button
            disabled={saveMutation.isPending || !form.profile_id.trim()}
            onClick={() => saveMutation.mutate()}
          >
            {t('Save')}
          </Button>
        </>
      }
    >
      <div className='max-h-[70vh] space-y-4 overflow-y-auto pr-1'>
        <div className='space-y-2'>
          <Label>{t('Profile id')}</Label>
          <FieldHelp>{t('Profile id help')}</FieldHelp>
          <Input
            value={form.profile_id}
            disabled={Boolean(editingProfile)}
            placeholder={t('Profile id placeholder')}
            onChange={(event) =>
              setForm((prev) => ({ ...prev, profile_id: event.target.value }))
            }
          />
        </div>
        {capability === 'video' ? (
          <>
            <div className='grid gap-3 md:grid-cols-2'>
              <div className='space-y-2 md:col-span-2'>
                <Label>{t('API mode')}</Label>
                <FieldHelp>{t('API mode help')}</FieldHelp>
                <Select
                  value={form.api_mode || '__none__'}
                  onValueChange={(value) => {
                    if (!value) return
                    setForm((prev) => ({
                      ...prev,
                      api_mode: value === '__none__' ? '' : value,
                    }))
                  }}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value='__none__'>{t('None')}</SelectItem>
                    {getVideoApiModeOptions(t).map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {form.api_mode ? (
                  <FieldHelp>
                    {getVideoApiModeOptions(t).find(
                      (option) => option.value === form.api_mode
                    )?.description}
                  </FieldHelp>
                ) : null}
              </div>
              <div className='space-y-2'>
                <Label>{t('Poll status')}</Label>
                <FieldHelp>{t('Poll status help')}</FieldHelp>
                <Select
                  value={form.poll_status || '__none__'}
                  onValueChange={(value) => {
                    if (!value) return
                    setForm((prev) => ({
                      ...prev,
                      poll_status: value === '__none__' ? '' : value,
                    }))
                  }}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value='__none__'>{t('None')}</SelectItem>
                    <SelectItem value='strict'>{t('Poll status: strict')}</SelectItem>
                    <SelectItem value='relaxed'>{t('Poll status: relaxed')}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div className='space-y-1'>
              <div className='flex items-center gap-2'>
                <Checkbox
                  checked={form.requires_reference_media}
                  onCheckedChange={(checked) =>
                    setForm((prev) => ({
                      ...prev,
                      requires_reference_media: checked === true,
                    }))
                  }
                />
                <Label>{t('Requires reference media')}</Label>
              </div>
              <FieldHelp>{t('Requires reference media help')}</FieldHelp>
            </div>
            <div className='space-y-2'>
              <Label>{t('Reference limits (JSON)')}</Label>
              <FieldHelp>{t('Reference limits help')}</FieldHelp>
              <Textarea
                rows={3}
                className='font-mono text-xs'
                value={form.reference_limits}
                onChange={(event) =>
                  setForm((prev) => ({
                    ...prev,
                    reference_limits: event.target.value,
                  }))
                }
              />
            </div>
            <div className='space-y-2'>
              <Label>{t('Poll override (JSON)')}</Label>
              <FieldHelp>{t('Poll override help')}</FieldHelp>
              <Textarea
                rows={3}
                className='font-mono text-xs'
                value={form.poll}
                onChange={(event) =>
                  setForm((prev) => ({ ...prev, poll: event.target.value }))
                }
              />
            </div>
          </>
        ) : null}
        <div className='space-y-2'>
          <Label>{t('Parameter toggles')}</Label>
          <FieldHelp>{t('Parameter toggles help')}</FieldHelp>
          <div className='grid gap-2 sm:grid-cols-2'>
            {paramKeys.map((key) => (
              <div key={key} className='flex items-center gap-2'>
                <Checkbox
                  checked={readParamEnabled(form.params, key)}
                  onCheckedChange={(checked) =>
                    setForm((prev) => ({
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
          <FieldHelp>{t('Params JSON help')}</FieldHelp>
          <Textarea
            rows={10}
            className='font-mono text-xs'
            value={form.params}
            onChange={(event) =>
              setForm((prev) => ({ ...prev, params: event.target.value }))
            }
          />
        </div>
        {capability === 'video' ? (
          <>
            <div className='space-y-2'>
              <Label>{t('Option rules (JSON)')}</Label>
              <FieldHelp>{t('Option rules help')}</FieldHelp>
              <Textarea
                rows={4}
                className='font-mono text-xs'
                value={form.option_rules}
                onChange={(event) =>
                  setForm((prev) => ({
                    ...prev,
                    option_rules: event.target.value,
                  }))
                }
              />
            </div>
            <div className='space-y-2'>
              <Label>{t('Hints (JSON)')}</Label>
              <FieldHelp>{t('Hints help')}</FieldHelp>
              <Textarea
                rows={4}
                className='font-mono text-xs'
                value={form.hints}
                onChange={(event) =>
                  setForm((prev) => ({ ...prev, hints: event.target.value }))
                }
              />
            </div>
          </>
        ) : null}
        <div className='space-y-2'>
          <Label>{t('Admin note')}</Label>
          <FieldHelp>{t('Admin note help')}</FieldHelp>
          <Input
            value={form.note}
            onChange={(event) =>
              setForm((prev) => ({ ...prev, note: event.target.value }))
            }
          />
        </div>
      </div>
    </Dialog>
  )
}

function ModelBindingsPanel({
  videoProfiles,
  imageProfiles,
}: {
  videoProfiles: ModelUiParamProfile[]
  imageProfiles: ModelUiParamProfile[]
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [keyword, setKeyword] = useState('')
  const [drafts, setDrafts] = useState<
    Record<number, { video: string; image: string }>
  >({})

  const { data, isLoading } = useQuery({
    queryKey: modelParamsQueryKeys.modelBindings(keyword),
    queryFn: () =>
      searchModels({
        keyword: keyword.trim() || undefined,
        p: 1,
        page_size: 200,
      }),
  })

  const models = data?.data?.items ?? []

  useEffect(() => {
    const next: Record<number, { video: string; image: string }> = {}
    for (const model of models) {
      next[model.id] = {
        video: model.video_profile_id || '',
        image: model.image_profile_id || '',
      }
    }
    setDrafts(next)
  }, [models])

  const saveMutation = useMutation({
    mutationFn: (model: Model) =>
      updateModel({
        ...model,
        video_profile_id: drafts[model.id]?.video || '',
        image_profile_id: drafts[model.id]?.image || '',
      }),
    onSuccess: async (res) => {
      if (!res.success) {
        toast.error(res.message || t('Save failed'))
        return
      }
      toast.success(t('Saved'))
      await queryClient.invalidateQueries({ queryKey: modelsQueryKeys.all })
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const profileOptions = (capability: ModelUiParamCapability) => {
    const profiles =
      capability === 'video' ? videoProfiles : imageProfiles
    const defaultId =
      capability === 'video'
        ? DEFAULT_VIDEO_PROFILE_ID
        : DEFAULT_IMAGE_PROFILE_ID
    return [
      { value: '', label: t('Default ({{id}})', { id: defaultId }) },
      ...profiles.map((profile) => ({
        value: profile.profile_id,
        label: profile.profile_id,
      })),
    ]
  }

  return (
    <Card>
      <CardHeader className='pb-3'>
        <CardTitle className='text-base'>{t('Model profile bindings')}</CardTitle>
      </CardHeader>
      <CardContent className='space-y-3'>
        <FieldHelp>{t('Model profile bindings help')}</FieldHelp>
        <Input
          placeholder={t('Search models')}
          value={keyword}
          onChange={(event) => setKeyword(event.target.value)}
        />
        <div className='max-h-[min(60vh,720px)] overflow-auto rounded-md border'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Model name')}</TableHead>
                <TableHead>{t('Video profile')}</TableHead>
                <TableHead>{t('Image profile')}</TableHead>
                <TableHead className='text-right'>{t('Actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={4}>{t('Loading...')}</TableCell>
                </TableRow>
              ) : models.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={4} className='text-muted-foreground'>
                    {t('No models found')}
                  </TableCell>
                </TableRow>
              ) : (
                models.map((model) => (
                  <TableRow key={model.id}>
                    <TableCell className='font-mono text-xs'>
                      {model.model_name}
                    </TableCell>
                    <TableCell>
                      <Select
                        value={drafts[model.id]?.video || '__default__'}
                        onValueChange={(value) => {
                          if (!value) return
                          setDrafts((prev) => ({
                            ...prev,
                            [model.id]: {
                              video: value === '__default__' ? '' : value,
                              image: prev[model.id]?.image || '',
                            },
                          }))
                        }}
                      >
                        <SelectTrigger className='h-8'>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {profileOptions('video').map((option) => (
                            <SelectItem
                              key={option.value || '__default__'}
                              value={option.value || '__default__'}
                            >
                              {option.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </TableCell>
                    <TableCell>
                      <Select
                        value={drafts[model.id]?.image || '__default__'}
                        onValueChange={(value) => {
                          if (!value) return
                          setDrafts((prev) => ({
                            ...prev,
                            [model.id]: {
                              video: prev[model.id]?.video || '',
                              image: value === '__default__' ? '' : value,
                            },
                          }))
                        }}
                      >
                        <SelectTrigger className='h-8'>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {profileOptions('image').map((option) => (
                            <SelectItem
                              key={option.value || '__default__'}
                              value={option.value || '__default__'}
                            >
                              {option.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </TableCell>
                    <TableCell className='text-right'>
                      <Button
                        size='sm'
                        variant='outline'
                        disabled={saveMutation.isPending}
                        onClick={() => saveMutation.mutate(model)}
                      >
                        {t('Save')}
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  )
}

export function ModelParamsSection() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [capability, setCapability] = useState<ModelUiParamCapability>('video')
  const [profileDialogOpen, setProfileDialogOpen] = useState(false)
  const [editingProfile, setEditingProfile] = useState<ModelUiParamProfile | null>(
    null
  )
  const [deleteProfileId, setDeleteProfileId] = useState<number | null>(null)

  const invalidateProfiles = async () => {
    await Promise.all([
      queryClient.invalidateQueries({
        queryKey: modelParamsQueryKeys.registry(capability),
      }),
      queryClient.invalidateQueries({
        queryKey: modelParamsQueryKeys.profiles('video'),
      }),
      queryClient.invalidateQueries({
        queryKey: modelParamsQueryKeys.profiles('image'),
      }),
      queryClient.invalidateQueries({ queryKey: modelParamsQueryKeys.all }),
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

  const { data: videoProfiles = [] } = useQuery({
    queryKey: modelParamsQueryKeys.profiles('video'),
    queryFn: () => listModelUiParamProfiles('video'),
  })

  const { data: imageProfiles = [] } = useQuery({
    queryKey: modelParamsQueryKeys.profiles('image'),
    queryFn: () => listModelUiParamProfiles('image'),
  })

  const registryForm = useMemo(
    () => ({
      default_profile_id: registry?.default_profile_id ?? '',
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
        poll_defaults:
          capability === 'video' ? registryDraft.poll_defaults : undefined,
      }),
    onSuccess: async (res) => {
      if (!res.success) {
        toast.error(res.message || t('Save failed'))
        return
      }
      toast.success(t('Saved'))
      await invalidateProfiles()
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const deleteProfileMutation = useMutation({
    mutationFn: (id: number) => deleteModelUiParamProfile(id),
    onSuccess: async (res) => {
      if (!res.success) {
        toast.error(res.message || t('Delete failed'))
        return
      }
      toast.success(t('Deleted'))
      setDeleteProfileId(null)
      await invalidateProfiles()
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const paramKeys =
    capability === 'video' ? VIDEO_PARAM_KEYS : IMAGE_PARAM_KEYS

  return (
    <div className='flex h-full min-h-0 flex-col gap-4'>
      <Tabs defaultValue='profiles' className='flex min-h-0 flex-1 flex-col'>
        <TabsList className='shrink-0'>
          <TabsTrigger value='profiles'>{t('Parameter profiles')}</TabsTrigger>
          <TabsTrigger value='bindings'>{t('Model bindings')}</TabsTrigger>
        </TabsList>

        <TabsContent
          value='profiles'
          className='mt-4 flex min-h-0 flex-1 flex-col gap-4 overflow-hidden'
        >
          <Tabs
            value={capability}
            onValueChange={(value) =>
              setCapability(value as ModelUiParamCapability)
            }
            className='shrink-0'
          >
            <TabsList>
              <TabsTrigger value='video'>{t('Video parameters')}</TabsTrigger>
              <TabsTrigger value='image'>{t('Image parameters')}</TabsTrigger>
            </TabsList>
          </Tabs>

          <Card className='shrink-0'>
            <CardHeader className='pb-3'>
              <CardTitle className='text-base'>{t('Registry settings')}</CardTitle>
              <FieldHelp>{t('Registry settings help')}</FieldHelp>
            </CardHeader>
            <CardContent className='grid gap-3 md:grid-cols-2'>
              <div className='space-y-2'>
                <Label>{t('Default profile id')}</Label>
                <FieldHelp>{t('Default profile id help')}</FieldHelp>
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
              {capability === 'video' ? (
                <div className='space-y-2 md:col-span-2'>
                  <Label>{t('Poll defaults (JSON)')}</Label>
                  <FieldHelp>{t('Poll defaults help')}</FieldHelp>
                  <Textarea
                    rows={4}
                    className='font-mono text-xs'
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

          <div className='flex shrink-0 items-center justify-between'>
            <div>
              <h3 className='text-sm font-medium'>{t('Profile templates')}</h3>
              <FieldHelp>{t('Profile templates help')}</FieldHelp>
            </div>
            <Button
              size='sm'
              onClick={() => {
                setEditingProfile(null)
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
                  {capability === 'video' ? (
                    <TableHead>{t('API mode')}</TableHead>
                  ) : null}
                  <TableHead>{t('Params')}</TableHead>
                  <TableHead className='text-right'>{t('Actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {profilesLoading ? (
                  <TableRow>
                    <TableCell colSpan={4}>{t('Loading...')}</TableCell>
                  </TableRow>
                ) : profiles.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={4} className='text-muted-foreground'>
                      {t('No profiles yet')}
                    </TableCell>
                  </TableRow>
                ) : (
                  profiles.map((profile) => (
                    <TableRow key={profile.id}>
                      <TableCell className='font-mono text-xs'>
                        {profile.profile_id}
                      </TableCell>
                      {capability === 'video' ? (
                        <TableCell className='text-xs'>
                          {profile.api_mode
                            ? getVideoApiModeLabel(t, profile.api_mode)
                            : '—'}
                        </TableCell>
                      ) : null}
                      <TableCell className='text-xs text-muted-foreground'>
                        {paramKeys
                          .filter((key) =>
                            readParamEnabled(profile.params, key)
                          )
                          .join(', ') || '—'}
                      </TableCell>
                      <TableCell className='text-right'>
                        <div className='flex justify-end gap-1'>
                          <Button
                            variant='ghost'
                            size='icon'
                            onClick={() => {
                              setEditingProfile(profile)
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
        </TabsContent>

        <TabsContent
          value='bindings'
          className='mt-4 min-h-0 flex-1 overflow-y-auto'
        >
          <ModelBindingsPanel
            videoProfiles={videoProfiles}
            imageProfiles={imageProfiles}
          />
        </TabsContent>
      </Tabs>

      <ProfileEditorDialog
        open={profileDialogOpen}
        onOpenChange={setProfileDialogOpen}
        capability={capability}
        editingProfile={editingProfile}
        onSaved={invalidateProfiles}
      />

      <ConfirmDialog
        open={deleteProfileId !== null}
        onOpenChange={(open) => !open && setDeleteProfileId(null)}
        title={t('Delete profile')}
        desc={t('This action cannot be undone.')}
        confirmText={t('Delete')}
        destructive
        handleConfirm={() => {
          if (deleteProfileId) deleteProfileMutation.mutate(deleteProfileId)
        }}
      />
    </div>
  )
}
