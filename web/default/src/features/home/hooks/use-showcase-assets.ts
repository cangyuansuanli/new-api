/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { SHOWCASE_ROW_A, SHOWCASE_ROW_B } from '../lib/site-assets'
import type { ShowcaseAsset } from '../types'

export const HOME_SHOWCASE_ROW_A: ShowcaseAsset[] = [...SHOWCASE_ROW_A]
export const HOME_SHOWCASE_ROW_B: ShowcaseAsset[] = [...SHOWCASE_ROW_B]

/** Scale animation duration by item count so scroll speed stays comfortable */
export function showcaseMarqueeDuration(itemCount: number, baseSec = 45): number {
  return Math.max(baseSec, Math.round(itemCount * 2.5))
}
