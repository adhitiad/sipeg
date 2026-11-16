import { createFileRoute } from '@tanstack/react-router'
import KontrakPage from '@/components/KontrakPage'

export const Route = createFileRoute('/_app/admin/kontrak')({
  component: KontrakPage,
})
