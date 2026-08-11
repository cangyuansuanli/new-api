/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import { DEFAULT_CANVAS_BASE_URL } from '@/features/canvas/lib/canvas-config'

/** 将当前路由位置序列化为登录 redirect 参数（仅 pathname + search + hash）。 */
export function toAuthRedirectParam(location: {
  pathname: string
  searchStr?: string
  search?: string
  hash?: string
}): string {
  const search = location.searchStr ?? location.search ?? ''
  const hash = location.hash ?? ''
  return `${location.pathname}${search}${hash}`
}

/**
 * 解析登录后的 redirect 目标。
 * 同域完整 URL 会降级为站内路径，避免 TanStack Router 将 href 当作 route id。
 */
export function resolveAuthRedirectTarget(redirect?: string): string | undefined {
  if (!redirect?.trim()) {
    return undefined
  }

  let trimmed = redirect.trim()

  // Reject control characters and collapse the nested values produced by
  // older clients/proxies.  Without this guard, /sign-in?redirect=/sign-in
  // can recursively create a phishing-looking URL.
  if ([...trimmed].some((char) => {
    const code = char.charCodeAt(0)
    return code < 0x20 || code === 0x7f
  })) {
    return undefined
  }
  for (let i = 0; i < 3; i += 1) {
    try {
      const decoded = decodeURIComponent(trimmed)
      if (decoded === trimmed) break
      trimmed = decoded
    } catch {
      break
    }
  }

  if (trimmed.startsWith('http://') || trimmed.startsWith('https://')) {
    try {
      const url = new URL(trimmed)
      if (typeof window !== 'undefined' && url.origin === window.location.origin) {
        return `${url.pathname}${url.search}${url.hash}`
      }
      // The canvas is the only supported cross-origin post-auth destination.
      // Reject arbitrary external URLs to prevent open redirects and phishing.
      if (url.origin === new URL(DEFAULT_CANVAS_BASE_URL).origin) {
        return `${url.origin}${url.pathname}${url.search}${url.hash}`
      }
      return undefined
    } catch {
      return undefined
    }
  }

  const internal = trimmed.startsWith('/') ? trimmed : `/${trimmed}`
  try {
    const url = new URL(internal, 'http://localhost')
    // Never redirect back to an auth entry point. This breaks redirect loops
    // and prevents nested sign-in URLs from being generated.
    if (
      ['/sign-in', '/sign-up', '/forgot-password', '/reset', '/otp'].includes(
        url.pathname
      )
    ) {
      return undefined
    }
    return `${url.pathname}${url.search}${url.hash}`
  } catch {
    return undefined
  }
}

export function parseInternalRedirectPath(path: string): {
  to: string
  search?: Record<string, string>
} {
  try {
    const url = new URL(path, 'http://localhost')
    const search: Record<string, string> = {}
    url.searchParams.forEach((value, key) => {
      search[key] = value
    })
    return {
      to: url.pathname,
      search: Object.keys(search).length > 0 ? search : undefined,
    }
  } catch {
    return { to: path }
  }
}
