import { createFileRoute } from '@tanstack/react-router'
import BpjsPage from '@/components/BpjsPage'

export const Route = createFileRoute('/_app/admin/bpjs')({
  component: BpjsPage,
})
