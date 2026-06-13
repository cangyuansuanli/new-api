/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

Public canvas links open the playground directly. Trust-token SSO is only
used after an explicit New-API sign-in with a canvas redirect target.
*/
import { useEffect, useState } from 'react'
import { useAuthStore } from '@/stores/auth-store'
import {
  appendTrustTokenToUrl,
  fetchCanvasTrustToken,
} from '@/features/canvas/api'

type CanvasEntryOptions = {
  /** When true and user is signed in, append a one-time trust token. */
  withTrust?: boolean
}

export function useCanvasEntryUrl(
  canvasBaseUrl: string,
  options: CanvasEntryOptions = {}
) {
  const { withTrust = false } = options
  const user = useAuthStore((state) => state.auth.user)
  const [url, setUrl] = useState(canvasBaseUrl)

  useEffect(() => {
    if (!withTrust || !user) {
      setUrl(canvasBaseUrl)
      return
    }

    let cancelled = false
    void fetchCanvasTrustToken()
      .then(({ token, canvasUrl }) => {
        if (cancelled) return
        const base = canvasUrl || canvasBaseUrl
        setUrl(appendTrustTokenToUrl(base, token))
      })
      .catch(() => {
        if (!cancelled) setUrl(canvasBaseUrl)
      })

    return () => {
      cancelled = true
    }
  }, [canvasBaseUrl, user, withTrust])

  return url
}
