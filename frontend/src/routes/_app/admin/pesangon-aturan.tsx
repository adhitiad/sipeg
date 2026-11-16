import { createFileRoute } from '@tanstack/react-router'
import PesangonAturanPage from '@/components/PesangonAturanPage'

export const Route = createFileRoute('/_app/admin/pesangon-aturan')({
  component: PesangonAturanPage,
})
