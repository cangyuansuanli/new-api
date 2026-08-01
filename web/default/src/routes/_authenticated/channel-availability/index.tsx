import { createFileRoute } from '@tanstack/react-router'
import { ChannelAvailability } from '@/features/channel-availability'

export const Route = createFileRoute('/_authenticated/channel-availability/')({
  component: ChannelAvailability,
})
