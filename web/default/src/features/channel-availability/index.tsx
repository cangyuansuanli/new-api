import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '@/stores/auth-store'
import { ROLE } from '@/lib/roles'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { SectionPageLayout } from '@/components/layout'
import { AdminMonitorPanel } from './admin-monitor-panel'
import { AvailabilityDashboard } from './availability-dashboard'

export function ChannelAvailability() {
  const { t } = useTranslation()
  const role = useAuthStore((state) => state.auth.user?.role ?? ROLE.GUEST)
  const [windowDays, setWindowDays] = useState(7)

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Channel Availability')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        {role >= ROLE.ADMIN ? (
          <Tabs defaultValue='status'>
            <TabsList className='mb-3'>
              <TabsTrigger value='status'>{t('Status')}</TabsTrigger>
              <TabsTrigger value='configuration'>
                {t('Configuration')}
              </TabsTrigger>
            </TabsList>
            <TabsContent value='status'>
              <AvailabilityDashboard
                windowDays={windowDays}
                onWindowDaysChange={setWindowDays}
              />
            </TabsContent>
            <TabsContent value='configuration'>
              <AdminMonitorPanel />
            </TabsContent>
          </Tabs>
        ) : (
          <AvailabilityDashboard
            windowDays={windowDays}
            onWindowDaysChange={setWindowDays}
          />
        )}
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
