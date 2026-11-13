import { createFileRoute } from '@tanstack/react-router'
import AuditLogPage from '@/components/AuditLogPage'

export const Route = createFileRoute('/_app/audit-logs')({
  component: AuditLogPage,
})
