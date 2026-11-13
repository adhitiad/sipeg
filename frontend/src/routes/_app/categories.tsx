import { createFileRoute } from '@tanstack/react-router'
import CategoryPage from '@/components/CategoryPage'

export const Route = createFileRoute('/_app/categories')({
  component: CategoryPage,
})
