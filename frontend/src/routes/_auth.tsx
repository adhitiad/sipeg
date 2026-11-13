import { Outlet, createFileRoute } from '@tanstack/react-router'
import ThemeToggle from '@/components/ThemeToggle'
import { RequireNoAuth } from '@/lib/require-auth'

export const Route = createFileRoute('/_auth')({
  component: AuthLayout,
})

function AuthLayout() {
  return (
    <RequireNoAuth>
      <main className="flex min-h-screen flex-col px-4 py-6">
        <div className="flex justify-end">
          <ThemeToggle />
        </div>
        <div className="flex flex-1 items-center justify-center py-8">
          <Outlet />
        </div>
      </main>
    </RequireNoAuth>
  )
}
