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
import { useMessages } from '@/i18n/locale-context'
import { DEFAULT_LOCALE, translate, type MessageKey } from '@/i18n/dictionary'
import { socialMeta } from '@/lib/seo'
import type { RouterContext } from '@/lib/types/router'

import appCss from '../styles.css?url'
import componentsCss from '../styles/components.css?url'

const THEME_INIT_SCRIPT = `(function(){try{var stored=window.localStorage.getItem('theme');var mode=(stored==='light'||stored==='dark')?stored:'light';var root=document.documentElement;root.classList.remove('light','dark');root.classList.add(mode);root.setAttribute('data-theme',mode);root.style.colorScheme=mode;}catch(e){}})();`

const t = (key: MessageKey) => translate(DEFAULT_LOCALE, key)

export const Route = createRootRouteWithContext<RouterContext>()({
  head: () => ({
    meta: [
      { charSet: 'utf-8' },
      {
        name: 'viewport',
        content: 'width=device-width, initial-scale=1',
      },
      { title: t('seo.siteTitle') },
      {
        name: 'description',
        content: t('seo.siteDescription'),
      },
      ...socialMeta({
        title: t('seo.siteTitle'),
        description: t('seo.siteDescription'),
        path: '/',
      }),
    ],
    links: [
      { rel: 'stylesheet', href: appCss },
      { rel: 'stylesheet', href: componentsCss },
      { rel: 'icon', href: '/favicon.svg', type: 'image/svg+xml' },
      { rel: 'icon', href: '/favicon-32x32.png', sizes: '32x32', type: 'image/png' },
      { rel: 'apple-touch-icon', href: '/apple-touch-icon.png', sizes: '180x180' },
    ],
  }),
  errorComponent: GlobalError,
  notFoundComponent: GlobalNotFound,
  pendingComponent: GlobalLoading,
  shellComponent: RootDocument,
})

function SkipLink() {
  const t = useMessages()
  return (
    <a href="#konten-utama" className="skip-link">
      {t('nav.skipToContent')}
    </a>
  )
}

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
            <SkipLink />
            <TooltipProvider>
              <div id="konten-utama" tabIndex={-1} className="outline-none">
                {children}
              </div>
            </TooltipProvider>
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
