import { createFileRoute } from '@tanstack/react-router'
import PayrollScalePage from '@/components/PayrollScalePage'

export const Route = createFileRoute('/_app/admin/payroll-scale')({
  component: PayrollScalePage,
})
