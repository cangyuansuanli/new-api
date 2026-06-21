/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

type DocsSectionProps = {
  id: string
  title: string
  description?: string
  children: ReactNode
  className?: string
}

export function DocsSection(props: DocsSectionProps) {
  return (
    <section
      id={props.id}
      className={cn('scroll-mt-24 border-t border-border/40 pt-12 first:border-t-0 first:pt-0', props.className)}
    >
      <div className='mb-6 space-y-2'>
        <h2 className='text-2xl font-semibold tracking-tight'>{props.title}</h2>
        {props.description ? (
          <p className='text-muted-foreground text-base leading-relaxed'>{props.description}</p>
        ) : null}
      </div>
      <div className='text-foreground/90 space-y-4 text-[15px] leading-7'>{props.children}</div>
    </section>
  )
}
