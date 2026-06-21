/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { Copy } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

type CopyBlockProps = {
  label: string
  value: string
  className?: string
}

export function CopyBlock(props: CopyBlockProps) {
  const { t } = useTranslation()

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(props.value)
      toast.success(t('userDocs.copySuccess'))
    } catch {
      toast.error(t('userDocs.copyFailed'))
    }
  }

  return (
    <div
      className={cn(
        'border-border/60 bg-muted/30 flex flex-col gap-3 rounded-xl border p-4 sm:flex-row sm:items-center sm:justify-between',
        props.className
      )}
    >
      <div className='min-w-0 flex-1'>
        <p className='text-muted-foreground mb-1 text-xs font-medium tracking-wide uppercase'>
          {props.label}
        </p>
        <pre className='text-foreground overflow-x-auto font-mono text-sm whitespace-pre-wrap break-all'>
          {props.value}
        </pre>
      </div>
      <Button type='button' variant='outline' size='sm' className='shrink-0' onClick={handleCopy}>
        <Copy className='size-4' />
        {t('userDocs.copy')}
      </Button>
    </div>
  )
}
