import { useState } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  Link,
  Outlet,
  RouterProvider,
  createRootRoute,
  createRoute,
  createRouter
} from '@tanstack/react-router'
import { CalendarDays, ChartNoAxesCombined, Package, Store, UserRound } from 'lucide-react'
import type { Locale, TranslationKey } from 'i18n'
import { translations } from 'i18n'

import { AccountPage, AcceptInvitePage } from '../features/account/AccountPages'
import { LoginPage, SignupPage } from '../features/auth/AuthPages'
import { AuthSessionProvider } from '../shared/auth/session'
import { detectLocale } from '../shared/i18n/detectLocale'

type NavigationItem = {
  key: NavigationLabelKey
  href: '/' | '/merch-booth' | '/financial-reports' | '/calendar' | '/account'
  icon: typeof Package
}

type NavigationLabelKey =
  | 'nav.inventory'
  | 'nav.merchBooth'
  | 'nav.reports'
  | 'nav.calendar'
  | 'nav.account'

const navigationItems: NavigationItem[] = [
  { key: 'nav.inventory', href: '/', icon: Package },
  { key: 'nav.merchBooth', href: '/merch-booth', icon: Store },
  { key: 'nav.reports', href: '/financial-reports', icon: ChartNoAxesCombined },
  { key: 'nav.calendar', href: '/calendar', icon: CalendarDays },
  { key: 'nav.account', href: '/account', icon: UserRound }
]

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

const accountRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/account',
  component: AccountRoutePage
})

const acceptInviteRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/account/invites/accept',
  validateSearch: (search: Record<string, unknown>): { token: string } => {
    return {
      token: typeof search.token === 'string' ? search.token : ''
    }
  },
  component: AcceptInviteRoutePage
})

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/login',
  component: LoginRoutePage
})

const signupRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/signup',
  component: SignupRoutePage
})

const routeTree = rootRoute.addChildren([
  inventoryRoute,
  merchBoothRoute,
  financialReportsRoute,
  calendarRoute,
  accountRoute,
  acceptInviteRoute,
  loginRoute,
  signupRoute
])

const router = createRouter({ routeTree })

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}

export function App() {
  const [queryClient] = useState(() => new QueryClient())

  return (
    <QueryClientProvider client={queryClient}>
      <AuthSessionProvider>
        <RouterProvider router={router} />
      </AuthSessionProvider>
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

function AccountRoutePage() {
  const translate = useTranslate()

  return <AccountPage translate={translate} />
}

function AcceptInviteRoutePage() {
  const translate = useTranslate()
  const search = acceptInviteRoute.useSearch()

  return <AcceptInvitePage translate={translate} token={search.token} />
}

function LoginRoutePage() {
  const translate = useTranslate()

  return <LoginPage translate={translate} />
}

function SignupRoutePage() {
  const translate = useTranslate()

  return <SignupPage translate={translate} />
}

function WorkspaceHeader(props: { titleKey: NavigationLabelKey }) {
  const translate = useTranslate()

  return (
    <div className="workspace-header">
      <h2>{translate(props.titleKey)}</h2>
      <p>{translate('status.backendReady')}</p>
    </div>
  )
}

function useTranslate(): (key: TranslationKey) => string {
  const locale = detectLocale(window.navigator.language)

  return createTranslator(locale)
}

function createTranslator(locale: Locale): (key: TranslationKey) => string {
  return (key: TranslationKey): string => translations[locale][key]
}
