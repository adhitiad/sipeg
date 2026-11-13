import { createFileRoute } from '@tanstack/react-router'
import PayrollSchedulesPage from '@/components/PayrollSchedulesPage'

export const Route = createFileRoute('/_app/payroll/schedules')({
  component: PayrollSchedulesPage,
})
