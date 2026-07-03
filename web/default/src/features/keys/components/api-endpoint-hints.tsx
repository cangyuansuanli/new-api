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
import { useTranslation } from 'react-i18next'
import { CopyButton } from '@/components/copy-button'
import { useStatus } from '@/hooks/use-status'

const AI_ENDPOINT = 'https://ai.cangyuansuanli.cn'
const DIRECT_ENDPOINT = 'http://direct-api.cangyuansuanli.cn'

function EndpointBlock(props: {
  url: string
  audience: string
  description: string
  copyLabel: string
  recommended?: boolean
}) {
  const { t } = useTranslation()

  return (
    <li className='border-border bg-muted/20 space-y-1.5 rounded-lg border px-3 py-2.5'>
      <div className='flex flex-wrap items-center gap-2'>
        <p className='text-foreground/90 text-xs font-medium sm:text-sm'>
          {props.audience}
        </p>
        {props.recommended ? (
          <span className='bg-primary/10 text-primary rounded px-1.5 py-0.5 text-[10px] font-medium sm:text-[11px]'>
            {t('Recommended')}
          </span>
        ) : null}
      </div>
      <p className='text-muted-foreground text-[11px] leading-relaxed sm:text-xs'>
        {props.description}
      </p>
      <div className='flex flex-wrap items-center gap-x-1.5 gap-y-1 pt-0.5'>
        <code className='bg-muted rounded px-1 py-0.5 text-[11px] break-all sm:text-xs'>
          {props.url}
        </code>
        <CopyButton
          value={props.url}
          size='icon'
          variant='ghost'
          className='size-6 shrink-0'
          iconClassName='size-3'
          tooltip={props.copyLabel}
          aria-label={props.copyLabel}
        />
      </div>
    </li>
  )
}

export function ApiEndpointHints() {
  const { t } = useTranslation()
  const { status } = useStatus()

  const trafficRelayBaseUrl =
    typeof status?.traffic_relay_base_url === 'string'
      ? status.traffic_relay_base_url.trim().replace(/\/$/, '')
      : ''

  return (
    <div className='text-muted-foreground w-full space-y-3 text-xs sm:text-sm'>
      <p className='text-muted-foreground text-[11px] leading-relaxed sm:text-xs'>
        {t(
          'All endpoints below accept the same API key. Choose based on your usage scenario.'
        )}
      </p>

      <div className='space-y-2'>
        <p className='text-foreground/80 font-medium'>
          {t('High-volume API access')}
        </p>
        <p className='text-muted-foreground text-[11px] leading-relaxed sm:text-xs'>
          {t(
            'For batch jobs, long streaming sessions, and large-scale integrations.'
          )}
        </p>
        <ul className='space-y-2'>
          {trafficRelayBaseUrl ? (
            <EndpointBlock
              url={trafficRelayBaseUrl}
              audience={t('Dedicated relay node')}
              description={t(
                'High-bandwidth dedicated line for API calls. Same API key and billing as the console.'
              )}
              copyLabel={t('Copy traffic relay URL')}
              recommended
            />
          ) : null}
          <EndpointBlock
            url={DIRECT_ENDPOINT}
            audience={t('Direct high-bandwidth line')}
            description={t(
              'Alternative high-throughput endpoint for integrations with strict bandwidth needs.'
            )}
            copyLabel={t('Copy API base URL')}
            recommended={!trafficRelayBaseUrl}
          />
        </ul>
      </div>

      <div className='space-y-2'>
        <p className='text-foreground/80 font-medium'>
          {t('Alternative HTTPS access')}
        </p>
        <ul className='space-y-2'>
          <EndpointBlock
            url={AI_ENDPOINT}
            audience={t('Main site HTTPS endpoint')}
            description={t(
              'If relay or direct nodes time out or fail to connect, use this address instead.'
            )}
            copyLabel={t('Copy API base URL')}
          />
        </ul>
      </div>
    </div>
  )
}
