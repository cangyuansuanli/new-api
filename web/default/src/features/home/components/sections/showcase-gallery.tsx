/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { ArrowRight } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { AnimateInView } from '@/components/animate-in-view'
import { DEFAULT_CANVAS_BASE_URL } from '@/features/canvas/lib/canvas-config'
import { useCanvasKeyPicker } from '@/features/canvas/hooks/use-canvas-key-picker'
import { mkt } from '../../lib/marketing-theme'
import { ShowcaseMarqueeRow } from '../showcase-marquee-row'
import {
  HOME_SHOWCASE_ROW_A,
  HOME_SHOWCASE_ROW_B,
  showcaseMarqueeDuration,
} from '../../hooks/use-showcase-assets'

export function ShowcaseGallery() {
  const { t } = useTranslation()
  const { requestOpen, dialog } = useCanvasKeyPicker(DEFAULT_CANVAS_BASE_URL)

  const durationA = useMemo(
    () => showcaseMarqueeDuration(HOME_SHOWCASE_ROW_A.length, 50),
    []
  )
  const durationB = useMemo(
    () => showcaseMarqueeDuration(HOME_SHOWCASE_ROW_B.length, 55),
    []
  )

  return (
    <section
      id='showcase'
      className={cn('relative z-10 py-12 md:py-16', mkt.sectionBorder)}
    >
      <div className='mx-auto max-w-6xl px-6'>
        <AnimateInView className='mb-8 flex flex-col items-center text-center md:mb-10'>
          <p className={cn('mb-3 text-xs font-medium tracking-widest uppercase', mkt.eyebrow)}>
            {t('Creative showcase')}
          </p>
          <h2 className={cn('text-xl font-bold tracking-tight md:text-2xl', mkt.heading)}>
            {t('Visual works from the canvas asset library')}
          </h2>
          <p className={cn('mx-auto mt-3 max-w-2xl text-sm leading-relaxed', mkt.muted)}>
            {t('Home showcase CDN subtitle')}
          </p>
        </AnimateInView>
      </div>

      <AnimateInView animation='fade-up' className='space-y-3 sm:space-y-4'>
        <ShowcaseMarqueeRow
          assets={HOME_SHOWCASE_ROW_A}
          direction='left'
          durationSec={durationA}
          ready
          onSelect={() => requestOpen()}
        />
        <ShowcaseMarqueeRow
          assets={HOME_SHOWCASE_ROW_B}
          direction='right'
          durationSec={durationB}
          ready
          onSelect={() => requestOpen()}
        />
      </AnimateInView>

      <div className='mx-auto mt-8 flex max-w-6xl justify-center px-6'>
        <Button
          variant='outline'
          className={cn('group rounded-lg', mkt.btnGhost)}
          onClick={() => requestOpen()}
        >
          {t('Open Infinite Canvas')}
          <ArrowRight className='ml-1.5 size-3.5 transition-transform duration-200 group-hover:translate-x-0.5' />
        </Button>
      </div>

      {dialog}
    </section>
  )
}
