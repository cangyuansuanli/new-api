/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useEffect, useMemo } from 'react'
import { Link } from '@tanstack/react-router'
import { ArrowRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { PublicLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { useSystemConfig } from '@/hooks/use-system-config'
import { docsNavItems } from './docs-nav'
import { UserDocsSections } from './user-docs-sections'

export function UserDocsPage() {
  const { t } = useTranslation()
  const { systemName } = useSystemConfig()

  const siteOrigin = useMemo(() => {
    if (typeof window === 'undefined') return ''
    return window.location.origin
  }, [])

  const displayName = systemName?.trim() || '沧元算力'

  useEffect(() => {
    document.title = t('userDocs.pageTitle', { siteName: displayName })
  }, [displayName, t])

  return (
    <PublicLayout>
      <div className='mx-auto flex max-w-6xl flex-col gap-10 pb-16 lg:flex-row lg:gap-12'>
        <aside className='lg:w-56 lg:shrink-0'>
          <div className='lg:sticky lg:top-24'>
            <p className='text-muted-foreground mb-3 text-xs font-semibold tracking-[0.16em] uppercase'>
              {t('userDocs.sidebarLabel')}
            </p>
            <nav className='flex flex-wrap gap-2 lg:flex-col lg:gap-0.5'>
              {docsNavItems.map((item) => (
                <a
                  key={item.id}
                  href={`#${item.id}`}
                  className={cn(
                    'text-muted-foreground hover:text-foreground rounded-lg px-3 py-2 text-sm transition-colors lg:block',
                    'hover:bg-muted/60'
                  )}
                >
                  {t(item.titleKey)}
                </a>
              ))}
            </nav>
          </div>
        </aside>

        <div className='min-w-0 flex-1'>
          <header className='mb-10 space-y-4'>
            <p className='text-primary text-xs font-semibold tracking-[0.2em] uppercase'>
              {t('userDocs.eyebrow')}
            </p>
            <h1 className='text-3xl font-bold tracking-tight md:text-4xl'>
              {t('userDocs.title', { siteName: displayName })}
            </h1>
            <p className='text-muted-foreground max-w-2xl text-base leading-relaxed'>
              {t('userDocs.subtitle')}
            </p>
            <div className='flex flex-wrap gap-3 pt-1'>
              <Button render={<Link to='/sign-up' search={{ redirect: undefined }} />}>
                {t('Get API Key')}
                <ArrowRight className='size-4' />
              </Button>
              <Button variant='outline' render={<Link to='/keys' />}>
                {t('userDocs.nav.apiKey')}
              </Button>
              <Button variant='outline' render={<Link to='/pricing' />}>
                {t('Model Square')}
              </Button>
            </div>
          </header>

          <UserDocsSections siteOrigin={siteOrigin} siteName={displayName} />
        </div>
      </div>
    </PublicLayout>
  )
}
