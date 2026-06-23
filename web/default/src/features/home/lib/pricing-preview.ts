/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { QUOTA_TYPE_VALUES } from '@/features/pricing/constants'
import {
  formatPrice,
  formatRequestPrice,
  stripTrailingZeros,
} from '@/features/pricing/lib/price'
import type { PricingModel, PriceType } from '@/features/pricing/types'
import type { TFunction } from 'i18next'
import {
  calcSavePercent,
  formatOfficialTokenPair,
  lookupModelsDevCost,
  normalizeModelLookupKey,
  type ModelsDevCost,
} from './models-dev-official'

export interface HomePricingSelectOptions {
  limit?: number
  maxPerPrefix?: number
}

export type HomePricingCategory = 'text' | 'image' | 'video' | 'audio' | 'music'

export interface HomePricingSections {
  text: PricingModel[]
  image: PricingModel[]
  video: PricingModel[]
  audio: PricingModel[]
  music: PricingModel[]
}

const HOME_MUSIC_NAME =
  /(?:^|[/_-])(?:suno|music|udio|mureka|ace-step)(?:[/_-]|$)/i
const HOME_AUDIO_NAME =
  /(?:^|[/_-])(?:speech|tts|voice|whisper|audio)(?:[/_-]|$)/i
const HOME_VIDEO_NAME =
  /(?:^|[/_-])(?:sora|veo|kling|pika|seedance|wan|hunyuanvideo|runway|luma|cogvideo|video|omni)(?:[/_-]|$)/i
const HOME_IMAGE_NAME =
  /(?:^|[/_-])(?:seedream|dalle|dall-e|imagen|flux|midjourney|stable-diffusion|gpt-image|jimeng|ideogram|recraft)(?:[/_-]|$)/i

export function classifyHomePricingModel(
  model: PricingModel
): HomePricingCategory {
  const name = model.model_name
  const lower = name.toLowerCase()
  const endpoints = model.supported_endpoint_types ?? []

  if (HOME_MUSIC_NAME.test(name)) {
    return 'music'
  }

  if (HOME_AUDIO_NAME.test(name)) {
    return 'audio'
  }

  if (/image/.test(lower)) {
    return 'image'
  }

  if (/omni/.test(lower)) {
    return 'video'
  }

  if (model.billing_mode === 'per_second') return 'video'
  if (endpoints.includes('openai-video')) return 'video'
  if (HOME_VIDEO_NAME.test(name) || /video/.test(lower)) return 'video'

  if (endpoints.includes('image-generation')) return 'image'
  if (model.request_unit === 'image') return 'image'
  if (HOME_IMAGE_NAME.test(name)) return 'image'

  if (model.quota_type === QUOTA_TYPE_VALUES.TOKEN) {
    return 'text'
  }

  if (model.quota_type === QUOTA_TYPE_VALUES.REQUEST) {
    if (/second|duration|seedance|sora|video|omni/.test(lower)) {
      return 'video'
    }
    return 'image'
  }

  return 'text'
}

/** Human-readable model name for the home pricing table (drops date / snapshot suffixes). */
export function formatHomeModelDisplayName(modelName: string): string {
  return stripModelDateSuffix(modelName)
}

function getMinGroupRatio(
  enableGroups: string[],
  groupRatio: Record<string, number>
): number {
  if (enableGroups.length === 0) return 1
  let minRatio = Number.POSITIVE_INFINITY
  for (const group of enableGroups) {
    const ratio = groupRatio[group]
    if (ratio !== undefined && ratio < minRatio) minRatio = ratio
  }
  return minRatio === Number.POSITIVE_INFINITY ? 1 : minRatio
}

function calculateTokenPriceUSD(
  model: PricingModel,
  type: PriceType,
  ratio: number
): number | null {
  const base = model.model_ratio * 2 * ratio
  switch (type) {
    case 'input':
      return base
    case 'output':
      return base * model.completion_ratio
    default:
      return null
  }
}

/** Strip date / variant suffixes so model family variants share one prefix key */
export function getModelFamilyPrefix(modelName: string): string {
  return normalizeModelLookupKey(modelName)
}

/** Strip snapshot / date suffixes from model ids (YYMMDD, YYYYMMDD, YYYY-MM-DD, …). */
export function stripModelDateSuffix(name: string): string {
  let result = name.trim()
  result = result.replace(/(-latest|-preview|-exp|-experimental)$/i, '')
  result = result.replace(/-\d{4}-\d{2}-\d{2}$/, '')
  result = result.replace(/-\d{8}$/, '')
  result = result.replace(/-\d{6}$/, '')
  result = result.replace(/-\d{4}-\d{2}$/, '')
  return result
}

export function selectHomePricingModels(
  models: PricingModel[],
  options: HomePricingSelectOptions = {}
): PricingModel[] {
  const { limit = 20, maxPerPrefix = 2 } = options
  const sorted = [...models].sort((a, b) =>
    a.model_name.localeCompare(b.model_name)
  )
  const prefixCounts = new Map<string, number>()
  const selected: PricingModel[] = []

  for (const model of sorted) {
    const prefix = getModelFamilyPrefix(model.model_name)
    const count = prefixCounts.get(prefix) ?? 0
    if (count >= maxPerPrefix) continue
    prefixCounts.set(prefix, count + 1)
    selected.push(model)
    if (selected.length >= limit) break
  }

  return selected
}

export function buildHomePricingModels(
  models: PricingModel[],
  options: HomePricingSelectOptions = {}
): PricingModel[] {
  return selectHomePricingModels(models, options)
}

export interface HomePricingSectionOptions extends HomePricingSelectOptions {
  limits?: Partial<Record<HomePricingCategory, number>>
}

const DEFAULT_HOME_SECTION_LIMITS: Record<HomePricingCategory, number> = {
  text: 12,
  image: 8,
  video: 8,
  audio: 6,
  music: 6,
}

export function buildHomePricingSections(
  models: PricingModel[],
  options: HomePricingSectionOptions = {}
): HomePricingSections {
  const limits = {
    ...DEFAULT_HOME_SECTION_LIMITS,
    ...options.limits,
  }
  const maxPerPrefix = options.maxPerPrefix ?? 2
  const buckets: HomePricingSections = {
    text: [],
    image: [],
    video: [],
    audio: [],
    music: [],
  }
  const sorted = [...models].sort((a, b) =>
    a.model_name.localeCompare(b.model_name)
  )
  const prefixCounts = new Map<string, number>()

  for (const model of sorted) {
    const category = classifyHomePricingModel(model)
    if (buckets[category].length >= limits[category]) continue

    const prefix = `${category}:${getModelFamilyPrefix(model.model_name)}`
    const count = prefixCounts.get(prefix) ?? 0
    if (count >= maxPerPrefix) continue

    prefixCounts.set(prefix, count + 1)
    buckets[category].push(model)
  }

  return buckets
}

export function formatUsdPerM(value: number | null): string {
  if (value === null || !Number.isFinite(value)) return '—'
  if (value >= 1) return `$${value.toFixed(2)}`
  if (value >= 0.01) return `$${value.toFixed(3)}`
  return `$${value.toFixed(4)}`
}

export function getOurTokenPricesUsd(
  model: PricingModel
): { input: number; output: number } | null {
  if (model.quota_type !== QUOTA_TYPE_VALUES.TOKEN) return null
  const enableGroups = Array.isArray(model.enable_groups)
    ? model.enable_groups
    : []
  const minRatio = getMinGroupRatio(enableGroups, model.group_ratio || {})
  const input = calculateTokenPriceUSD(model, 'input', minRatio)
  const output = calculateTokenPriceUSD(model, 'output', minRatio)
  if (input == null || output == null) return null
  return { input, output }
}

function formatTokenPriceWithUnit(
  model: PricingModel,
  type: 'input' | 'output',
  t: TFunction
): string {
  const price = stripTrailingZeros(
    formatPrice(model, type, 'M', false, 1, 1)
  )
  return `${price}/${t('per 1M tokens unit')}`
}

export function formatHomeInputPrice(model: PricingModel, t: TFunction): string {
  if (model.quota_type === QUOTA_TYPE_VALUES.REQUEST) {
    return stripTrailingZeros(formatRequestPrice(model, false, 1, 1, t))
  }
  return formatTokenPriceWithUnit(model, 'input', t)
}

export function formatHomeUnitPrice(model: PricingModel, t: TFunction): string {
  return stripTrailingZeros(formatRequestPrice(model, false, 1, 1, t))
}

/** Price cell for image / video / audio / music tables (supports token-priced edge cases). */
export function formatHomeMediaPrice(model: PricingModel, t: TFunction): string {
  if (model.quota_type === QUOTA_TYPE_VALUES.TOKEN) {
    return formatTokenPriceWithUnit(model, 'input', t)
  }
  return formatHomeUnitPrice(model, t)
}

export function formatHomeOutputPrice(
  model: PricingModel,
  t: TFunction
): string {
  if (model.quota_type === QUOTA_TYPE_VALUES.REQUEST) {
    return '—'
  }
  return formatTokenPriceWithUnit(model, 'output', t)
}

export function formatHomeOfficialPricing(
  model: PricingModel,
  officialIndex: Record<string, ModelsDevCost>
): string | null {
  const cost = lookupModelsDevCost(officialIndex, model.model_name)
  if (!cost) return null

  if (model.quota_type === QUOTA_TYPE_VALUES.REQUEST) {
    if (cost.input == null || !Number.isFinite(cost.input)) return null
    if (cost.output != null && cost.output > 0) {
      return formatOfficialTokenPair(cost, formatUsdPerM)
    }
    return formatUsdPerM(cost.input)
  }

  return formatOfficialTokenPair(cost, formatUsdPerM)
}

export function formatHomeSavePercent(
  model: PricingModel,
  officialIndex: Record<string, ModelsDevCost>
): number | null {
  const cost = lookupModelsDevCost(officialIndex, model.model_name)
  const ours = getOurTokenPricesUsd(model)
  return calcSavePercent(model, cost, ours?.input ?? null, ours?.output ?? null)
}
