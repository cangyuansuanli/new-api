/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { QUOTA_TYPE_VALUES } from '../constants'
import type { PricingModel } from '../types'
import { getPricingSignature } from './price'

/**
 * `/api/pricing` 已返回 public 模型名；`enrichPricingModels` 会设 `display_name = model_name`。
 * 模型广场直接展示 public 名，不再做前端前缀剔除。
 *
 * `stripModelVendorPrefix` 仅保留给 api_doc 文案里残留的 internal 注册名替换，不用于广场展示。
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
  'happyhouse',
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

/** public 路由前缀：sd1 / sd4 / sd99 等。 */
const PUBLIC_ROUTE_PREFIX_PATTERN = /^sd\d+$/i

function isPublicRoutePrefixSegment(segment: string): boolean {
  return PUBLIC_ROUTE_PREFIX_PATTERN.test(segment.trim())
}

function getNameFirstSegment(modelName: string): string | null {
  const trimmed = modelName.trim()
  const dash = trimmed.indexOf('-')
  if (dash <= 0) return null
  return trimmed.slice(0, dash).toLowerCase()
}

export function isModelFamilyFirstSegment(segment: string): boolean {
  const normalized = segment.toLowerCase()
  return (
    MODEL_FAMILY_FIRST_SEGMENTS.has(normalized) ||
    isPublicRoutePrefixSegment(normalized)
  )
}

/** 是否带有渠道注册前缀（首段不是官方模型族名）。仅用于 api_doc 文案替换等遗留场景。 */
export function hasChannelRegistrationPrefix(modelName: string): boolean {
  const first = getNameFirstSegment(modelName)
  if (!first) return false
  return !isModelFamilyFirstSegment(first)
}

/** @deprecated 模型广场勿用；仅 api_doc 内 internal 注册名 → public 名替换。 */
export function stripModelVendorPrefix(modelName: string): string {
  const trimmed = modelName.trim()
  if (!hasChannelRegistrationPrefix(trimmed)) return trimmed
  const dash = trimmed.indexOf('-')
  return trimmed.slice(dash + 1).trim()
}

export function resolvePricingDisplayName(
  model: Pick<PricingModel, 'model_name' | 'display_name'>
): string {
  return (model.display_name?.trim() || model.model_name.trim())
}

/** public 模型名原样返回；不再剔除前缀。 */
export function formatModelDisplayName(modelName: string): string {
  return modelName.trim()
}

export function getModelDisplayName(
  model: Pick<PricingModel, 'model_name' | 'display_name'>
): string {
  return resolvePricingDisplayName(model)
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

/** 模型广场：按 public 展示名合并同名的多渠道条目。画布/生成台不调用此函数。 */
export function groupPricingModelsByDisplayName(
  models: PricingModel[]
): PricingModel[] {
  const groups = new Map<string, PricingModel[]>()

  for (const model of models) {
    const key = resolvePricingDisplayName(model).toLowerCase()
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
    const displayName = resolvePricingDisplayName(primary)
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
