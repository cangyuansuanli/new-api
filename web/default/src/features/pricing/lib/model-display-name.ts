/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import type { PricingModel } from '../types'
import { getPricingSignature } from './price'

/**
 * 内部别名渠道前缀。
 * 规则：渠道 base_url 的域名主体（如 ctlove.cn → ctlove，api.119337.xyz → 119337）。
 * 与 infinite-canvas docs/dev/model-names.md 及 NewAPI 渠道注册名对齐。
 */
const MODEL_VENDOR_PREFIX =
  /^(?:gz|oairegbox|yunwu|119337|byte|niming|zeabur|happyhorse|ctlove)-/i

export function stripModelVendorPrefix(modelName: string) {
  return modelName.replace(MODEL_VENDOR_PREFIX, '')
}

export function formatModelDisplayName(modelName: string) {
  return stripModelVendorPrefix(modelName.trim())
}

export function getModelDisplayName(
  model: Pick<PricingModel, 'model_name' | 'display_name'>
) {
  return model.display_name || formatModelDisplayName(model.model_name)
}

function hasModelVendorPrefix(modelName: string) {
  return MODEL_VENDOR_PREFIX.test(modelName)
}

function mergeEnableGroups(variants: PricingModel[]): string[] {
  const groups = new Set<string>()
  for (const variant of variants) {
    for (const group of variant.enable_groups ?? []) {
      if (group) groups.add(group)
    }
  }
  return Array.from(groups)
}

function pickPrimaryVariant(variants: PricingModel[]): PricingModel {
  return [...variants].sort((a, b) => {
    const aPrefixed = hasModelVendorPrefix(a.model_name)
    const bPrefixed = hasModelVendorPrefix(b.model_name)
    if (aPrefixed !== bPrefixed) return aPrefixed ? 1 : -1
    return a.model_name.localeCompare(b.model_name)
  })[0]
}

/** 模型广场：按展示名合并多渠道别名，减少重复条目。画布/生成台不调用此函数。 */
export function groupPricingModelsByDisplayName(
  models: PricingModel[]
): PricingModel[] {
  const groups = new Map<string, PricingModel[]>()

  for (const model of models) {
    const key = formatModelDisplayName(model.model_name).toLowerCase()
    const bucket = groups.get(key) ?? []
    bucket.push(model)
    groups.set(key, bucket)
  }

  const grouped: PricingModel[] = []

  for (const variants of groups.values()) {
    const sorted = [...variants].sort((a, b) =>
      a.model_name.localeCompare(b.model_name)
    )
    const primary = pickPrimaryVariant(sorted)
    const displayName = formatModelDisplayName(primary.model_name)
    const signatures = new Set(sorted.map(getPricingSignature))
    const hasVariantPricing = signatures.size > 1

    grouped.push({
      ...primary,
      display_name: displayName,
      model_aliases: sorted.map((item) => item.model_name),
      enable_groups: mergeEnableGroups(sorted),
      ...(hasVariantPricing
        ? {
            pricing_variants: sorted.sort((a, b) =>
              a.model_name.localeCompare(b.model_name)
            ),
          }
        : {}),
    })
  }

  return grouped.sort((a, b) =>
    getModelDisplayName(a).localeCompare(getModelDisplayName(b))
  )
}
