import { useState } from 'react'
import type { ReactNode } from 'react'
import { QueryClient, QueryClientProvider, useQuery } from '@tanstack/react-query'
import {
  Link,
  Navigate,
  Outlet,
  RouterProvider,
  createRootRoute,
  createRoute,
  createRouter,
  useNavigate
} from '@tanstack/react-router'
import { CalendarDays, ChartNoAxesCombined, Package, Store, UserRound } from 'lucide-react'
import type { Locale, TranslationKey } from 'i18n'
import { translations } from 'i18n'

import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { AccountPage, AcceptInvitePage } from '../features/account/AccountPages'
import { LoginPage, SignupPage } from '../features/auth/AuthPages'
import { getCurrentAccount } from '../features/auth/api'
import type { CurrentAccountResponse } from '../features/auth/api'
import { AuthSessionProvider, useAuthSession } from '../shared/auth/session'
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

type ProtectedRoutePath = NavigationItem['href']

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
  validateSearch: (search: Record<string, unknown>): { redirect: ProtectedRoutePath } => {
    return {
      redirect: parseProtectedRoutePath(search.redirect)
    }
  },
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
        <div className="brand-lockup">
          <h1>{translate('app.title')}</h1>
        </div>
        <HeaderAccountSummary translate={translate} />
      </header>

      <section className="workspace" aria-label={translate('app.title')}>
        <Outlet />
      </section>

      <nav className="bottom-nav" aria-label={translate('app.title')}>
        {navigationItems.map((item) => {
          const Icon = item.icon

          return (
            <Link
              key={item.key}
              to={item.href}
              activeOptions={{ exact: item.href === '/' }}
              activeProps={{ className: 'is-active' }}
            >
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
  return (
    <ProtectedRoute redirect="/">
      <WorkspaceHeader titleKey="nav.inventory" />
    </ProtectedRoute>
  )
}

function MerchBoothPage() {
  return (
    <ProtectedRoute redirect="/merch-booth">
      <WorkspaceHeader titleKey="nav.merchBooth" />
    </ProtectedRoute>
  )
}

function FinancialReportsPage() {
  return (
    <ProtectedRoute redirect="/financial-reports">
      <WorkspaceHeader titleKey="nav.reports" />
    </ProtectedRoute>
  )
}

function CalendarPage() {
  return (
    <ProtectedRoute redirect="/calendar">
      <WorkspaceHeader titleKey="nav.calendar" />
    </ProtectedRoute>
  )
}

function AccountRoutePage() {
  const translate = useTranslate()

  return (
    <ProtectedRoute redirect="/account">
      <AccountPage translate={translate} />
    </ProtectedRoute>
  )
}

function AcceptInviteRoutePage() {
  const translate = useTranslate()
  const search = acceptInviteRoute.useSearch()

  return <AcceptInvitePage translate={translate} token={search.token} />
}

function LoginRoutePage() {
  const translate = useTranslate()
  const navigate = useNavigate()
  const session = useAuthSession()
  const search = loginRoute.useSearch()

  return (
    <LoginPage
      translate={translate}
      onLoginSuccess={() => {
        void session.refresh().then(() => navigate({ to: search.redirect }))
      }}
    />
  )
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

function HeaderAccountSummary(props: { translate: (key: TranslationKey) => string }) {
  const session = useAuthSession()
  const accessToken = session.state.status === 'authenticated' ? session.state.accessToken : ''

  const accountQuery = useQuery({
    queryKey: ['account', 'current', accessToken],
    queryFn: () => getCurrentAccount(accessToken),
    enabled: session.state.status === 'authenticated'
  })

  if (session.state.status === 'loading') {
    return <p className="top-bar-status">{props.translate('account.loading')}</p>
  }

  if (session.state.status === 'unauthenticated') {
    return null
  }

  if (accountQuery.isLoading) {
    return <p className="top-bar-status">{props.translate('account.loading')}</p>
  }

  if (accountQuery.isError || accountQuery.data === undefined) {
    return (
      <p className="top-bar-status" role="status">
        {props.translate('account.genericError')}
      </p>
    )
  }

  return <BandContext account={accountQuery.data} translate={props.translate} />
}

function BandContext(props: {
  account: CurrentAccountResponse
  translate: (key: TranslationKey) => string
}) {
  return (
    <div className="band-context">
      <Avatar className="band-avatar" aria-hidden="true">
        <AvatarFallback className="band-avatar-fallback">
          {bandInitials(props.account.activeBand.bandName)}
        </AvatarFallback>
      </Avatar>
      <span className="band-copy">
        <span className="band-name">{props.account.activeBand.bandName}</span>
        <span className="band-meta">
          {props.account.user.email} |{' '}
          {props.translate(roleLabelKey(props.account.activeBand.role))}
        </span>
      </span>
    </div>
  )
}

function ProtectedRoute(props: { redirect: ProtectedRoutePath; children: ReactNode }) {
  const session = useAuthSession()
  const translate = useTranslate()

  if (session.state.status === 'loading') {
    return (
      <div className="workspace-header">
        <p>{translate('account.loading')}</p>
      </div>
    )
  }

  if (session.state.status === 'unauthenticated') {
    return <Navigate to="/login" search={{ redirect: props.redirect }} />
  }

  return <>{props.children}</>
}

function useTranslate(): (key: TranslationKey) => string {
  const locale = detectLocale(window.navigator.language)

  return createTranslator(locale)
}

function createTranslator(locale: Locale): (key: TranslationKey) => string {
  return (key: TranslationKey): string => translations[locale][key]
}

function roleLabelKey(role: CurrentAccountResponse['activeBand']['role']): TranslationKey {
  return `account.role.${role}`
}

function bandInitials(value: string): string {
  const words = value
    .trim()
    .split(/\s+/)
    .filter((word) => word !== '')

  if (words.length === 0) {
    return 'BM'
  }

  return words
    .slice(0, 2)
    .map((word) => word.slice(0, 1).toUpperCase())
    .join('')
}

function parseProtectedRoutePath(value: unknown): ProtectedRoutePath {
  if (typeof value !== 'string') {
    return '/'
  }

  const matchingItem = navigationItems.find((item) => item.href === value)
  if (matchingItem === undefined) {
    return '/'
  }

  return matchingItem.href
}
