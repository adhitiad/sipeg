import { createFileRoute } from '@tanstack/react-router'
import SloMetricsPage from '@/components/SloMetricsPage'

export const Route = createFileRoute('/_app/admin/slo')({
  component: SloMetricsPage,
})
