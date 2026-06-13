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
import {
  appendTrustTokenToUrl,
  fetchCanvasTrustToken,
} from '@/features/canvas/api'
import { DEFAULT_CANVAS_BASE_URL } from '@/features/canvas/lib/canvas-config'

function normalizeOrigin(url: string) {
  try {
    return new URL(url).origin
  } catch {
    return ''
  }
}

export function isCanvasRedirectUrl(
  redirectTo: string,
  canvasBaseUrl = DEFAULT_CANVAS_BASE_URL
) {
  const targetOrigin = normalizeOrigin(redirectTo)
  const canvasOrigin = normalizeOrigin(canvasBaseUrl)
  return Boolean(targetOrigin && canvasOrigin && targetOrigin === canvasOrigin)
}

export function isExternalRedirect(redirectTo: string) {
  if (!redirectTo.startsWith('http://') && !redirectTo.startsWith('https://')) {
    return false
  }
  try {
    const target = new URL(redirectTo)
    return target.origin !== window.location.origin
  } catch {
    return false
  }
}

export async function resolveCanvasRedirectUrl(redirectTo: string) {
  const { token, canvasUrl } = await fetchCanvasTrustToken()
  const base = redirectTo || canvasUrl || DEFAULT_CANVAS_BASE_URL
  return appendTrustTokenToUrl(base, token)
}
