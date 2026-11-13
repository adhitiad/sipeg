import { createFileRoute } from '@tanstack/react-router'
import JabatanPage from '@/components/JabatanPage'

export const Route = createFileRoute('/_app/jabatan')({
  component: JabatanPage,
})
