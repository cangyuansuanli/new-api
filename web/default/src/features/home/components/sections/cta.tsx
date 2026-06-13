/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { Link } from '@tanstack/react-router'
import { ArrowRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { AnimateInView } from '@/components/animate-in-view'
import { cn } from '@/lib/utils'
import { mkt } from '../../lib/marketing-theme'

interface CTAProps {
  className?: string
  isAuthenticated?: boolean
}

export function CTA(props: CTAProps) {
  const { t } = useTranslation()

  if (props.isAuthenticated) {
    return null
  }

  return (
    <section className='relative z-10 overflow-hidden px-6 py-24 md:py-32'>
      <div
        aria-hidden
        className='absolute inset-0 -z-10 opacity-80 dark:hidden'
        style={{
          background:
            'radial-gradient(ellipse 50% 50% at 30% 50%, rgba(6,182,212,0.14) 0%, transparent 70%), radial-gradient(ellipse 40% 40% at 70% 40%, rgba(99,102,241,0.1) 0%, transparent 70%)',
        }}
      />
      <div
        aria-hidden
        className='absolute inset-0 -z-10 hidden opacity-60 dark:block'
        style={{
          background:
            'radial-gradient(ellipse 50% 50% at 30% 50%, rgba(37,232,255,0.12) 0%, transparent 70%), radial-gradient(ellipse 40% 40% at 70% 40%, rgba(33,255,200,0.1) 0%, transparent 70%)',
        }}
      />

      <AnimateInView
        className='mx-auto max-w-2xl text-center'
        animation='scale-in'
      >
        <h2 className={cn('text-2xl leading-tight font-bold tracking-tight md:text-4xl', mkt.heading)}>
          {t('Ready to simplify')}
          <br />
          <span className='bg-gradient-to-r from-cyan-600 via-emerald-600 to-violet-600 bg-clip-text text-transparent dark:from-cyan-300 dark:via-emerald-300 dark:to-violet-300'>
            {t('your AI integration?')}
          </span>
        </h2>
        <p className={cn('mx-auto mt-5 max-w-md text-sm leading-relaxed md:text-base', mkt.muted)}>
          {t(
            'Deploy your own gateway and start routing requests through your configured upstream services.'
          )}
        </p>
        <div className='mt-8 flex items-center justify-center gap-3'>
          <Button className='group rounded-lg' render={<Link to='/sign-up' />}>
            {t('Get Started')}
            <ArrowRight className='ml-1 size-3.5 transition-transform duration-200 group-hover:translate-x-0.5' />
          </Button>
          <Button
            variant='outline'
            className={cn('rounded-lg', mkt.btnGhost)}
            render={<Link to='/pricing' />}
          >
            {t('View Pricing')}
          </Button>
        </div>
      </AnimateInView>
    </section>
  )
}
