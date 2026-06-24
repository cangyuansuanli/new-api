/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import type { PricingModel } from '../types'
import { getPricingSignature } from './price'
import { QUOTA_TYPE_VALUES } from '../constants'

/**
 * 官方模型名首段（`-` 前）。若首段属于此集合，则视为模型本名而非渠道别名前缀。
 * 渠道注册名形如 `{渠道}-{模型}`；首段不在此集合时去掉渠道前缀。
 */
const MODEL_FAMILY_FIRST_SEGMENTS = new Set([
  'gpt',
  'claude',
  'gemini',
  'gemma',
  'grok',
  'imagen',
  'veo',
  'palm',
  'o1',
  'o2',
  'o3',
  'o4',
  'omni',
  'sora',
  'dall',
  'dalle',
  'whisper',
  'tts',
  'davinci',
  'babbage',
  'text',
  'embed',
  'embedding',
  'llama',
  'codellama',
  'mistral',
  'mixtral',
  'codestral',
  'magistral',
  'pixtral',
  'qwen',
  'qwq',
  'qvq',
  'deepseek',
  'command',
  'cohere',
  'aya',
  'ernie',
  'wenxin',
  'hunyuan',
  'hunyuanvideo',
  'glm',
  'chatglm',
  'cogview',
  'cogvideo',
  'kimi',
  'moonshot',
  'abab',
  'minimax',
  'hailuo',
  'doubao',
  'seedance',
  'seedream',
  'jimeng',
  'kling',
  'wan',
  'pika',
  'runway',
  'luma',
  'flux',
  'ideogram',
  'recraft',
  'midjourney',
  'niji',
  'sd',
  'sdxl',
  'stable',
  'suno',
  'udio',
  'mureka',
  'meta',
])

function getNameFirstSegment(modelName: string): string | null {
  const trimmed = modelName.trim()
  const dash = trimmed.indexOf('-')
  if (dash <= 0) return null
  return trimmed.slice(0, dash).toLowerCase()
}

export function isModelFamilyFirstSegment(segment: string): boolean {
  return MODEL_FAMILY_FIRST_SEGMENTS.has(segment.toLowerCase())
}

/** 是否带有渠道注册前缀（首段不是官方模型族名）。 */
export function hasChannelRegistrationPrefix(modelName: string): boolean {
  const first = getNameFirstSegment(modelName)
  if (!first) return false
  return !isModelFamilyFirstSegment(first)
}

export function stripModelVendorPrefix(modelName: string): string {
  const trimmed = modelName.trim()
  if (!hasChannelRegistrationPrefix(trimmed)) return trimmed
  const dash = trimmed.indexOf('-')
  return trimmed.slice(dash + 1).trim()
}

export function formatModelDisplayName(modelName: string) {
  return stripModelVendorPrefix(modelName.trim())
}

export function getModelDisplayName(
  model: Pick<PricingModel, 'model_name' | 'display_name'>
) {
  return model.display_name || formatModelDisplayName(model.model_name)
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

function variantPricingScore(model: PricingModel): number {
  let score = 0
  if (model.billing_mode === 'tiered_expr' && model.billing_expr?.trim()) {
    score += 4
  }
  if (
    model.quota_type === QUOTA_TYPE_VALUES.TOKEN &&
    (model.model_ratio ?? 0) > 0
  ) {
    score += 3
  }
  if (
    model.quota_type === QUOTA_TYPE_VALUES.REQUEST &&
    (model.model_price ?? 0) > 0
  ) {
    score += 3
  }
  if (!hasChannelRegistrationPrefix(model.model_name)) {
    score += 1
  }
  return score
}

function pickPrimaryVariant(variants: PricingModel[]): PricingModel {
  return [...variants].sort((a, b) => {
    const scoreDiff = variantPricingScore(b) - variantPricingScore(a)
    if (scoreDiff !== 0) return scoreDiff
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
