/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { KeyRound, Link2, UserPlus, BarChart3 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { AnimateInView } from '@/components/animate-in-view'
import { mkt } from '../../lib/marketing-theme'

export function HowItWorks() {
  const { t } = useTranslation()

  const steps = [
    {
      num: '01',
      title: t('Create an account'),
      desc: t('Sign up and access your dashboard in minutes.'),
      icon: <UserPlus className='size-6' strokeWidth={1.5} />,
    },
    {
      num: '02',
      title: t('Generate an API key'),
      desc: t('Create an API key for secure authentication.'),
      icon: <KeyRound className='size-6' strokeWidth={1.5} />,
    },
    {
      num: '03',
      title: t('Set the unified endpoint'),
      desc: t('Use a single endpoint for all model requests.'),
      icon: <Link2 className='size-6' strokeWidth={1.5} />,
    },
    {
      num: '04',
      title: t('Route requests and review logs'),
      desc: t('Monitor usage, trace requests and optimize continuously.'),
      icon: <BarChart3 className='size-6' strokeWidth={1.5} />,
    },
  ]

  return (
    <section
      id='integration'
      className={cn('relative z-10 px-6 py-24 md:py-32', mkt.sectionBorder)}
    >
      <div className='mx-auto max-w-6xl'>
        <AnimateInView className='mb-16 text-center md:mb-20'>
          <p className={cn('mb-3 text-xs font-medium tracking-widest uppercase', mkt.eyebrow)}>
            {t('Quick integration')}
          </p>
          <h2 className={cn('text-2xl font-bold tracking-tight md:text-3xl', mkt.heading)}>
            {t('Quick integration without changing your workflow')}
          </h2>
        </AnimateInView>

        <div className='grid gap-8 sm:grid-cols-2 lg:grid-cols-4 lg:gap-6'>
          {steps.map((step, i) => (
            <AnimateInView
              key={step.num}
              delay={i * 100}
              animation='fade-up'
              className='relative flex flex-col items-start'
            >
              <div className='relative mb-5'>
                <div className={cn(mkt.cardIcon, 'size-14 text-cyan-600 dark:text-cyan-300')}>
                  {step.icon}
                </div>
                <span className={cn('absolute -top-2 -right-2 text-[10px] font-bold tracking-widest', mkt.eyebrow)}>
                  {step.num}
                </span>
              </div>
              <h3 className={cn('mb-2 text-base font-semibold', mkt.heading)}>{step.title}</h3>
              <p className={cn('text-sm leading-relaxed', mkt.muted)}>
                {step.desc}
              </p>
            </AnimateInView>
          ))}
        </div>
      </div>
    </section>
  )
}
