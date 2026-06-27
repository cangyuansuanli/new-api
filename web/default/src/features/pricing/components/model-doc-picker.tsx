/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useQuery } from '@tanstack/react-query'
import { getLobeIcon } from '@/lib/lobe-icon'
import { getPricing } from '../api'
import { groupPricingModelsByDisplayName } from '../lib/model-display-name'
import type { PricingModel } from '../types'
import { ModelDocDialog } from './model-doc-dialog'

type ModelDocPickerProps = {
  siteOrigin: string
  /** 仅展示带视频或图像 UI 配置的模型 */
  capability?: 'video' | 'image' | 'all'
  className?: string
}

type ModelDocCapability = 'video' | 'image'

type VendorModelGroup = {
  vendorName: string
  vendorIcon?: string
  models: PricingModel[]
}

type CapabilityModelGroup = {
  capability: ModelDocCapability
  vendors: VendorModelGroup[]
}

function groupModelsByCapabilityAndVendor(
  models: PricingModel[],
  uncategorizedLabel: string
): CapabilityModelGroup[] {
  const capabilities: ModelDocCapability[] = ['video', 'image']
  const result: CapabilityModelGroup[] = []

  for (const capability of capabilities) {
    const filtered = models.filter((model) =>
      capability === 'video'
        ? Boolean(model.video_ui_params)
        : Boolean(model.image_ui_params)
    )
    if (filtered.length === 0) continue

    const vendorMap = new Map<string, VendorModelGroup>()
    for (const model of filtered) {
      const vendorName = model.vendor_name?.trim() || uncategorizedLabel
      const existing = vendorMap.get(vendorName)
      if (existing) {
        existing.models.push(model)
        if (!existing.vendorIcon && model.vendor_icon) {
          existing.vendorIcon = model.vendor_icon
        }
      } else {
        vendorMap.set(vendorName, {
          vendorName,
          vendorIcon: model.vendor_icon,
          models: [model],
        })
      }
    }

    const vendors = Array.from(vendorMap.values())
      .map((vendor) => ({
        ...vendor,
        models: [...vendor.models].sort((a, b) =>
          (a.display_name || a.model_name).localeCompare(
            b.display_name || b.model_name,
            'zh-CN'
          )
        ),
      }))
      .sort((a, b) => a.vendorName.localeCompare(b.vendorName, 'zh-CN'))

    result.push({ capability, vendors })
  }

  return result
}

function ModelDocPickerButton(props: {
  model: PricingModel
  onClick: () => void
}) {
  const iconKey = props.model.vendor_icon || props.model.icon
  const icon = iconKey ? getLobeIcon(iconKey, 16) : null
  const initial =
    (props.model.display_name || props.model.model_name)?.charAt(0).toUpperCase() ||
    '?'

  return (
    <button
      type='button'
      onClick={props.onClick}
      className='border-input bg-background hover:bg-muted/60 inline-flex items-center gap-1.5 rounded-lg border px-3 py-1.5 text-xs font-medium transition-colors'
    >
      {icon ? (
        <span className='inline-flex shrink-0 items-center'>{icon}</span>
      ) : (
        <span className='bg-muted text-muted-foreground inline-flex size-4 shrink-0 items-center justify-center rounded text-[10px] font-semibold'>
          {initial}
        </span>
      )}
      {props.model.display_name || props.model.model_name}
    </button>
  )
}

export function ModelDocPicker(props: ModelDocPickerProps) {
  const { t } = useTranslation()
  const [docModel, setDocModel] = useState<PricingModel | null>(null)
  const capability = props.capability ?? 'all'

  const pricingQuery = useQuery({
    queryKey: ['pricing', 'model-doc-picker'],
    queryFn: getPricing,
    staleTime: 5 * 60 * 1000,
  })

  const capabilityGroups = useMemo(() => {
    const raw = pricingQuery.data?.data ?? []
    const grouped = groupPricingModelsByDisplayName(raw)
    const filtered = grouped.filter((model) => {
      if (capability === 'video') return Boolean(model.video_ui_params)
      if (capability === 'image') return Boolean(model.image_ui_params)
      return Boolean(model.video_ui_params || model.image_ui_params)
    })
    return groupModelsByCapabilityAndVendor(
      filtered,
      t('modelDoc.uncategorizedVendor')
    )
  }, [pricingQuery.data?.data, capability, t])

  const hasModels = capabilityGroups.some((group) => group.vendors.length > 0)

  if (pricingQuery.isLoading) {
    return (
      <p className='text-muted-foreground text-sm'>{t('Loading pricing data...')}</p>
    )
  }

  if (!hasModels) {
    return (
      <p className='text-muted-foreground text-sm'>
        {t('modelDoc.pickerEmpty')}
      </p>
    )
  }

  return (
    <div className={props.className}>
      <p className='text-muted-foreground mb-4 text-sm leading-relaxed'>
        {t('modelDoc.pickerHint')}
      </p>
      <div className='space-y-6'>
        {capabilityGroups.map((group) => (
          <div key={group.capability} className='space-y-4'>
            <h4 className='text-foreground text-sm font-semibold'>
              {group.capability === 'video'
                ? t('modelDoc.sectionVideo')
                : t('modelDoc.sectionImage')}
            </h4>
            <div className='space-y-4'>
              {group.vendors.map((vendor) => {
                const vendorIcon = vendor.vendorIcon
                  ? getLobeIcon(vendor.vendorIcon, 18)
                  : null
                return (
                  <div key={`${group.capability}-${vendor.vendorName}`}>
                    <div className='mb-2 flex items-center gap-2'>
                      {vendorIcon ? (
                        <span className='inline-flex shrink-0 items-center'>
                          {vendorIcon}
                        </span>
                      ) : null}
                      <span className='text-muted-foreground text-xs font-medium'>
                        {vendor.vendorName}
                      </span>
                    </div>
                    <div className='flex flex-wrap gap-2'>
                      {vendor.models.map((model) => (
                        <ModelDocPickerButton
                          key={model.model_name}
                          model={model}
                          onClick={() => setDocModel(model)}
                        />
                      ))}
                    </div>
                  </div>
                )
              })}
            </div>
          </div>
        ))}
      </div>

      <ModelDocDialog
        model={docModel}
        siteOrigin={props.siteOrigin}
        open={Boolean(docModel)}
        onOpenChange={(open) => {
          if (!open) setDocModel(null)
        }}
      />
    </div>
  )
}
