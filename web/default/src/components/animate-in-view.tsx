/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useRef, useEffect, useState, type ReactNode } from 'react'
import { cn } from '@/lib/utils'

interface AnimateInViewProps {
  children: ReactNode
  className?: string
  delay?: number
  threshold?: number
  animation?: 'fade-up' | 'fade-in' | 'scale-in' | 'fade-left' | 'fade-right'
  once?: boolean
  as?: 'div' | 'section' | 'li' | 'span'
}

export function AnimateInView(props: AnimateInViewProps) {
  const {
    as: Tag = 'div',
    delay = 0,
    threshold = 0.15,
    animation = 'fade-up',
    once = true,
  } = props

  const ref = useRef<HTMLDivElement>(null)
  const [active, setActive] = useState(false)

  useEffect(() => {
    const el = ref.current
    if (!el) return

    const mq = window.matchMedia('(prefers-reduced-motion: reduce)')
    if (mq.matches) {
      setActive(true)
      return
    }

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setActive(true)
          if (once) observer.disconnect()
        } else if (!once) {
          setActive(false)
        }
      },
      { threshold, rootMargin: '0px 0px -40px 0px' }
    )

    observer.observe(el)
    return () => observer.disconnect()
  }, [threshold, once])

  return (
    <Tag
      ref={ref as never}
      className={cn(
        'will-change-[transform,opacity]',
        !active && 'opacity-0',
        active && `landing-animate-${animation}`,
        props.className
      )}
      style={{ animationDelay: active && delay ? `${delay}ms` : undefined }}
    >
      {props.children}
    </Tag>
  )
}
