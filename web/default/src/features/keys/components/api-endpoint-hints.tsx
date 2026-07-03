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
const DIRECT_ENDPOINT = 'https://direct-api.cangyuansuanli.cn'

function EndpointRow(props: {
  url: string
  label: string
  copyLabel: string
}) {
  return (
    <li className='flex flex-wrap items-center gap-x-1.5 gap-y-1'>
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
      <span className='text-muted-foreground w-full sm:w-auto'>{props.label}</span>
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
      <div className='space-y-1'>
        <p className='text-foreground/80 font-medium'>
          {t('API base URL (OpenAI-compatible)')}
        </p>
        <ul className='space-y-2'>
          <EndpointRow
            url={AI_ENDPOINT}
            label={t('General customers (recommended)')}
            copyLabel={t('Copy API base URL')}
          />
          <EndpointRow
            url={DIRECT_ENDPOINT}
            label={t('Origin direct high-volume endpoint')}
            copyLabel={t('Copy API base URL')}
          />
        </ul>
      </div>

      {trafficRelayBaseUrl ? (
        <div className='border-border bg-muted/30 space-y-1 rounded-lg border px-3 py-2.5'>
          <p className='text-foreground/80 font-medium'>
            {t('Traffic relay node entry')}
          </p>
          <p className='text-muted-foreground text-[11px] sm:text-xs'>
            {t(
              'High-bandwidth relay node; uses the same API keys as the main site.'
            )}
          </p>
          <ul className='space-y-2 pt-1'>
            <EndpointRow
              url={trafficRelayBaseUrl}
              label={t('Recommended for large API traffic')}
              copyLabel={t('Copy traffic relay URL')}
            />
          </ul>
        </div>
      ) : null}
    </div>
  )
}
