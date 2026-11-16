import { createFileRoute } from '@tanstack/react-router'
import KeluargaPage from '@/components/KeluargaPage'

export const Route = createFileRoute('/_app/keluarga')({
  component: KeluargaPage,
})
