import { createFileRoute } from '@tanstack/react-router'
import HonorSksPage from '@/components/HonorSksPage'

export const Route = createFileRoute('/_app/admin/honor-sks')({
  component: HonorSksPage,
})
