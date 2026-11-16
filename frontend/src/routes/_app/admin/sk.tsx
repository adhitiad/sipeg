import { createFileRoute } from '@tanstack/react-router'
import SkPage from '@/components/SkPage'

export const Route = createFileRoute('/_app/admin/sk')({
  component: SkPage,
})
