import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  Link,
  Outlet,
  RouterProvider,
  createRootRoute,
  createRoute,
  createRouter
} from '@tanstack/react-router'
import { CalendarDays, ChartNoAxesCombined, Package, Store } from 'lucide-react'
import type { Locale, TranslationKey } from 'i18n'
import { translations } from 'i18n'

import { detectLocale } from '../shared/i18n/detectLocale'

type NavigationItem = {
  key: NavigationLabelKey
  href: '/' | '/merch-booth' | '/financial-reports' | '/calendar'
  icon: typeof Package
}

type NavigationLabelKey = 'nav.inventory' | 'nav.merchBooth' | 'nav.reports' | 'nav.calendar'

const navigationItems: NavigationItem[] = [
  { key: 'nav.inventory', href: '/', icon: Package },
  { key: 'nav.merchBooth', href: '/merch-booth', icon: Store },
  { key: 'nav.reports', href: '/financial-reports', icon: ChartNoAxesCombined },
  { key: 'nav.calendar', href: '/calendar', icon: CalendarDays }
]

const queryClient = new QueryClient()
const rootRoute = createRootRoute({ component: RootLayout })

const inventoryRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: InventoryPage
})

const merchBoothRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/merch-booth',
  component: MerchBoothPage
})

const financialReportsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/financial-reports',
  component: FinancialReportsPage
})

const calendarRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/calendar',
  component: CalendarPage
})

const routeTree = rootRoute.addChildren([
  inventoryRoute,
  merchBoothRoute,
  financialReportsRoute,
  calendarRoute
])

const router = createRouter({ routeTree })

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
}

function RootLayout() {
  const locale = detectLocale(window.navigator.language)
  const translate = createTranslator(locale)

  return (
    <main className="app-shell">
      <header className="top-bar">
        <h1>{translate('app.title')}</h1>
      </header>

      <section className="workspace" aria-label={translate('app.title')}>
        <Outlet />
      </section>

      <nav className="bottom-nav" aria-label={translate('app.title')}>
        {navigationItems.map((item) => {
          const Icon = item.icon

          return (
            <Link key={item.key} to={item.href}>
              <Icon aria-hidden="true" size={20} strokeWidth={2} />
              <span>{translate(item.key)}</span>
            </Link>
          )
        })}
      </nav>
    </main>
  )
}

function InventoryPage() {
  return <WorkspaceHeader titleKey="nav.inventory" />
}

function MerchBoothPage() {
  return <WorkspaceHeader titleKey="nav.merchBooth" />
}

function FinancialReportsPage() {
  return <WorkspaceHeader titleKey="nav.reports" />
}

function CalendarPage() {
  return <WorkspaceHeader titleKey="nav.calendar" />
}

function WorkspaceHeader(props: { titleKey: NavigationLabelKey }) {
  const locale = detectLocale(window.navigator.language)
  const translate = createTranslator(locale)

  return (
    <div className="workspace-header">
      <h2>{translate(props.titleKey)}</h2>
      <p>{translate('status.backendReady')}</p>
    </div>
  )
}

function createTranslator(locale: Locale): (key: TranslationKey) => string {
  return (key: TranslationKey): string => translations[locale][key]
}
