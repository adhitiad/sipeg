import { QueryClientProvider } from '@tanstack/react-query'
import { HeadContent, Scripts, createRootRouteWithContext } from '@tanstack/react-router'
import { TanStackRouterDevtoolsPanel } from '@tanstack/react-router-devtools'
import { TanStackDevtools } from '@tanstack/react-devtools'
import AppToaster from '@/components/AppToaster'
import GlobalError from '@/components/GlobalError'
import GlobalLoading from '@/components/GlobalLoading'
import GlobalNotFound from '@/components/GlobalNotFound'
import { TooltipProvider } from '@/components/ui/tooltip'
import { LocaleProvider } from '@/i18n/locale'
import type { RouterContext } from '@/lib/types/router'

import appCss from '../styles.css?url'
import componentsCss from '../styles/components.css?url'

const THEME_INIT_SCRIPT = `(function(){try{var stored=window.localStorage.getItem('theme');var mode=(stored==='light'||stored==='dark')?stored:'light';var root=document.documentElement;root.classList.remove('light','dark');root.classList.add(mode);root.setAttribute('data-theme',mode);root.style.colorScheme=mode;}catch(e){}})();`

export const Route = createRootRouteWithContext<RouterContext>()({
  head: () => ({
    meta: [
      { charSet: 'utf-8' },
      {
        name: 'viewport',
        content: 'width=device-width, initial-scale=1',
      },
      { title: 'SIMPEG' },
      {
        name: 'description',
        content:
          'SIMPEG adalah sistem informasi kepegawaian untuk mengelola data pegawai, organisasi, cuti, dan penggajian.',
      },
    ],
    links: [
      { rel: 'stylesheet', href: appCss },
      { rel: 'stylesheet', href: componentsCss },
    ],
  }),
  errorComponent: GlobalError,
  notFoundComponent: GlobalNotFound,
  pendingComponent: GlobalLoading,
  shellComponent: RootDocument,
})

function RootDocument({ children }: { children: React.ReactNode }) {
  const { queryClient } = Route.useRouteContext()

  return (
    <html lang="id" suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: THEME_INIT_SCRIPT }} />
        <HeadContent />
      </head>
      <body className="font-sans antialiased [overflow-wrap:anywhere]">
        <QueryClientProvider client={queryClient}>
          <LocaleProvider>
            <TooltipProvider>{children}</TooltipProvider>
            <AppToaster />
          </LocaleProvider>
        </QueryClientProvider>
        {import.meta.env.DEV ? (
          <TanStackDevtools
            config={{ position: 'bottom-right' }}
            plugins={[
              {
                name: 'Tanstack Router',
                render: <TanStackRouterDevtoolsPanel />,
              },
            ]}
          />
        ) : null}
        <Scripts />
      </body>
    </html>
  )
}
