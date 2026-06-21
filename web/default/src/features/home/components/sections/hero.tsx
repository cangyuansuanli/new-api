/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { Link } from '@tanstack/react-router'
import { cn } from '@/lib/utils'
import { ArrowRight, BookOpen, KeyRound, LineChart, Wallet } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useStatus } from '@/hooks/use-status'
import { Button } from '@/components/ui/button'
import { mkt } from '../../lib/marketing-theme'
import { HeroTerminalDemo } from '../hero-terminal-demo'

interface HeroProps {
  className?: string
  isAuthenticated?: boolean
}

export function Hero(props: HeroProps) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const docsUrl =
    (status?.docs_link as string | undefined) || '/docs'

  const renderDocsButton = () => {
    const isExternal = docsUrl.startsWith('http')
    const className = cn(
      'group inline-flex h-11 items-center gap-1.5 rounded-lg px-5 text-sm font-medium',
      mkt.btnGhost
    )
    if (isExternal) {
      return (
        <Button
          variant='outline'
          className={className}
          render={
            <a href={docsUrl} target='_blank' rel='noopener noreferrer' />
          }
        >
          <BookOpen className={cn('size-4 transition-colors duration-200 group-hover:text-cyan-600 dark:group-hover:text-cyan-200', mkt.iconAccent)} />
          <span>{t('Docs')}</span>
        </Button>
      )
    }
    return (
      <Button variant='outline' className={className} render={<Link to={docsUrl} />}>
        <BookOpen className='size-4 text-cyan-300 transition-colors duration-200 group-hover:text-cyan-200' />
        <span>{t('Docs')}</span>
      </Button>
    )
  }

  const trustBadges = [
    { icon: KeyRound, label: t('One API key') },
    { icon: LineChart, label: t('Request logs') },
    { icon: Wallet, label: t('Pay as you go') },
  ]

  return (
    <section className='relative z-10 overflow-hidden px-6 pt-24 pb-12 md:pt-32 md:pb-16 lg:pt-36 lg:pb-20'>
      <div className='mx-auto grid max-w-6xl grid-cols-1 items-center gap-12 lg:grid-cols-12 lg:gap-10'>
        <div className='flex flex-col items-start text-left lg:col-span-6'>
          <p
            className={cn('landing-animate-fade-up mb-4 text-[11px] font-semibold tracking-[0.2em] uppercase', mkt.eyebrow)}
            style={{ animationDelay: '0ms' }}
          >
            {t('Unified Multi-Model API Gateway')}
          </p>

          <h1
            className={cn('landing-animate-fade-up text-[clamp(2.25rem,4.8vw,3.5rem)] leading-[1.1] font-bold tracking-tight', mkt.heading)}
            style={{ animationDelay: '60ms' }}
          >
            {t('One API for production AI')}
          </h1>

          <p
            className={cn('landing-animate-fade-up mt-5 max-w-xl text-base leading-relaxed md:text-lg', mkt.body)}
            style={{ animationDelay: '120ms' }}
          >
            {t(
              'Connect enterprise systems and product backends to multiple model families through one stable API gateway.'
            )}
          </p>

          <div
            className='landing-animate-fade-up mt-8 flex flex-wrap items-center gap-3'
            style={{ animationDelay: '180ms' }}
          >
            {props.isAuthenticated ? (
              <Button
                className='group h-11 rounded-lg px-5 text-sm font-medium'
                render={<Link to='/dashboard' />}
              >
                {t('Go to Dashboard')}
                <ArrowRight className='ml-1.5 size-4 transition-transform duration-200 group-hover:translate-x-0.5' />
              </Button>
            ) : (
              <Button
                className='group h-11 rounded-lg px-5 text-sm font-medium'
                render={<Link to='/sign-up' />}
              >
                {t('Get API Key')}
                <ArrowRight className='ml-1.5 size-4 transition-transform duration-200 group-hover:translate-x-0.5' />
              </Button>
            )}
            {renderDocsButton()}
          </div>

          <div
            className='landing-animate-fade-up mt-8 flex flex-wrap gap-4'
            style={{ animationDelay: '240ms' }}
          >
            {trustBadges.map(({ icon: Icon, label }) => (
              <div
                key={label}
                className={cn('inline-flex items-center gap-2 text-xs font-medium md:text-sm', mkt.body)}
              >
                <span className={mkt.trustBadge}>
                  <Icon className={cn('size-3.5', mkt.iconAccent)} strokeWidth={1.75} />
                </span>
                {label}
              </div>
            ))}
          </div>
        </div>

        <div
          className='landing-animate-fade-up flex w-full justify-center lg:col-span-6'
          style={{ animationDelay: '320ms' }}
        >
          <HeroTerminalDemo className='w-full max-w-lg lg:max-w-none' />
        </div>
      </div>
    </section>
  )
}
