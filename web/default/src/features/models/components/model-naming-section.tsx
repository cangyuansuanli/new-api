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
import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  AlertTriangle,
  ArrowRight,
  Pencil,
  Plus,
  Route,
  Store,
  Trash2,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { Dialog } from '@/components/dialog'
import {
  createModelPublicAlias,
  deleteModelPublicAlias,
  getModelPublicNameRegistryStatus,
  listModelPublicAliases,
  updateModelPublicAlias,
} from '../model-naming-api'
import {
  modelNamingQueryKeys,
  type ModelPublicAlias,
} from '../model-naming-types'
import { RoutingAliasesCard } from './routing-aliases-card'

type AliasFormState = {
  internal_name: string
  public_name: string
  show_in_marketplace: boolean
}

const emptyAliasForm = (): AliasFormState => ({
  internal_name: '',
  public_name: '',
  show_in_marketplace: true,
})

function NamingModelGuide() {
  const { t } = useTranslation()

  return (
    <section className='bg-muted/30 border-y px-4 py-4 sm:px-6'>
      <div className='mb-3'>
        <h2 className='text-base font-semibold'>{t('How model names work')}</h2>
        <p className='text-muted-foreground mt-1 text-sm'>
          {t(
            'Customers use stable API routes by default, or a channel-specific name when they need to select a channel.'
          )}
        </p>
      </div>
      <div className='grid gap-3 lg:grid-cols-2'>
        <div className='bg-background flex min-w-0 items-center gap-2 border-l-2 border-sky-500 px-3 py-3'>
          <Route className='size-4 shrink-0 text-sky-600' />
          <div className='min-w-0'>
            <p className='text-xs font-medium text-sky-700 dark:text-sky-400'>
              {t('Default: stable API route')}
            </p>
            <div className='mt-1 flex flex-wrap items-center gap-2 font-mono text-xs sm:text-sm'>
              <span>{t('Inbound model name')}</span>
              <ArrowRight className='text-muted-foreground size-3.5' />
              <span>{t('Internal ability')}</span>
            </div>
          </div>
        </div>
        <div className='bg-background flex min-w-0 items-center gap-2 border-l-2 border-emerald-500 px-3 py-3'>
          <Store className='size-4 shrink-0 text-emerald-600' />
          <div className='min-w-0'>
            <p className='text-xs font-medium text-emerald-700 dark:text-emerald-400'>
              {t('Optional: select a channel')}
            </p>
            <div className='mt-1 flex flex-wrap items-center gap-2 font-mono text-xs sm:text-sm'>
              <span>{t('Channel-specific name')}</span>
              <ArrowRight className='text-muted-foreground size-3.5' />
              <span>{t('Internal ability')}</span>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}

export function ModelNamingSection() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingAlias, setEditingAlias] = useState<ModelPublicAlias | null>(
    null
  )
  const [form, setForm] = useState<AliasFormState>(emptyAliasForm)
  const [deleteId, setDeleteId] = useState<number | null>(null)

  const { data: aliases = [], isLoading } = useQuery({
    queryKey: modelNamingQueryKeys.aliases(),
    queryFn: listModelPublicAliases,
  })
  const { data: registryStatus } = useQuery({
    queryKey: modelNamingQueryKeys.status(),
    queryFn: getModelPublicNameRegistryStatus,
  })

  const collisionEntries = useMemo(
    () => Object.entries(registryStatus?.collisions ?? {}),
    [registryStatus?.collisions]
  )
  const missingAliases = registryStatus?.missing_aliases ?? []

  const invalidateRegistry = async () => {
    await Promise.all([
      queryClient.invalidateQueries({
        queryKey: modelNamingQueryKeys.aliases(),
      }),
      queryClient.invalidateQueries({
        queryKey: modelNamingQueryKeys.status(),
      }),
    ])
  }

  const saveMutation = useMutation({
    mutationFn: async () => {
      const payload = {
        internal_name: form.internal_name.trim(),
        public_name: form.public_name.trim(),
        hidden_from_marketplace: !form.show_in_marketplace,
      }
      if (editingAlias) {
        return updateModelPublicAlias({ id: editingAlias.id, ...payload })
      }
      return createModelPublicAlias(payload)
    },
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t('Operation failed'))
        return
      }
      toast.success(
        editingAlias ? t('Channel name updated') : t('Channel name created')
      )
      setDialogOpen(false)
      setEditingAlias(null)
      setForm(emptyAliasForm())
      await invalidateRegistry()
    },
    onError: (error: Error) => {
      toast.error(error.message || t('Operation failed'))
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteModelPublicAlias(id),
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t('Operation failed'))
        return
      }
      toast.success(t('Channel name deleted'))
      setDeleteId(null)
      await invalidateRegistry()
    },
    onError: (error: Error) => {
      toast.error(error.message || t('Operation failed'))
    },
  })

  const visibilityMutation = useMutation({
    mutationFn: (alias: ModelPublicAlias) =>
      updateModelPublicAlias({
        id: alias.id,
        internal_name: alias.internal_name,
        public_name: alias.public_name,
        hidden_from_marketplace: !alias.hidden_from_marketplace,
      }),
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t('Operation failed'))
        return
      }
      await invalidateRegistry()
    },
    onError: (error: Error) => {
      toast.error(error.message || t('Operation failed'))
    },
  })

  const openCreate = () => {
    setEditingAlias(null)
    setForm(emptyAliasForm())
    setDialogOpen(true)
  }

  const openEdit = (alias: ModelPublicAlias) => {
    setEditingAlias(alias)
    setForm({
      internal_name: alias.internal_name,
      public_name: alias.public_name,
      show_in_marketplace: !alias.hidden_from_marketplace,
    })
    setDialogOpen(true)
  }

  return (
    <div className='flex flex-col gap-4 pb-4'>
      <NamingModelGuide />

      {missingAliases.length > 0 ? (
        <Alert variant='destructive'>
          <AlertTriangle className='h-4 w-4' />
          <AlertTitle>{t('Explicit marketplace names required')}</AlertTitle>
          <AlertDescription>
            <p className='mb-2'>
              {t(
                'These internal abilities are hidden from public model lists until a marketplace name is configured.'
              )}
            </p>
            <ul className='list-disc space-y-1 pl-5 font-mono text-sm'>
              {missingAliases.map((internal) => (
                <li key={internal}>{internal}</li>
              ))}
            </ul>
          </AlertDescription>
        </Alert>
      ) : null}

      {collisionEntries.length > 0 ? (
        <Alert variant='destructive'>
          <AlertTriangle className='h-4 w-4' />
          <AlertTitle>{t('Marketplace name collisions detected')}</AlertTitle>
          <AlertDescription>
            <ul className='list-disc space-y-1 pl-5 text-sm'>
              {collisionEntries.map(([publicName, internals]) => (
                <li key={publicName}>
                  <span className='font-medium'>{publicName}</span>:{' '}
                  {internals.join(', ')}
                </li>
              ))}
            </ul>
          </AlertDescription>
        </Alert>
      ) : null}

      {missingAliases.length === 0 && collisionEntries.length === 0 ? (
        <Alert>
          <AlertTitle>
            {t('Public naming configuration is complete')}
          </AlertTitle>
          <AlertDescription>
            {t('No implicit prefix stripping is used.')}
          </AlertDescription>
        </Alert>
      ) : null}

      <RoutingAliasesCard />

      <Card>
        <CardHeader className='flex flex-col items-start justify-between gap-3 space-y-0 sm:flex-row sm:items-center'>
          <div>
            <CardTitle>{t('Channel-specific names')}</CardTitle>
            <p className='text-muted-foreground mt-1 text-sm'>
              {t(
                'Direct API names for selecting a specific channel. Visibility only controls the marketplace and model list.'
              )}
            </p>
          </div>
          <Button size='sm' onClick={openCreate}>
            <Plus className='h-4 w-4' />
            {t('Add channel name')}
          </Button>
        </CardHeader>
        <CardContent className='overflow-x-auto'>
          <Table className='min-w-[640px]'>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Internal ability')}</TableHead>
                <TableHead>{t('Channel-specific name')}</TableHead>
                <TableHead>{t('Visible')}</TableHead>
                <TableHead className='w-[100px]'>{t('Actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={4} className='text-muted-foreground'>
                    {t('Loading...')}
                  </TableCell>
                </TableRow>
              ) : aliases.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={4} className='text-muted-foreground'>
                    {t('No channel-specific names configured')}
                  </TableCell>
                </TableRow>
              ) : (
                aliases.map((alias) => (
                  <TableRow key={alias.id}>
                    <TableCell className='font-mono text-sm'>
                      {alias.internal_name}
                    </TableCell>
                    <TableCell className='font-mono text-sm'>
                      {alias.public_name}
                    </TableCell>
                    <TableCell>
                      <Switch
                        checked={!alias.hidden_from_marketplace}
                        disabled={visibilityMutation.isPending}
                        aria-label={t('Show in marketplace')}
                        onCheckedChange={() => visibilityMutation.mutate(alias)}
                      />
                    </TableCell>
                    <TableCell>
                      <div className='flex gap-1'>
                        <Button
                          variant='ghost'
                          size='icon'
                          aria-label={t('Edit channel name')}
                          title={t('Edit channel name')}
                          onClick={() => openEdit(alias)}
                        >
                          <Pencil className='h-4 w-4' />
                        </Button>
                        <Button
                          variant='ghost'
                          size='icon'
                          aria-label={t('Delete channel name')}
                          title={t('Delete channel name')}
                          onClick={() => setDeleteId(alias.id)}
                        >
                          <Trash2 className='text-destructive h-4 w-4' />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Dialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        title={editingAlias ? t('Edit channel name') : t('Add channel name')}
        footer={
          <>
            <Button variant='outline' onClick={() => setDialogOpen(false)}>
              {t('Cancel')}
            </Button>
            <Button
              onClick={() => saveMutation.mutate()}
              disabled={
                saveMutation.isPending ||
                !form.internal_name.trim() ||
                !form.public_name.trim()
              }
            >
              {t('Save')}
            </Button>
          </>
        }
      >
        <div className='space-y-4'>
          <div className='space-y-2'>
            <Label htmlFor='internal-name'>{t('Internal ability')}</Label>
            <Input
              id='internal-name'
              placeholder='cy-sd4-seedance-2.5-480p'
              value={form.internal_name}
              onChange={(event) =>
                setForm((previous) => ({
                  ...previous,
                  internal_name: event.target.value,
                }))
              }
            />
          </div>
          <div className='flex items-center justify-between gap-4'>
            <Label htmlFor='show-in-marketplace'>
              {t('Show in marketplace')}
            </Label>
            <Switch
              id='show-in-marketplace'
              checked={form.show_in_marketplace}
              onCheckedChange={(checked) =>
                setForm((previous) => ({
                  ...previous,
                  show_in_marketplace: checked,
                }))
              }
            />
          </div>
          <div className='space-y-2'>
            <Label htmlFor='public-name'>{t('Channel-specific name')}</Label>
            <Input
              id='public-name'
              placeholder='sd4-seedance-2.5-480p'
              value={form.public_name}
              onChange={(event) =>
                setForm((previous) => ({
                  ...previous,
                  public_name: event.target.value,
                }))
              }
            />
          </div>
        </div>
      </Dialog>

      <ConfirmDialog
        open={deleteId !== null}
        onOpenChange={(open) => !open && setDeleteId(null)}
        title={t('Delete channel name')}
        desc={t(
          'Internal models that require an explicit marketplace name will be hidden after deletion.'
        )}
        confirmText={t('Delete')}
        destructive
        handleConfirm={() => {
          if (deleteId !== null) {
            deleteMutation.mutate(deleteId)
          }
        }}
      />
    </div>
  )
}
