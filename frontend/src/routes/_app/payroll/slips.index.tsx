import { createFileRoute } from '@tanstack/react-router'
import PayrollSlipsPage from '@/components/PayrollSlipsPage'

export const Route = createFileRoute('/_app/payroll/slips/')({
  component: PayrollSlipsPage,
})
