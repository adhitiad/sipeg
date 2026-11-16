import { createFileRoute } from '@tanstack/react-router'
import RbacAdminPage from '@/components/RbacAdminPage'

export const Route = createFileRoute('/_app/admin/rbac')({
  component: RbacAdminPage,
})
