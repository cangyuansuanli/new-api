/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { AnimateInView } from '@/components/animate-in-view'
import { getLobeIcon } from '@/lib/lobe-icon'
import { mkt } from '../../lib/marketing-theme'
import { PROVIDER_ICONS } from '../../constants'

export function ProviderLogos() {
  const { t } = useTranslation()

  return (
    <section className={cn('relative z-10 px-6 py-14 md:py-20', mkt.sectionBorder)}>
      <div className='mx-auto max-w-6xl'>
        <AnimateInView className='mb-8 text-center md:mb-10'>
          <p className={cn('text-base font-light md:text-lg', mkt.body)}>
            {t('Supports a wide range of model providers')}
          </p>
        </AnimateInView>
        <AnimateInView
          animation='fade-up'
          className='flex flex-wrap items-center justify-center gap-4 sm:gap-5 md:gap-6 lg:gap-8'
        >
          {PROVIDER_ICONS.map((iconName) => (
            <div
              key={iconName}
              className='flex size-10 items-center justify-center sm:size-11 md:size-12'
            >
              {getLobeIcon(iconName, 40)}
            </div>
          ))}
          <div className={cn('flex size-10 items-center justify-center text-xl font-bold sm:size-11 md:size-12 md:text-2xl', mkt.muted)}>
            30+
          </div>
        </AnimateInView>
      </div>
    </section>
  )
}
