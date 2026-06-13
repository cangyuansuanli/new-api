/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useCallback, useEffect, useState } from 'react'
import { ArrowRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '@/stores/auth-store'
import { cn } from '@/lib/utils'
import {
  Carousel,
  CarouselContent,
  CarouselItem,
  CarouselNext,
  CarouselPrevious,
  type CarouselApi,
} from '@/components/ui/carousel'
import { DEFAULT_CANVAS_BASE_URL } from '@/features/canvas/lib/canvas-config'
import { useCanvasEntryUrl } from '@/features/canvas/hooks/use-canvas-entry-url'
import { mkt } from '../lib/marketing-theme'
import { INSPIRATION_SLIDES } from '../lib/site-assets'

export function FreeImageCarousel() {
  const { t } = useTranslation()
  const isAuthenticated = !!useAuthStore((state) => state.auth.user)
  const canvasUrl = useCanvasEntryUrl(DEFAULT_CANVAS_BASE_URL, {
    withTrust: isAuthenticated,
  })
  const [api, setApi] = useState<CarouselApi>()
  const [active, setActive] = useState(0)

  const onSelect = useCallback((carouselApi: CarouselApi) => {
    if (!carouselApi) return
    setActive(carouselApi.selectedScrollSnap())
  }, [])

  useEffect(() => {
    if (!api) return
    onSelect(api)
    api.on('select', onSelect)
    return () => {
      api.off('select', onSelect)
    }
  }, [api, onSelect])

  useEffect(() => {
    if (!api) return
    const timer = window.setInterval(() => {
      if (api.canScrollNext()) api.scrollNext()
      else api.scrollTo(0)
    }, 5000)
    return () => window.clearInterval(timer)
  }, [api])

  return (
    <div className='relative'>
      <Carousel
        setApi={setApi}
        opts={{ align: 'start', loop: true }}
        className='w-full'
      >
        <CarouselContent className='-ml-3 md:-ml-4'>
          {INSPIRATION_SLIDES.map((slide) => (
            <CarouselItem
              key={slide.id}
              className='basis-full pl-3 md:basis-[88%] md:pl-4 lg:basis-[78%]'
            >
              <a
                href={canvasUrl}
                target='_blank'
                rel='noopener noreferrer'
                className={cn(
                  'group relative block aspect-[3/4] overflow-hidden rounded-2xl sm:aspect-[4/5] md:max-h-[min(72vh,640px)]',
                  mkt.mediaCard
                )}
              >
                <img
                  src={slide.image}
                  alt={slide.title}
                  loading='lazy'
                  className='absolute inset-0 size-full object-cover object-center transition-transform duration-700 group-hover:scale-[1.03]'
                />
                <div className={cn('absolute inset-0', mkt.mediaOverlay)} />
                <div className='absolute inset-x-0 bottom-0 flex flex-col gap-3 p-5 sm:flex-row sm:items-end sm:justify-between sm:p-8'>
                  <div>
                    <span className='mb-2 inline-flex rounded-full border border-cyan-400/40 bg-cyan-500/15 px-2.5 py-0.5 text-[10px] font-semibold tracking-wide text-cyan-100 uppercase'>
                      {t('Free trial')}
                    </span>
                    <p className='text-[11px] font-medium tracking-[0.2em] text-cyan-200/90 uppercase'>
                      {t('Cangyuan Image to Video')}
                    </p>
                    <h3 className='mt-1 text-xl font-semibold text-white sm:text-2xl'>
                      {slide.title}
                    </h3>
                    <p className='mt-1 text-sm text-white/75'>{slide.tags}</p>
                  </div>
                  <span className='inline-flex w-fit items-center gap-1.5 rounded-lg bg-white px-4 py-2.5 text-sm font-semibold text-slate-900 transition-colors group-hover:bg-cyan-50'>
                    {t('Try for free')}
                    <ArrowRight className='size-4' />
                  </span>
                </div>
              </a>
            </CarouselItem>
          ))}
        </CarouselContent>
        <CarouselPrevious className={cn('left-3', mkt.carouselBtn)} />
        <CarouselNext className={cn('right-3', mkt.carouselBtn)} />
      </Carousel>

      <div className='mt-4 flex items-center justify-center gap-2'>
        {INSPIRATION_SLIDES.map((slide, index) => (
          <button
            key={slide.id}
            type='button'
            aria-label={`${slide.title} ${index + 1}`}
            onClick={() => api?.scrollTo(index)}
            className={cn(
              'h-1.5 rounded-full transition-all',
              active === index
                ? cn('w-8', mkt.carouselDotActive)
                : cn('w-2', mkt.carouselDot)
            )}
          />
        ))}
      </div>
    </div>
  )
}
