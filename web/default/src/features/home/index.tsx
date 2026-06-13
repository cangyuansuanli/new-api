/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '@/stores/auth-store'
import { Markdown } from '@/components/ui/markdown'
import { PublicLayout } from '@/components/layout'
import { Footer } from '@/components/layout/components/footer'
import { mkt } from './lib/marketing-theme'
import { SITE_BRAND } from './lib/site-assets'
import { HomeBackground } from './components/home-background'
import {
  CTA,
  Features,
  Hero,
  Highlights,
  HowItWorks,
  ProductTools,
  ProviderLogos,
  Stats,
} from './components'
import { useHomePageContent } from './hooks'

const marketingLogo = (
  <img
    src={SITE_BRAND.logo}
    alt={SITE_BRAND.name}
    className='size-full rounded-lg object-contain'
  />
)

const homeLayoutProps = {
  showMainContainer: false as const,
  variant: 'marketing' as const,
  siteName: SITE_BRAND.name,
  logo: marketingLogo,
}

export function Home() {
  const { t } = useTranslation()
  const { auth } = useAuthStore()
  const isAuthenticated = !!auth.user
  const { content, isLoaded, isUrl } = useHomePageContent()

  if (!isLoaded) {
    return (
      <PublicLayout {...homeLayoutProps}>
        <main className='flex min-h-screen items-center justify-center'>
          <div className={mkt.muted}>{t('Loading...')}</div>
        </main>
      </PublicLayout>
    )
  }

  if (content) {
    return (
      <PublicLayout {...homeLayoutProps}>
        <main className='overflow-x-hidden'>
          {isUrl ? (
            <iframe
              src={content}
              className='h-screen w-full border-none'
              title={t('Custom Home Page')}
            />
          ) : (
            <div className='container mx-auto py-8'>
              <Markdown className='custom-home-content'>{content}</Markdown>
            </div>
          )}
        </main>
      </PublicLayout>
    )
  }

  return (
    <PublicLayout {...homeLayoutProps}>
      <div className={`marketing-home relative min-h-svh ${mkt.page}`}>
        <HomeBackground />
        <Hero isAuthenticated={isAuthenticated} />
        <ProviderLogos />
        <Highlights />
        <ProductTools />
        <Stats />
        <Features />
        <HowItWorks />
        <CTA isAuthenticated={isAuthenticated} />
        <Footer className={mkt.footer} />
      </div>
    </PublicLayout>
  )
}
