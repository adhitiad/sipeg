import { createFileRoute } from '@tanstack/react-router'
import PemberhentianPage from '@/components/PemberhentianPage'

export const Route = createFileRoute('/_app/admin/pemberhentian')({
  component: PemberhentianPage,
})
