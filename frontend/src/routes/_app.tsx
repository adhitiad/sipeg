import { Outlet, createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import Footer from '@/components/Footer'
import Header from '@/components/Header'
import Sidebar from '@/components/Sidebar'
import { RequireAuth } from '@/lib/require-auth'
import { useSession } from '@/lib/session'
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar'

export const Route = createFileRoute('/_app')({
  component: AppLayout,
})

function AppLayout() {
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const session = useSession()
  const account = session.data ?? null
  const permissions = account?.permissions ?? []

  return (
    <RequireAuth>
      <SidebarProvider open={sidebarOpen} onOpenChange={setSidebarOpen}>
        <Sidebar
          account={{
            email: account?.email ?? '',
            roleName: account?.roleName ?? null,
            permissions,
          }}
        />
        <SidebarInset>
          <Header
            withSidebarTrigger
            account={{
              email: account?.email ?? '',
              roleName: account?.roleName ?? null,
              permissions,
            }}
          />
          <div className="flex-1">
            <Outlet />
          </div>
          <Footer />
        </SidebarInset>
      </SidebarProvider>
    </RequireAuth>
  )
}
