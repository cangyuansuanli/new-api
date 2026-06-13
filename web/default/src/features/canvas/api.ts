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
import { api } from '@/lib/api'

export type CanvasTrustTokenResponse = {
  token: string
  canvasUrl: string
  expiresIn: number
}

export async function fetchCanvasTrustToken(): Promise<CanvasTrustTokenResponse> {
  const res = await api.get('/api/user/canvas/trust-token')
  if (!res?.data?.success || !res.data?.data?.token) {
    throw new Error(res?.data?.message || 'Failed to create canvas trust token')
  }
  return res.data.data as CanvasTrustTokenResponse
}

export function appendTrustTokenToUrl(baseUrl: string, trustToken: string) {
  if (!trustToken) return baseUrl
  try {
    const url = new URL(baseUrl)
    url.searchParams.set('trustToken', trustToken)
    return url.toString()
  } catch {
    const separator = baseUrl.includes('?') ? '&' : '?'
    return `${baseUrl}${separator}trustToken=${encodeURIComponent(trustToken)}`
  }
}
