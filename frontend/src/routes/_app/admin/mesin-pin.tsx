import { createFileRoute } from '@tanstack/react-router'
import MesinPinPage from '@/components/MesinPinPage'

export const Route = createFileRoute('/_app/admin/mesin-pin')({
  component: MesinPinPage,
})
