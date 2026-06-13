/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { Code2, LineChart, ShieldCheck } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { AnimateInView } from '@/components/animate-in-view'
import { mkt } from '../../lib/marketing-theme'

export function Highlights() {
  const { t } = useTranslation()

  const items = [
    {
      icon: <Code2 className='size-5 text-blue-500' strokeWidth={1.5} />,
      title: t('Standard API integration'),
      desc: t(
        'A consistent request format for backend systems, automation workflows and internal tools.'
      ),
    },
    {
      icon: <LineChart className='size-5 text-violet-500' strokeWidth={1.5} />,
      title: t('Transparent usage'),
      desc: t(
        'Request-level logs, usage visibility and pay-as-you-go billing for product teams.'
      ),
    },
    {
      icon: (
        <ShieldCheck className='size-5 text-emerald-500' strokeWidth={1.5} />
      ),
      title: t('Production reliability'),
      desc: t(
        'Stable gateway design with operational visibility and long-term maintenance.'
      ),
    },
  ]

  return (
    <section className={cn('relative z-10 px-6 py-16 md:py-20', mkt.sectionBorder)}>
      <div className='mx-auto grid max-w-6xl gap-6 md:grid-cols-3 md:gap-8'>
        {items.map((item, i) => (
          <AnimateInView
            key={item.title}
            delay={i * 80}
            animation='fade-up'
            className={cn(mkt.card, 'p-6 md:p-7')}
          >
            <div className={cn(mkt.cardIcon, 'mb-4 size-10')}>
              {item.icon}
            </div>
            <h3 className={cn('mb-2 text-base font-semibold tracking-tight', mkt.heading)}>
              {item.title}
            </h3>
            <p className={cn('text-sm leading-relaxed', mkt.muted)}>
              {item.desc}
            </p>
          </AnimateInView>
        ))}
      </div>
    </section>
  )
}
