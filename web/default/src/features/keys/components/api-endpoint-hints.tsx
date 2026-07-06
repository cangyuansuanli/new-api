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
import { isEuDeployment } from '@/i18n/region'

const AI_ENDPOINT = 'https://ai.cangyuansuanli.cn'
const DIRECT_ENDPOINT = 'http://direct-api.cangyuansuanli.cn'
const DEFAULT_RELAY_ENDPOINT = 'https://vip-api.cangyuansuanli.cn'

function EndpointBlock(props: {
  url: string
  audience: string
  description: string
  whenToUse: string
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
      <p className='text-muted-foreground/80 text-[11px] leading-relaxed sm:text-xs'>
        <span className='text-foreground/70 font-medium'>
          {t('Choose this when')}
        </span>
        {props.whenToUse}
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
  const euDeployment = isEuDeployment()

  const trafficRelayBaseUrl = (() => {
    const fromStatus =
      typeof status?.traffic_relay_base_url === 'string'
        ? status.traffic_relay_base_url.trim().replace(/\/$/, '')
        : ''
    if (fromStatus) return fromStatus
    return DEFAULT_RELAY_ENDPOINT
  })()

  const euPrimaryEndpoint = (() => {
    const fromStatus =
      typeof status?.server_address === 'string'
        ? status.server_address.trim().replace(/\/$/, '')
        : ''
    if (fromStatus) return fromStatus
    if (trafficRelayBaseUrl && euDeployment) return trafficRelayBaseUrl
    return 'https://eu-ai.cangyuansuanli.cn'
  })()

  if (euDeployment) {
    return (
      <div className='text-muted-foreground w-full space-y-3 text-xs sm:text-sm'>
        <p className='text-muted-foreground text-[11px] leading-relaxed sm:text-xs'>
          {t(
            'This European deployment accepts the same API key format. Use the endpoint below as your base URL.'
          )}
        </p>

        <div className='space-y-2'>
          <p className='text-foreground/80 font-medium'>
            {t('European API access')}
          </p>
          <ul className='space-y-2'>
            <EndpointBlock
              url={euPrimaryEndpoint}
              audience={t('European primary endpoint')}
              description={t(
                'HTTPS entry point hosted in Europe with low latency for EU customers and integrations.'
              )}
              whenToUse={t(
                'You deploy or integrate from Europe — this is the recommended base URL for this site.'
              )}
              copyLabel={t('Copy European API URL')}
              recommended
            />
          </ul>
        </div>
      </div>
    )
  }

  return (
    <div className='text-muted-foreground w-full space-y-3 text-xs sm:text-sm'>
      <p className='text-muted-foreground text-[11px] leading-relaxed sm:text-xs'>
        {t(
          'All endpoints below accept the same API key. Pick the one that matches where you deploy and how much traffic you send — details for each option follow.'
        )}
      </p>

      <div className='space-y-2'>
        <p className='text-foreground/80 font-medium'>
          {t('Overseas API access')}
        </p>
        <p className='text-muted-foreground text-[11px] leading-relaxed sm:text-xs'>
          {t(
            'For servers and integrations outside mainland China. Prefer the acceleration node for sustained high-volume traffic; keep origin fallback as a backup.'
          )}
        </p>
        <ul className='space-y-2'>
          <EndpointBlock
            url={trafficRelayBaseUrl}
            audience={t('Dedicated acceleration node')}
            description={t(
              'Overseas relay node with generous bandwidth over HTTPS. Suited to batch jobs, long streaming sessions, and large-scale API integrations.'
            )}
            whenToUse={t(
              'Your workloads run overseas and send sustained or heavy traffic — this is the default choice for most high-volume integrations.'
            )}
            copyLabel={t('Copy traffic relay URL')}
            recommended
          />
          <EndpointBlock
            url={DIRECT_ENDPOINT}
            audience={t('Origin fallback access')}
            description={t(
              'Direct connection to origin NewAPI with limited bandwidth — a fallback option when other overseas endpoints are unavailable.'
            )}
            whenToUse={t(
              'The acceleration node cannot be reached from your network and you need a backup path — not for primary high-volume traffic.'
            )}
            copyLabel={t('Copy API base URL')}
          />
        </ul>
      </div>

      <div className='space-y-2'>
        <p className='text-foreground/80 font-medium'>
          {t('Mainland network access')}
        </p>
        <p className='text-muted-foreground text-[11px] leading-relaxed sm:text-xs'>
          {t(
            'For customers and integrations running inside mainland China.'
          )}
        </p>
        <ul className='space-y-2'>
          <EndpointBlock
            url={AI_ENDPOINT}
            audience={t('Mainland direct access')}
            description={t(
              'Main site HTTPS entry via CDN and edge protection. Stable for everyday API calls in the domestic network environment.'
            )}
            whenToUse={t(
              'Your clients or servers are in mainland China — choose this for routine integrations and general API usage.'
            )}
            copyLabel={t('Copy API base URL')}
          />
        </ul>
      </div>
    </div>
  )
}
