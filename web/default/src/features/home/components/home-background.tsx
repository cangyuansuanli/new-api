/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

const LIGHT_BASE = [
  'linear-gradient(180deg, #eef2f8 0%, #e9eef5 48%, #e5eaf2 100%)',
  'linear-gradient(118deg, rgba(6,182,212,0.09) 0%, transparent 30%, transparent 60%, rgba(99,102,241,0.07) 100%)',
  'repeating-linear-gradient(105deg, rgba(15,23,42,0.035) 0 1px, transparent 1px 46px)',
  'repeating-linear-gradient(14deg, rgba(6,182,212,0.028) 0 1px, transparent 1px 62px)',
].join(', ')

const LIGHT_GLOW = [
  'radial-gradient(circle at 16% 14%, rgba(6,182,212,0.16), transparent 280px)',
  'radial-gradient(circle at 84% 10%, rgba(99,102,241,0.11), transparent 260px)',
  'radial-gradient(circle at 48% 92%, rgba(16,185,129,0.09), transparent 320px)',
].join(', ')

const LIGHT_BEAM =
  'linear-gradient(90deg, transparent 0 8%, rgba(6,182,212,0.06) 8.1%, transparent 8.35% 64%, rgba(99,102,241,0.05) 64.1%, transparent 64.35%)'

const DARK_BASE = [
  'linear-gradient(115deg, rgba(37,232,255,0.12), transparent 28%, transparent 62%, rgba(33,255,200,0.1))',
  'repeating-linear-gradient(105deg, rgba(255,255,255,0.035) 0 1px, transparent 1px 42px)',
  'repeating-linear-gradient(15deg, rgba(33,255,200,0.026) 0 1px, transparent 1px 58px)',
  'linear-gradient(180deg, #06131c, #050b12 58%, #05070d)',
].join(', ')

const DARK_GLOW = [
  'radial-gradient(circle at 18% 16%, rgba(37,232,255,0.28), transparent 260px)',
  'radial-gradient(circle at 82% 12%, rgba(33,255,200,0.22), transparent 240px)',
  'radial-gradient(circle at 50% 85%, rgba(138,43,226,0.12), transparent 320px)',
].join(', ')

const DARK_BEAM =
  'linear-gradient(90deg, transparent 0 9%, rgba(0,240,255,0.08) 9.1%, transparent 9.35% 62%, rgba(138,43,226,0.08) 62.1%, transparent 62.35%)'

export function HomeBackground() {
  return (
    <div
      aria-hidden
      className='pointer-events-none fixed inset-0 -z-10 overflow-hidden'
    >
      {/* Light: pearl mist + subtle tech mesh */}
      <div className='absolute inset-0 dark:hidden' style={{ background: LIGHT_BASE }} />
      <div
        className='absolute inset-0 opacity-70 dark:hidden'
        style={{ background: LIGHT_GLOW }}
      />
      <div
        className='absolute inset-0 opacity-50 dark:hidden'
        style={{ background: LIGHT_BEAM }}
      />
      <div className='absolute -top-32 left-1/4 size-[520px] rounded-full bg-cyan-400/20 blur-[120px] dark:hidden' />
      <div className='absolute top-1/3 -right-24 size-[420px] rounded-full bg-violet-400/15 blur-[100px] dark:hidden' />

      {/* Dark: deep navy mesh */}
      <div
        className='absolute inset-0 hidden dark:block'
        style={{ background: DARK_BASE }}
      />
      <div
        className='absolute inset-0 hidden opacity-40 dark:block'
        style={{ background: DARK_GLOW }}
      />
      <div
        className='absolute inset-0 hidden opacity-25 dark:block'
        style={{ background: DARK_BEAM }}
      />
      <div className='absolute -top-32 left-1/4 hidden size-[520px] rounded-full bg-cyan-400/10 blur-[120px] dark:block' />
      <div className='absolute top-1/3 -right-24 hidden size-[420px] rounded-full bg-emerald-400/10 blur-[100px] dark:block' />
    </div>
  )
}
