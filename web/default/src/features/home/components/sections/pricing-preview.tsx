/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { ArrowRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useMemo, type ReactNode } from 'react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { AnimateInView } from '@/components/animate-in-view'
import { usePricingData } from '@/features/pricing/hooks/use-pricing-data'
import type { PricingModel } from '@/features/pricing/types'
import { mkt } from '../../lib/marketing-theme'
import { fetchModelsDevCostIndex, type ModelsDevCost } from '../../lib/models-dev-official'
import {
  buildHomePricingSections,
  formatHomeInputPrice,
  formatHomeMediaPrice,
  formatHomeModelDisplayName,
  formatHomeOfficialPricing,
  formatHomeOutputPrice,
  formatHomeSavePercent,
} from '../../lib/pricing-preview'

const HOME_PRICING_MAX_PER_PREFIX = 2

function PricingTableShell(props: {
  children: ReactNode
  className?: string
}) {
  return (
    <div className={cn('overflow-hidden rounded-2xl', mkt.card, props.className)}>
      <div className='overflow-x-auto'>{props.children}</div>
    </div>
  )
}

function SkeletonRows(props: { rows: number; cols: number }) {
  return (
    <>
      {Array.from({ length: props.rows }).map((_, i) => (
        <tr key={i} className={cn('border-b last:border-0', mkt.sectionBorder)}>
          {Array.from({ length: props.cols }).map((__, j) => (
            <td key={j} className='px-4 py-3.5'>
              <div className='h-4 animate-pulse rounded bg-slate-200/80 dark:bg-white/10' />
            </td>
          ))}
        </tr>
      ))}
    </>
  )
}

function TextPricingTable(props: {
  rows: PricingModel[]
  isLoading: boolean
  officialIndex: Record<string, ModelsDevCost>
}) {
  const { t } = useTranslation()

  return (
    <PricingTableShell>
      <table className='w-full min-w-[640px] text-sm'>
        <thead>
          <tr className={cn('border-b text-left text-xs', mkt.sectionBorder)}>
            <th className={cn('px-5 py-3.5 font-medium', mkt.muted)}>
              {t('Model')}
            </th>
            <th className={cn('px-4 py-3.5 font-medium', mkt.muted)}>
              {t('Input')}
            </th>
            <th className={cn('px-4 py-3.5 font-medium', mkt.muted)}>
              {t('Output')}
            </th>
            <th
              className={cn(
                'hidden px-4 py-3.5 font-medium lg:table-cell',
                mkt.muted
              )}
            >
              {t('Official pricing')}
            </th>
            <th className={cn('px-4 py-3.5 text-right font-medium', mkt.muted)}>
              {t('Save')}
            </th>
          </tr>
        </thead>
        <tbody>
          {props.isLoading ? (
            <SkeletonRows rows={6} cols={5} />
          ) : props.rows.length === 0 ? (
            <tr>
              <td
                colSpan={5}
                className={cn('px-5 py-8 text-center', mkt.muted)}
              >
                {t('Pricing data unavailable')}
              </td>
            </tr>
          ) : (
            props.rows.map((model) => {
              const official = formatHomeOfficialPricing(
                model,
                props.officialIndex
              )
              const savePercent = formatHomeSavePercent(
                model,
                props.officialIndex
              )

              return (
                <tr
                  key={model.model_name}
                  className={cn(
                    'border-b transition-colors last:border-0 hover:bg-slate-50/60 dark:hover:bg-white/[0.03]',
                    mkt.sectionBorder
                  )}
                >
                  <td className={cn('px-5 py-3.5 font-medium', mkt.heading)}>
                    <span className='font-mono text-xs sm:text-sm'>
                      {formatHomeModelDisplayName(model.model_name)}
                    </span>
                  </td>
                  <td className={cn('px-4 py-3.5 tabular-nums', mkt.body)}>
                    {formatHomeInputPrice(model, t)}
                  </td>
                  <td className={cn('px-4 py-3.5 tabular-nums', mkt.body)}>
                    {formatHomeOutputPrice(model, t)}
                  </td>
                  <td
                    className={cn(
                      'hidden px-4 py-3.5 tabular-nums lg:table-cell',
                      mkt.muted
                    )}
                  >
                    {official ?? '—'}
                  </td>
                  <td className='px-4 py-3.5 text-right'>
                    {savePercent != null ? (
                      <span className='inline-flex rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs font-semibold text-emerald-600 dark:text-emerald-400'>
                        −{savePercent}%
                      </span>
                    ) : (
                      <span className={mkt.muted}>—</span>
                    )}
                  </td>
                </tr>
              )
            })
          )}
        </tbody>
      </table>
    </PricingTableShell>
  )
}

function UnitPricingTable(props: {
  rows: PricingModel[]
  isLoading: boolean
  priceColumnLabel: string
}) {
  const { t } = useTranslation()

  return (
    <PricingTableShell>
      <table className='w-full min-w-[360px] text-sm'>
        <thead>
          <tr className={cn('border-b text-left text-xs', mkt.sectionBorder)}>
            <th className={cn('px-5 py-3.5 font-medium', mkt.muted)}>
              {t('Model')}
            </th>
            <th className={cn('px-4 py-3.5 font-medium', mkt.muted)}>
              {props.priceColumnLabel}
            </th>
          </tr>
        </thead>
        <tbody>
          {props.isLoading ? (
            <SkeletonRows rows={4} cols={2} />
          ) : props.rows.length === 0 ? null : (
            props.rows.map((model) => (
              <tr
                key={model.model_name}
                className={cn(
                  'border-b transition-colors last:border-0 hover:bg-slate-50/60 dark:hover:bg-white/[0.03]',
                  mkt.sectionBorder
                )}
              >
                <td className={cn('px-5 py-3.5 font-medium', mkt.heading)}>
                  <span className='font-mono text-xs sm:text-sm'>
                    {formatHomeModelDisplayName(model.model_name)}
                  </span>
                </td>
                <td className={cn('px-4 py-3.5 tabular-nums', mkt.body)}>
                  {formatHomeMediaPrice(model, t)}
                </td>
              </tr>
            ))
          )}
        </tbody>
      </table>
    </PricingTableShell>
  )
}

function PricingSectionBlock(props: {
  title: string
  unitLabel: string
  description?: string
  children: ReactNode
}) {
  return (
    <div className='space-y-3'>
      <div className='text-center'>
        <h3 className={cn('text-base font-semibold md:text-lg', mkt.heading)}>
          {props.title}
        </h3>
        <p className={cn('mt-0.5 text-xs', mkt.muted)}>{props.unitLabel}</p>
        {props.description ? (
          <p className={cn('mt-1 text-xs leading-relaxed', mkt.muted)}>
            {props.description}
          </p>
        ) : null}
      </div>
      {props.children}
    </div>
  )
}

export function PricingPreview() {
  const { t } = useTranslation()
  const { models, isLoading } = usePricingData()
  const { data: officialIndex = {} } = useQuery({
    queryKey: ['models-dev-official-costs'],
    queryFn: fetchModelsDevCostIndex,
    staleTime: 24 * 60 * 60 * 1000,
  })

  const sections = useMemo(
    () =>
      buildHomePricingSections(models, {
        maxPerPrefix: HOME_PRICING_MAX_PER_PREFIX,
      }),
    [models]
  )

  const hasAnyRows =
    sections.text.length > 0 ||
    sections.image.length > 0 ||
    sections.video.length > 0 ||
    sections.audio.length > 0 ||
    sections.music.length > 0

  return (
    <section
      id='pricing'
      className={cn('relative z-10 px-6 py-12 md:py-16', mkt.sectionBorder)}
    >
      <div className='mx-auto max-w-5xl'>
        <AnimateInView className='mb-8 text-center'>
          <h2 className={cn('text-lg font-semibold md:text-xl', mkt.heading)}>
            {t('Model Pricing')}
          </h2>
          <p className={cn('mt-1 text-sm', mkt.muted)}>
            {t('Home pricing preview subtitle')}
          </p>
          <p className={cn('mt-1 text-xs', mkt.muted)}>
            {t('Home pricing preview split hint')}
          </p>
        </AnimateInView>

        <AnimateInView animation='fade-up' className='space-y-10'>
          {(isLoading || sections.text.length > 0) && (
            <PricingSectionBlock
              title={t('Home text pricing title')}
              unitLabel={t('USD per 1M tokens')}
            >
              <TextPricingTable
                rows={sections.text}
                isLoading={isLoading}
                officialIndex={officialIndex}
              />
            </PricingSectionBlock>
          )}

          {(isLoading || sections.image.length > 0) && (
            <PricingSectionBlock
              title={t('Home image pricing title')}
              unitLabel={t('Home image pricing unit')}
              description={t('Home image pricing description')}
            >
              <UnitPricingTable
                rows={sections.image}
                isLoading={isLoading}
                priceColumnLabel={t('Home price per unit column')}
              />
            </PricingSectionBlock>
          )}

          {(isLoading || sections.video.length > 0) && (
            <PricingSectionBlock
              title={t('Home video pricing title')}
              unitLabel={t('Home video pricing unit')}
              description={t('Home video pricing description')}
            >
              <UnitPricingTable
                rows={sections.video}
                isLoading={isLoading}
                priceColumnLabel={t('Home price per second column')}
              />
            </PricingSectionBlock>
          )}

          {(isLoading || sections.audio.length > 0) && (
            <PricingSectionBlock
              title={t('Home audio pricing title')}
              unitLabel={t('Home audio pricing unit')}
              description={t('Home audio pricing description')}
            >
              <UnitPricingTable
                rows={sections.audio}
                isLoading={isLoading}
                priceColumnLabel={t('Home price per unit column')}
              />
            </PricingSectionBlock>
          )}

          {(isLoading || sections.music.length > 0) && (
            <PricingSectionBlock
              title={t('Home music pricing title')}
              unitLabel={t('Home music pricing unit')}
              description={t('Home music pricing description')}
            >
              <UnitPricingTable
                rows={sections.music}
                isLoading={isLoading}
                priceColumnLabel={t('Home price per unit column')}
              />
            </PricingSectionBlock>
          )}

          {!isLoading && !hasAnyRows ? (
            <p className={cn('text-center text-sm', mkt.muted)}>
              {t('Pricing data unavailable')}
            </p>
          ) : null}

          <div className='space-y-1 text-center'>
            <p className={cn('text-xs leading-relaxed', mkt.muted)}>
              {t(
                'Fixed pricing — what you see is what you pay. No fluctuation with network load, no hidden coefficient.'
              )}
            </p>
            {sections.text.length > 0 ? (
              <p className={cn('text-xs', mkt.muted)}>
                {t('Home official pricing footnote')}
              </p>
            ) : null}
          </div>

          <div className='flex justify-center'>
            <Button
              variant='outline'
              className={cn('group rounded-lg', mkt.btnGhost)}
              render={<Link to='/pricing' />}
            >
              {t('View all models')}
              <ArrowRight className='ml-1.5 size-3.5 transition-transform duration-200 group-hover:translate-x-0.5' />
            </Button>
          </div>
        </AnimateInView>
      </div>
    </section>
  )
}
