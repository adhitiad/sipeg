import { createFileRoute } from '@tanstack/react-router'
import PensiunPage from '@/components/PensiunPage'

export const Route = createFileRoute('/_app/admin/pensiun')({
  component: PensiunPage,
})
