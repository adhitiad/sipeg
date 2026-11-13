import { createFileRoute } from '@tanstack/react-router'
import RolesAdminPage from '@/components/RolesAdminPage'

export const Route = createFileRoute('/_app/admin/roles')({
  component: RolesAdminPage,
})
