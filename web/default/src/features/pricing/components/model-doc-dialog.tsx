/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { CopyButton } from '@/components/copy-button'
import { buildModelApiDoc, type ModelApiDocVariant, type ModelDocGenerationMode } from '../lib/model-api-doc'
import type { PricingModel } from '../types'

type ModelDocDialogProps = {
  model: PricingModel | null
  siteOrigin: string
  open: boolean
  onOpenChange: (open: boolean) => void
}

function modeLabel(t: (key: string) => string, mode: ModelApiDocVariant['mode']) {
  return mode === 'async' ? t('modelDoc.modeAsync') : t('modelDoc.modeSync')
}

function JsonBlock(props: { label: string; value: string; copyLabel: string }) {
  return (
    <div className='space-y-2'>
      <div className='flex items-center justify-between gap-2'>
        <h4 className='text-sm font-semibold'>{props.label}</h4>
        <CopyButton value={props.value} tooltip={props.copyLabel} />
      </div>
      <pre className='bg-muted/40 overflow-x-auto rounded-lg border p-3 font-mono text-xs leading-relaxed'>
        {props.value}
      </pre>
    </div>
  )
}

function GenerationModesTable(props: {
  modes: ModelDocGenerationMode[]
  t: (key: string) => string
}) {
  if (!props.modes.length) return null

  return (
    <div className='space-y-2'>
      <div>
        <h4 className='text-sm font-semibold'>{props.t('modelDoc.generationModesTitle')}</h4>
        <p className='text-muted-foreground text-xs'>{props.t('modelDoc.generationModesHint')}</p>
      </div>
      <div className='overflow-x-auto rounded-lg border'>
        <table className='w-full min-w-[520px] text-left text-sm'>
          <thead className='bg-muted/40 text-muted-foreground text-xs'>
            <tr>
              <th className='px-3 py-2 font-medium'>{props.t('modelDoc.generationModeLabel')}</th>
              <th className='px-3 py-2 font-medium'>{props.t('modelDoc.generationModeMinimum')}</th>
              <th className='px-3 py-2 font-medium'>{props.t('modelDoc.generationModeTrigger')}</th>
              <th className='px-3 py-2 font-medium'>{props.t('modelDoc.generationModePromptRefs')}</th>
            </tr>
          </thead>
          <tbody>
            {props.modes.map((mode) => (
              <tr key={mode.label} className='border-t align-top'>
                <td className='px-3 py-2.5 font-medium'>{mode.label}</td>
                <td className='px-3 py-2.5 font-mono text-xs'>{mode.minimum}</td>
                <td className='px-3 py-2.5 text-muted-foreground text-xs leading-relaxed'>
                  {mode.trigger}
                  {mode.notes ? (
                    <p className='text-foreground/70 mt-1 text-[11px] leading-relaxed'>{mode.notes}</p>
                  ) : null}
                </td>
                <td className='px-3 py-2.5 font-mono text-xs'>{mode.promptRefs ?? '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function VariantBody(props: {
  variant: ModelApiDocVariant
  t: (key: string) => string
}) {
  const { variant, t } = props
  const hasExamples = variant.examples.length > 0

  return (
    <div className='space-y-5'>
      <div className='space-y-2'>
        <p className='text-muted-foreground whitespace-pre-line text-sm leading-relaxed'>
          {variant.intro}
        </p>
      </div>

      <GenerationModesTable modes={variant.generationModes} t={t} />

      <div className='space-y-3'>
        {variant.endpoints.map((endpoint) => (
          <div
            key={`${endpoint.method}-${endpoint.path}`}
            className='rounded-lg border bg-muted/20 px-3 py-2.5 text-sm leading-relaxed'
          >
            <div className='mb-1 flex flex-wrap items-center gap-2'>
              <Badge variant='outline' className='font-mono text-[11px]'>
                {endpoint.method}
              </Badge>
              <code className='text-foreground/90 text-xs break-all'>
                {endpoint.path}
              </code>
            </div>
            <p className='text-muted-foreground text-xs'>
              {endpoint.description}
            </p>
          </div>
        ))}
      </div>

      <div className='space-y-2'>
        <h4 className='text-sm font-semibold'>{t('Request fields')}</h4>
        <ul className='space-y-1.5 text-sm'>
          {variant.params.map((param) => (
            <li key={param.name}>
              <strong className='font-mono text-xs'>{param.name}</strong>
              <span className='text-muted-foreground'>
                ：{param.description}
              </span>
            </li>
          ))}
        </ul>
      </div>

      {hasExamples ? (
        <div className='space-y-4'>
          <h4 className='text-sm font-semibold'>{t('modelDoc.requestExamples')}</h4>
          {variant.examples.map((example) => (
            <JsonBlock
              key={example.title}
              label={example.title}
              value={example.requestJson}
              copyLabel={t('Copy')}
            />
          ))}
        </div>
      ) : (
        <JsonBlock
          label={t('modelDoc.requestJson')}
          value={variant.requestJson}
          copyLabel={t('Copy')}
        />
      )}

      <div className='space-y-3'>
        <h4 className='text-sm font-semibold'>{t('Response fields')}</h4>
        <div className='space-y-2'>
          <p className='text-muted-foreground text-xs font-medium uppercase tracking-wide'>
            {variant.mode === 'async'
              ? t('Create task response')
              : t('modelDoc.syncResponse')}
          </p>
          <pre className='bg-muted/40 overflow-x-auto rounded-lg border p-3 font-mono text-xs leading-relaxed'>
            {variant.createResponseJson}
          </pre>
        </div>
        {variant.queryResponseJson ? (
          <JsonBlock
            label={t('Query task response')}
            value={variant.queryResponseJson}
            copyLabel={t('Copy')}
          />
        ) : null}
        {variant.queryFailedResponseJson ? (
          <JsonBlock
            label={t('modelDoc.queryFailedResponse')}
            value={variant.queryFailedResponseJson}
            copyLabel={t('Copy')}
          />
        ) : null}
      </div>
    </div>
  )
}

export function ModelDocDialog(props: ModelDocDialogProps) {
  const { t } = useTranslation()
  const doc = useMemo(() => {
    if (!props.model) return null
    return buildModelApiDoc(props.model, props.siteOrigin)
  }, [props.model, props.siteOrigin])

  const [activeMode, setActiveMode] = useState<string>('async')

  const title = doc
    ? `${doc.displayName} · ${t('API call guide')}`
    : t('View document')

  const defaultTab = doc?.variants[0]?.mode ?? 'async'

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='gap-0 p-0 sm:max-w-2xl'>
        <DialogHeader className='border-b px-5 py-4'>
          <DialogTitle className='text-base font-semibold'>{title}</DialogTitle>
        </DialogHeader>

        {doc ? (
          <div className='hover-scrollbar max-h-[min(72vh,640px)] overflow-y-auto px-5 py-4'>
            <div className='mb-4 space-y-2'>
              <Badge variant='secondary' className='text-xs'>
                {t('API documentation')}
              </Badge>
              {doc.variants.length === 1 ? (
                <p className='text-sm font-medium'>
                  {modeLabel(t, doc.variants[0].mode)}
                </p>
              ) : null}
            </div>

            {doc.variants.length > 1 ? (
              <Tabs
                value={activeMode || defaultTab}
                onValueChange={setActiveMode}
                className='gap-4'
              >
                <TabsList className='w-full'>
                  {doc.variants.map((variant) => (
                    <TabsTrigger
                      key={variant.mode}
                      value={variant.mode}
                      className='flex-1'
                    >
                      {modeLabel(t, variant.mode)}
                    </TabsTrigger>
                  ))}
                </TabsList>
                {doc.variants.map((variant) => (
                  <TabsContent key={variant.mode} value={variant.mode}>
                    <VariantBody variant={variant} t={t} />
                  </TabsContent>
                ))}
              </Tabs>
            ) : (
              <VariantBody variant={doc.variants[0]} t={t} />
            )}

            <p className='text-muted-foreground mt-5 text-xs'>
              {t('modelDoc.authHint')}
            </p>
          </div>
        ) : (
          <p className='text-muted-foreground px-5 py-6 text-sm'>
            {t('No description available.')}
          </p>
        )}
      </DialogContent>
    </Dialog>
  )
}
