/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import type { PricingModel } from '../types'

/** 内部别名渠道前缀，与 infinite-canvas docs/dev/model-names.md 对齐 */
const MODEL_VENDOR_PREFIX = /^(?:gz|oairegbox|yunwu|119337)-/i

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
    const sorted = [...variants].sort((a, b) => {
      const aPrefixed = hasModelVendorPrefix(a.model_name)
      const bPrefixed = hasModelVendorPrefix(b.model_name)
      if (aPrefixed !== bPrefixed) return aPrefixed ? 1 : -1
      return a.model_name.localeCompare(b.model_name)
    })
    const primary = sorted[0]
    grouped.push({
      ...primary,
      display_name: formatModelDisplayName(primary.model_name),
      model_aliases: sorted.map((item) => item.model_name),
    })
  }

  return grouped.sort((a, b) =>
    getModelDisplayName(a).localeCompare(getModelDisplayName(b))
  )
}
