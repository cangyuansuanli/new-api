/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useQuery } from '@tanstack/react-query'
import {
  INSPIRATION_SLIDES,
  SHOWCASE_MANIFEST_URL,
} from '../lib/site-assets'
import type { ShowcaseAsset } from '../types'

const ONE_DAY_MS = 24 * 60 * 60 * 1000
const DEFAULT_ASPECT_RATIO = 3 / 4

type ShowcaseManifest = {
  version?: number
  generatedAt?: string
  assets?: ShowcaseAsset[]
}

function normalizeShowcaseAsset(raw: ShowcaseAsset): ShowcaseAsset | null {
  if (!raw?.id || !raw.image || !raw.image.startsWith('http')) return null
  const width = raw.width
  const height = raw.height
  const aspectRatio =
    width && height && width > 0 && height > 0
      ? width / height
      : raw.aspectRatio && raw.aspectRatio > 0
        ? raw.aspectRatio
        : DEFAULT_ASPECT_RATIO
  return {
    id: raw.id,
    image: raw.image,
    title: raw.title,
    tags: raw.tags,
    width,
    height,
    aspectRatio,
  }
}

function fallbackAssets(): ShowcaseAsset[] {
  return INSPIRATION_SLIDES.map((slide) => ({
    id: slide.id,
    image: slide.image,
    title: slide.title,
    tags: slide.tags,
    aspectRatio: slide.width / slide.height,
  }))
}

/** Read pre-built daily manifest from static CDN (no canvas API at runtime). */
async function fetchShowcaseAssets(): Promise<ShowcaseAsset[]> {
  try {
    const res = await fetch(SHOWCASE_MANIFEST_URL)
    if (!res.ok) return fallbackAssets()

    const data = (await res.json()) as ShowcaseManifest
    const assets = data.assets
    if (!Array.isArray(assets) || assets.length === 0) {
      return fallbackAssets()
    }

    return assets
      .map((item) => normalizeShowcaseAsset(item))
      .filter((item): item is ShowcaseAsset => item != null)
  } catch {
    return fallbackAssets()
  }
}

export function useShowcaseAssets() {
  return useQuery({
    queryKey: ['home-showcase-assets', SHOWCASE_MANIFEST_URL],
    queryFn: fetchShowcaseAssets,
    staleTime: ONE_DAY_MS,
    gcTime: ONE_DAY_MS * 2,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    refetchOnMount: false,
  })
}

/** Split assets into two rows for opposite marquee directions */
export function splitShowcaseRows(assets: ShowcaseAsset[]) {
  const rowA: ShowcaseAsset[] = []
  const rowB: ShowcaseAsset[] = []
  assets.forEach((asset, index) => {
    if (index % 2 === 0) rowA.push(asset)
    else rowB.push(asset)
  })
  if (rowB.length === 0 && rowA.length > 1) {
    const half = Math.ceil(rowA.length / 2)
    return {
      rowA: rowA.slice(0, half),
      rowB: rowA.slice(half),
    }
  }
  return { rowA, rowB }
}

/** Scale animation duration by item count so scroll speed stays comfortable */
export function showcaseMarqueeDuration(itemCount: number, baseSec = 45): number {
  return Math.max(baseSec, Math.round(itemCount * 2.5))
}
