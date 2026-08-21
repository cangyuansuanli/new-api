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
import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { Pencil, Plus, SlidersHorizontal, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
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
  createModelRoutingAlias,
  deleteModelRoutingAlias,
  listModelRoutingAliases,
  updateModelRoutingAlias,
} from '../model-naming-api'
import {
  modelNamingQueryKeys,
  type ModelRoutingAlias,
} from '../model-naming-types'

type RoutingAliasFormState = {
  public_name: string
  internal_name: string
  note: string
}

const emptyForm = (): RoutingAliasFormState => ({
  public_name: '',
  internal_name: '',
  note: '',
})

export function RoutingAliasesCard() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingAlias, setEditingAlias] = useState<ModelRoutingAlias | null>(
    null
  )
  const [form, setForm] = useState<RoutingAliasFormState>(emptyForm)
  const [deleteId, setDeleteId] = useState<number | null>(null)

  const { data: aliases = [], isLoading } = useQuery({
    queryKey: modelNamingQueryKeys.routingAliases(),
    queryFn: listModelRoutingAliases,
  })

  const invalidateRegistry = async () => {
    await Promise.all([
      queryClient.invalidateQueries({
        queryKey: modelNamingQueryKeys.routingAliases(),
      }),
      queryClient.invalidateQueries({
        queryKey: modelNamingQueryKeys.status(),
      }),
    ])
  }

  const saveMutation = useMutation({
    mutationFn: async () => {
      const payload = {
        public_name: form.public_name.trim(),
        internal_name: form.internal_name.trim(),
        note: form.note.trim(),
      }
      if (editingAlias) {
        return updateModelRoutingAlias({ id: editingAlias.id, ...payload })
      }
      return createModelRoutingAlias(payload)
    },
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t('Operation failed'))
        return
      }
      toast.success(
        editingAlias ? t('Inbound route updated') : t('Inbound route created')
      )
      setDialogOpen(false)
      setEditingAlias(null)
      setForm(emptyForm())
      await invalidateRegistry()
    },
    onError: (error: Error) => {
      toast.error(error.message || t('Operation failed'))
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteModelRoutingAlias(id),
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t('Operation failed'))
        return
      }
      toast.success(t('Inbound route deleted'))
      setDeleteId(null)
      await invalidateRegistry()
    },
    onError: (error: Error) => {
      toast.error(error.message || t('Operation failed'))
    },
  })

  const openCreate = () => {
    setEditingAlias(null)
    setForm(emptyForm())
    setDialogOpen(true)
  }

  const openEdit = (alias: ModelRoutingAlias) => {
    setEditingAlias(alias)
    setForm({
      public_name: alias.public_name,
      internal_name: alias.internal_name,
      note: alias.note ?? '',
    })
    setDialogOpen(true)
  }

  return (
    <Card>
      <CardHeader className='flex flex-col items-start justify-between gap-3 space-y-0 sm:flex-row'>
        <div className='min-w-0'>
          <CardTitle>{t('API inbound routes')}</CardTitle>
          <p className='text-muted-foreground mt-1 text-sm'>
            {t(
              'Fallback routes used when no public mapping switch is on. Active public mappings take priority at runtime.'
            )}
          </p>
        </div>
        <div className='flex shrink-0 flex-wrap justify-end gap-2'>
          <Button variant='outline' size='sm' render={<Link to='/channels' />}>
            <SlidersHorizontal className='h-4 w-4' />
            {t('Channel priority')}
          </Button>
          <Button size='sm' onClick={openCreate}>
            <Plus className='h-4 w-4' />
            {t('Add inbound route')}
          </Button>
        </div>
      </CardHeader>
      <CardContent className='overflow-x-auto'>
        <Table className='min-w-[760px]'>
          <TableHeader>
            <TableRow>
              <TableHead>{t('Inbound model name')}</TableHead>
              <TableHead>{t('Internal ability')}</TableHead>
              <TableHead>{t('Note')}</TableHead>
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
                  {t('No routing aliases configured')}
                </TableCell>
              </TableRow>
            ) : (
              aliases.map((alias) => (
                <TableRow key={alias.id}>
                  <TableCell className='font-mono text-sm'>
                    {alias.public_name}
                  </TableCell>
                  <TableCell className='font-mono text-sm'>
                    {alias.internal_name}
                  </TableCell>
                  <TableCell>{alias.note || '—'}</TableCell>
                  <TableCell>
                    <div className='flex gap-1'>
                      <Button
                        variant='ghost'
                        size='icon'
                        aria-label={t('Edit routing alias')}
                        title={t('Edit routing alias')}
                        onClick={() => openEdit(alias)}
                      >
                        <Pencil className='h-4 w-4' />
                      </Button>
                      <Button
                        variant='ghost'
                        size='icon'
                        aria-label={t('Delete routing alias')}
                        title={t('Delete routing alias')}
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

      <Dialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        title={editingAlias ? t('Edit inbound route') : t('Add inbound route')}
        footer={
          <>
            <Button variant='outline' onClick={() => setDialogOpen(false)}>
              {t('Cancel')}
            </Button>
            <Button
              onClick={() => saveMutation.mutate()}
              disabled={
                saveMutation.isPending ||
                !form.public_name.trim() ||
                !form.internal_name.trim()
              }
            >
              {t('Save')}
            </Button>
          </>
        }
      >
        <div className='space-y-4'>
          <div className='space-y-2'>
            <Label htmlFor='routing-public-name'>
              {t('Inbound model name')}
            </Label>
            <Input
              id='routing-public-name'
              placeholder='seedance-2.0-fast'
              value={form.public_name}
              onChange={(event) =>
                setForm((previous) => ({
                  ...previous,
                  public_name: event.target.value,
                }))
              }
            />
          </div>
          <div className='space-y-2'>
            <Label htmlFor='routing-internal-name'>
              {t('Internal ability')}
            </Label>
            <Input
              id='routing-internal-name'
              placeholder='cy-sd4-seedance-2.0-fast'
              value={form.internal_name}
              onChange={(event) =>
                setForm((previous) => ({
                  ...previous,
                  internal_name: event.target.value,
                }))
              }
            />
          </div>
          <div className='space-y-2'>
            <Label htmlFor='routing-note'>{t('Note')}</Label>
            <Input
              id='routing-note'
              value={form.note}
              onChange={(event) =>
                setForm((previous) => ({
                  ...previous,
                  note: event.target.value,
                }))
              }
            />
          </div>
        </div>
      </Dialog>

      <ConfirmDialog
        open={deleteId !== null}
        onOpenChange={(open) => !open && setDeleteId(null)}
        title={t('Delete inbound route')}
        desc={t(
          'Clients using this inbound model name will no longer be routed by this alias.'
        )}
        confirmText={t('Delete')}
        destructive
        handleConfirm={() => {
          if (deleteId !== null) {
            deleteMutation.mutate(deleteId)
          }
        }}
      />
    </Card>
  )
}
