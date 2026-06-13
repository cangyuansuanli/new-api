/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { ArrowRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { AnimateInView } from '@/components/animate-in-view'
import { FreeImageCarousel } from '../free-image-carousel'
import { mkt } from '../../lib/marketing-theme'
import { SITE_ASSETS } from '../../lib/site-assets'

const CLI_TOOLS = [
  {
    id: 'claude-code',
    label: 'Claude Code',
    title: 'Claude Code',
    subtitle: 'Anthropic',
    image: SITE_ASSETS.tools.claudeCode,
    href: 'https://docs.anthropic.com/en/docs/claude-code',
  },
  {
    id: 'codex-cli',
    label: 'Codex',
    title: 'Codex CLI',
    subtitle: 'OpenAI /v1',
    image: SITE_ASSETS.tools.codexCli,
    href: 'https://github.com/openai/codex',
  },
  {
    id: 'gemini-cli',
    label: 'Gemini CLI',
    title: 'Gemini CLI',
    subtitle: 'Google',
    image: SITE_ASSETS.tools.geminiCli,
    href: 'https://github.com/google-gemini/gemini-cli',
  },
  {
    id: 'image-api',
    label: 'Image API',
    title: 'Image API',
    subtitle: 'OpenAI compatible',
    image: SITE_ASSETS.tools.imageApi,
    href: 'https://ai.cangyuansuanli.cn/pricing',
  },
] as const

export function ProductTools() {
  const { t } = useTranslation()

  return (
    <section
      id='tools'
      className={cn('relative z-10 px-6 py-20 md:py-28', mkt.sectionBorder)}
    >
      <div className='mx-auto max-w-6xl'>
        <AnimateInView className='mb-10 text-center md:mb-12'>
          <p className={cn('mb-3 text-xs font-medium tracking-widest uppercase', mkt.eyebrow)}>
            {t('Tool Entry')}
          </p>
          <h2 className={cn('text-2xl font-bold tracking-tight md:text-3xl', mkt.heading)}>
            {t('Tools and creative workflows')}
          </h2>
          <p className={cn('mx-auto mt-3 max-w-2xl text-sm leading-relaxed md:text-base', mkt.muted)}>
            {t(
              'Compatible with OpenAI API format — one gateway for CLI tools and visual creation.'
            )}
          </p>
        </AnimateInView>

        <AnimateInView animation='fade-up'>
          <FreeImageCarousel />
        </AnimateInView>

        <div className='mt-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-4'>
          {CLI_TOOLS.map((tool, i) => (
            <AnimateInView
              key={tool.id}
              delay={i * 60}
              animation='fade-up'
              className={cn(
                'group relative aspect-[3/2] overflow-hidden rounded-2xl border shadow-sm transition-all duration-300 hover:border-cyan-500/35 hover:shadow-[0_0_40px_-12px_rgba(6,182,212,0.25)] dark:hover:border-cyan-400/30 dark:hover:shadow-[0_0_40px_-12px_rgba(37,232,255,0.35)]',
                mkt.mediaCard
              )}
            >
              <a
                href={tool.href}
                target='_blank'
                rel='noopener noreferrer'
                className='block size-full'
              >
                <img
                  src={tool.image}
                  alt={tool.title}
                  loading='lazy'
                  className='absolute inset-0 size-full object-cover object-center transition-transform duration-500 group-hover:scale-105'
                />
                <div className={cn('absolute inset-0', mkt.mediaOverlay)} />
                <div className='relative flex h-full flex-col justify-end p-5'>
                  <span className={cn('mb-2 inline-flex w-fit rounded-full px-2.5 py-0.5 text-[10px] font-semibold tracking-wide uppercase', mkt.badgeOnImage)}>
                    {tool.label}
                  </span>
                  <h3 className='text-base font-semibold text-white'>
                    {tool.title}
                  </h3>
                  <p className='mt-1 text-sm text-white/75'>{tool.subtitle}</p>
                  <span className='mt-3 inline-flex items-center gap-1 text-xs font-medium text-cyan-200 opacity-0 transition-opacity group-hover:opacity-100'>
                    {t('Learn more')}
                    <ArrowRight className='size-3.5' />
                  </span>
                </div>
              </a>
            </AnimateInView>
          ))}
        </div>
      </div>
    </section>
  )
}
