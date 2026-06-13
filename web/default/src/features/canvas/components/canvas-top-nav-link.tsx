/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'
import { DEFAULT_CANVAS_BASE_URL } from '@/features/canvas/lib/canvas-config'
import { useCanvasEntryUrl } from '@/features/canvas/hooks/use-canvas-entry-url'

type CanvasTopNavLinkProps = {
  className?: string
  style?: React.CSSProperties
  onClick?: () => void
}

export function CanvasTopNavLink({ className, style, onClick }: CanvasTopNavLinkProps) {
  const { t } = useTranslation()
  const isAuthenticated = !!useAuthStore((state) => state.auth.user)
  const canvasUrl = useCanvasEntryUrl(DEFAULT_CANVAS_BASE_URL, {
    withTrust: isAuthenticated,
  })

  return (
    <a
      href={canvasUrl}
      target='_blank'
      rel='noopener noreferrer'
      onClick={onClick}
      style={style}
      className={cn(
        'hover:text-primary text-sm font-medium transition-colors text-muted-foreground',
        className
      )}
    >
      {t('Cangyuan Image to Video')}
    </a>
  )
}
