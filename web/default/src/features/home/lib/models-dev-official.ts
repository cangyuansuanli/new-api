/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { MODELS_DEV_PRESET_ENDPOINT } from '@/features/system-settings/models/constants'
import type { PricingModel } from '@/features/pricing/types'
import { QUOTA_TYPE_VALUES } from '@/features/pricing/constants'
import { stripModelVendorPrefix } from '@/features/pricing/lib/model-display-name'

/** Official $/1M token costs from models.dev (same source as admin ratio sync preset) */
export interface ModelsDevCost {
  input?: number
  output?: number
  cache_read?: number
  cache_write?: number
}

type ModelsDevProvider = {
  models?: Record<string, { cost?: ModelsDevCost }>
}

/** Canonical key for fuzzy model-name matching (local renames, provider prefixes, date suffixes). */
export function normalizeModelLookupKey(name: string): string {
  let key = name.trim().toLowerCase()
  if (key.includes('/')) {
    key = key.split('/').pop() ?? key
  }
  key = key.replace(/_/g, '-')
  key = key.replace(/\./g, '-')
  key = key.replace(/(-latest|-preview|-exp|-experimental)$/i, '')
  key = key.replace(/-\d{4}-\d{2}-\d{2}$/, '')
  key = key.replace(/-\d{8}$/, '')
  key = key.replace(/-\d{6}$/, '')
  key = key.replace(/-\d{4}-\d{2}$/, '')
  return key
}

function stripDoubaoPrefix(key: string): string {
  return key.startsWith('doubao-') ? key.slice('doubao-'.length) : key
}

/** Strip -thinking / -nothinking / -thinking-{budget} suffixes for official price lookup. */
function stripThinkingSuffix(key: string): string | null {
  if (key.endsWith('-nothinking')) {
    return key.slice(0, -'-nothinking'.length)
  }
  if (key.endsWith('-thinking')) {
    return key.slice(0, -'-thinking'.length)
  }
  const budgetMatch = key.match(/^(.+)-thinking-\d+$/)
  if (budgetMatch) return budgetMatch[1]
  return null
}

function registerLookupAlias(
  index: Record<string, ModelsDevCost>,
  alias: string,
  cost: ModelsDevCost
) {
  const trimmed = alias.trim()
  if (!trimmed) return

  const candidates = [trimmed, normalizeModelLookupKey(trimmed)]
  for (const key of candidates) {
    if (!key) continue
    const existing = index[key]
    if (!existing) {
      index[key] = cost
      continue
    }
    const existingInput = existing.input ?? Number.POSITIVE_INFINITY
    const nextInput = cost.input ?? Number.POSITIVE_INFINITY
    if (nextInput > 0 && nextInput < existingInput) {
      index[key] = cost
    }
  }
}

/** Expand a models.dev id or local model name into lookup aliases. */
export function expandModelLookupAliases(modelName: string): string[] {
  const raw = modelName.trim()
  if (!raw) return []

  const aliases = new Set<string>([raw])
  const normalized = normalizeModelLookupKey(raw)
  aliases.add(normalized)

  const withoutChannelPrefix = stripModelVendorPrefix(raw)
  if (withoutChannelPrefix && withoutChannelPrefix !== raw) {
    aliases.add(withoutChannelPrefix)
    aliases.add(normalizeModelLookupKey(withoutChannelPrefix))
  }

  const withoutThinking = stripThinkingSuffix(normalized)
  if (withoutThinking && withoutThinking !== normalized) {
    aliases.add(withoutThinking)
  }

  const withoutDoubao = stripDoubaoPrefix(normalized)
  if (withoutDoubao !== normalized) {
    aliases.add(withoutDoubao)
    aliases.add(`bytedance/${withoutDoubao}`)
  }

  if (
    withoutDoubao.startsWith('seedream') ||
    withoutDoubao.startsWith('seedance')
  ) {
    aliases.add(`doubao-${withoutDoubao}`)
    aliases.add(`bytedance/${withoutDoubao.replace(/\./g, '-')}`)
  }

  if (raw.includes('/')) {
    const short = raw.split('/').pop()
    if (short) aliases.add(short)
  }

  return [...aliases]
}

export async function fetchModelsDevCostIndex(): Promise<
  Record<string, ModelsDevCost>
> {
  const res = await fetch(MODELS_DEV_PRESET_ENDPOINT)
  if (!res.ok) return {}

  const data = (await res.json()) as Record<string, ModelsDevProvider>
  const index: Record<string, ModelsDevCost> = {}

  for (const provider of Object.values(data)) {
    if (!provider?.models) continue
    for (const [modelId, meta] of Object.entries(provider.models)) {
      if (!meta?.cost) continue
      for (const alias of expandModelLookupAliases(modelId)) {
        registerLookupAlias(index, alias, meta.cost)
      }
    }
  }

  return index
}

export function lookupModelsDevCost(
  index: Record<string, ModelsDevCost>,
  modelName: string
): ModelsDevCost | null {
  for (const alias of expandModelLookupAliases(modelName)) {
    const hit = index[alias] ?? index[normalizeModelLookupKey(alias)]
    if (hit) return hit
  }
  return null
}

export function formatOfficialTokenPair(
  cost: ModelsDevCost | null,
  formatUsd: (value: number) => string
): string | null {
  if (!cost) return null
  const input = cost.input
  const output = cost.output
  if (input == null || output == null) return null
  if (!Number.isFinite(input) || !Number.isFinite(output)) return null
  return `${formatUsd(input)} / ${formatUsd(output)}`
}

export function calcSavePercent(
  model: PricingModel,
  cost: ModelsDevCost | null,
  ourInputUsd: number | null,
  ourOutputUsd: number | null
): number | null {
  if (!cost) return null

  if (model.quota_type === QUOTA_TYPE_VALUES.REQUEST) {
    const official = cost.input
    const ours = model.model_price
    if (
      official == null ||
      ours == null ||
      !Number.isFinite(official) ||
      !Number.isFinite(ours) ||
      official <= 0 ||
      ours <= 0
    ) {
      return null
    }
    const save = Math.round((1 - ours / official) * 100)
    return save > 0 ? save : null
  }

  if (ourInputUsd == null || ourOutputUsd == null) return null
  if (cost.input == null || cost.output == null) return null
  if (cost.input <= 0 && cost.output <= 0) return null

  const avgOur = (ourInputUsd + ourOutputUsd) / 2
  const avgOfficial = (cost.input + cost.output) / 2
  if (avgOfficial <= 0) return null

  const save = Math.round((1 - avgOur / avgOfficial) * 100)
  return save > 0 ? save : null
}
