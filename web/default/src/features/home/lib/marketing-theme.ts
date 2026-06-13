/*
Copyright (C) 2023-2026 QuantumNous

Marketing landing tokens — light: pearl mist (not pure white); dark: deep navy.
*/

/** Section content tokens */
export const mkt = {
  page: 'text-slate-900 dark:text-[#eef8ff]',
  heading: 'text-slate-900 dark:text-white',
  body: 'text-slate-600 dark:text-[#c5dce8]',
  muted: 'text-slate-500 dark:text-[#98b4c1]',
  eyebrow: 'text-cyan-700 dark:text-cyan-300/80',
  iconAccent: 'text-cyan-600 dark:text-cyan-300',
  sectionBorder: 'border-t border-slate-200/80 dark:border-white/10',
  card:
    'rounded-2xl border border-slate-200/70 bg-white/75 shadow-[0_1px_0_rgba(15,23,42,0.04),0_8px_32px_-12px_rgba(15,23,42,0.08)] backdrop-blur-md dark:border-white/10 dark:bg-white/[0.04] dark:shadow-none',
  cardIcon:
    'flex items-center justify-center rounded-xl border border-slate-200/60 bg-slate-50/90 dark:border-white/10 dark:bg-white/5',
  trustBadge:
    'flex size-7 items-center justify-center rounded-lg border border-slate-200/70 bg-white/90 shadow-sm dark:border-white/10 dark:bg-white/5 dark:shadow-none',
  btnGhost:
    'border-slate-300/80 bg-white/70 text-slate-700 shadow-sm hover:border-slate-400/80 hover:bg-white hover:text-slate-900 dark:border-white/15 dark:bg-white/5 dark:text-white/80 dark:shadow-none dark:hover:bg-white/10 dark:hover:text-white',
  bentoWrap:
    'gap-px overflow-hidden rounded-2xl border border-slate-200/70 bg-slate-200/40 dark:border-white/10 dark:bg-white/[0.06]',
  bentoCell: 'bg-white/92 p-6 backdrop-blur-sm dark:bg-[#08111c]/90',
  bentoNum: 'font-mono text-xs text-slate-400 dark:text-cyan-300/50',
  footer: 'border-t border-slate-200/80 text-slate-500 dark:border-white/10 dark:text-[#98b4c1]',
  ctaGlow: 'opacity-100',
  statsBand:
    'border-y border-slate-200/80 bg-white/55 backdrop-blur-sm dark:border-white/10 dark:bg-white/[0.03]',
  mediaCard:
    'border-slate-200/80 shadow-[0_16px_48px_-20px_rgba(15,23,42,0.18)] dark:border-white/10 dark:shadow-[0_24px_80px_-24px_rgba(0,0,0,0.65)]',
  mediaOverlay:
    'bg-gradient-to-t from-slate-900/88 via-slate-900/28 to-transparent dark:from-[#050b12]/95 dark:via-[#050b12]/25',
  carouselBtn:
    'border-slate-200/90 bg-white/95 text-slate-700 shadow-sm hover:bg-white dark:border-white/15 dark:bg-[#07121bcc] dark:text-white dark:shadow-none dark:hover:bg-[#0a1824ee]',
  carouselDot: 'bg-slate-300/90 hover:bg-slate-400 dark:bg-white/25 dark:hover:bg-white/40',
  carouselDotActive: 'bg-cyan-600 dark:bg-cyan-400',
  badgeOnImage:
    'border-white/25 bg-black/25 text-white backdrop-blur-sm',
} as const

/** Layout / chrome tokens */
export const mktLayout = {
  shell: 'bg-[#e9eef5] dark:bg-[#050b12]',
  headerScrolled:
    'border border-slate-200/75 bg-white/80 shadow-[0_2px_20px_-8px_rgba(15,23,42,0.12)] backdrop-blur-xl dark:border-white/10 dark:bg-[#07121c]/80 dark:shadow-[0_2px_24px_-8px_rgba(0,0,0,0.5)]',
  siteName: 'text-slate-900 dark:text-white',
  navLink:
    'text-slate-600 hover:text-slate-900 dark:text-[#98b4c1] dark:hover:text-white',
  navLinkActive: 'text-slate-900 dark:text-cyan-200',
  mobileOverlay:
    'bg-[#e9eef5]/98 backdrop-blur-xl dark:bg-[#050b12]/98',
} as const
