/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

const CHUNK_RELOAD_MARKER_KEY = 'newapi:chunk-reload-marker'
const CHUNK_RELOAD_MARKER_CLEAR_DELAY_MS = 10_000

const CHUNK_LOAD_ERROR_PATTERNS = [
  'chunkloaderror',
  'loading chunk',
  'loading css chunk',
  'failed to fetch dynamically imported module',
  'error loading dynamically imported module',
  'importing a module script failed',
  'expected a javascript-or-wasm module script',
  'module script failed',
]

function getErrorText(error: unknown): string {
  if (typeof error === 'string') return error
  if (typeof error !== 'object' || error === null) return ''

  const record = error as Record<string, unknown>
  const parts = [record.name, record.message]
    .filter((value): value is string => typeof value === 'string')
    .join(' ')

  const cause = record.cause === error ? '' : getErrorText(record.cause)
  return `${parts} ${cause}`.trim()
}

export function isChunkLoadError(error: unknown): boolean {
  const errorText = getErrorText(error).toLowerCase()
  return CHUNK_LOAD_ERROR_PATTERNS.some((pattern) =>
    errorText.includes(pattern)
  )
}

export function reloadAfterChunkLoadError(error: unknown): boolean {
  if (!isChunkLoadError(error) || typeof window === 'undefined') return false

  const marker = window.location.href
  try {
    if (window.sessionStorage.getItem(CHUNK_RELOAD_MARKER_KEY) === marker) {
      return false
    }
    window.sessionStorage.setItem(CHUNK_RELOAD_MARKER_KEY, marker)
  } catch {
    return false
  }

  try {
    window.location.reload()
    return true
  } catch {
    try {
      window.sessionStorage.removeItem(CHUNK_RELOAD_MARKER_KEY)
    } catch {
      // Storage can be unavailable in restricted browsing contexts.
    }
    return false
  }
}

export function scheduleChunkReloadMarkerClear(): void {
  if (typeof window === 'undefined') return

  window.setTimeout(() => {
    try {
      window.sessionStorage.removeItem(CHUNK_RELOAD_MARKER_KEY)
    } catch {
      // Storage can be unavailable in restricted browsing contexts.
    }
  }, CHUNK_RELOAD_MARKER_CLEAR_DELAY_MS)
}
