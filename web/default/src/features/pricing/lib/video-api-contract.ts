/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or (at your option)
any later version.
*/

export type VideoDocParam = { name: string; description: string }

const CANONICAL_VIDEO_PARAM_ORDER = [
  'model',
  'prompt',
  'duration',
  'aspect_ratio',
  'resolution',
  'size',
  'seed',
  'generate_audio',
  'reference_image_urls',
  'reference_videos',
  'reference_audios',
  'first_image_url',
  'last_image_url',
] as const

const CANONICAL_VIDEO_PARAMS = new Set<string>(CANONICAL_VIDEO_PARAM_ORDER)
const CAPABILITY_DECLARED_FIELDS = new Set([
  'first_image_url',
  'last_image_url',
])

/**
 * Media api_doc rows are capability notes, never an extension mechanism for
 * the public request schema. Only canonical fields already enabled by the
 * model profile may override their generated descriptions.
 */
export function mergeVideoCapabilityParams(
  generated: VideoDocParam[],
  capability: VideoDocParam[]
): VideoDocParam[] {
  const byName = new Map(
    generated
      .filter((row) => CANONICAL_VIDEO_PARAMS.has(row.name))
      .map((row) => [row.name, row])
  )

  for (const row of capability) {
    if (!CANONICAL_VIDEO_PARAMS.has(row.name)) {
      continue
    }
    const current = byName.get(row.name)
    if (!row.description.trim()) continue
    if (current) {
      byName.set(row.name, { ...current, description: row.description })
    } else if (CAPABILITY_DECLARED_FIELDS.has(row.name)) {
      byName.set(row.name, row)
    }
  }

  return CANONICAL_VIDEO_PARAM_ORDER.flatMap((name) => {
    const row = byName.get(name)
    return row ? [row] : []
  })
}
