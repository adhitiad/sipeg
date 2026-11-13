import { createFileRoute } from '@tanstack/react-router'
import EmployeeManagementPage from '@/components/EmployeeManagementPage'

export const Route = createFileRoute('/_app/employees')({
  component: EmployeeManagementPage,
})
