import { createFileRoute } from '@tanstack/react-router'
import ArsipPage from '@/components/ArsipPage'

export const Route = createFileRoute('/_app/arsip')({
  component: ArsipPage,
})
