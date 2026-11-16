import { createFileRoute } from '@tanstack/react-router'
import ThrPage from '@/components/ThrPage'

export const Route = createFileRoute('/_app/admin/thr')({
  component: ThrPage,
})
