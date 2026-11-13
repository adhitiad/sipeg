import { createFileRoute } from '@tanstack/react-router'
import PayrollSlipDetailPage from '@/components/PayrollSlipDetailPage'

export const Route = createFileRoute('/_app/payroll/slips/$payrollId')({
  component: PayrollSlipDetailPage,
})
