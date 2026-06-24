import { useEffect, useState } from 'react'
import type { ReactNode } from 'react'
import { QueryClient, QueryClientProvider, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Link,
  Navigate,
  Outlet,
  RouterProvider,
  createRootRoute,
  createRoute,
  createRouter,
  useLocation,
  useNavigate
} from '@tanstack/react-router'
import { CalendarDays, ChartNoAxesCombined, Package, Store, UserRound } from 'lucide-react'
import type { Locale, TranslationKey } from 'i18n'
import { translations } from 'i18n'

import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import { AccountPage, AcceptInvitePage } from '../features/account/AccountPages'
import {
  LoginPage,
  OnboardingPage,
  PasswordResetPage,
  PasswordUpdatePage,
  SignupPage
} from '../features/auth/AuthPages'
import { finishAuthCallback, getCurrentAccount } from '../features/auth/api'
import type { CurrentAccountResponse } from '../features/auth/api'
import { ApiError } from '../shared/api/client'
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

type HeaderLabelKey =
  | NavigationLabelKey
  | 'auth.loginTitle'
  | 'auth.signupTitle'
  | 'auth.onboardingTitle'
  | 'auth.passwordReset'
  | 'auth.passwordUpdateTitle'
  | 'account.acceptTitle'

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

const onboardingRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/onboarding',
  component: OnboardingRoutePage
})

const authCallbackRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/auth/callback',
  validateSearch: (search: Record<string, unknown>): { code: string; next: string } => ({
    code: typeof search.code === 'string' ? search.code : '',
    next: typeof search.next === 'string' ? search.next : '/onboarding'
  }),
  component: AuthCallbackRoutePage
})

const passwordResetRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/password-reset',
  component: PasswordResetRoutePage
})

const passwordUpdateRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/password-update',
  component: PasswordUpdateRoutePage
})

const routeTree = rootRoute.addChildren([
  inventoryRoute,
  merchBoothRoute,
  financialReportsRoute,
  calendarRoute,
  accountRoute,
  acceptInviteRoute,
  loginRoute,
  signupRoute,
  onboardingRoute,
  authCallbackRoute,
  passwordResetRoute,
  passwordUpdateRoute
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
  const location = useLocation()
  const headerLabelKey = headerLabelForPath(location.pathname)

  return (
    <main className="min-h-screen bg-[linear-gradient(180deg,#151813_0%,var(--color-black-100)_32%,var(--color-black-100)_100%)] pb-[72px] min-[800px]:grid min-[800px]:grid-cols-[240px_minmax(0,1fr)] min-[800px]:pb-0">
      <header className="flex min-h-16 items-center justify-between gap-ui-16 border-b border-border bg-[rgba(17,19,15,0.92)] p-ui-16 backdrop-blur-md min-[800px]:col-span-full min-[800px]:px-ui-32 min-[800px]:py-[18px]">
        <div className="grid min-w-0 gap-ui-2">
          <h1 className="m-0 text-lg font-bold">{translate('app.title')}</h1>
          <p className="m-0 overflow-hidden text-ellipsis whitespace-nowrap text-xs font-[650] text-white-300">
            {translate(headerLabelKey)}
          </p>
        </div>
        <HeaderAccountSummary translate={translate} />
      </header>

      <section
        className="mx-auto w-[min(100%,960px)] px-ui-16 py-ui-32 min-[800px]:col-start-2 min-[800px]:row-start-2 min-[800px]:p-ui-32"
        aria-label={translate('app.title')}
      >
        <Outlet />
      </section>

      <nav
        className="fixed inset-x-0 bottom-0 grid min-h-16 grid-cols-5 border-t border-border bg-[rgba(20,23,18,0.96)] backdrop-blur-md min-[800px]:sticky min-[800px]:top-[65px] min-[800px]:col-start-1 min-[800px]:row-start-2 min-[800px]:flex min-[800px]:min-h-[calc(100vh-65px)] min-[800px]:flex-col min-[800px]:border-r min-[800px]:border-t-0 min-[800px]:p-ui-12"
        aria-label={translate('app.title')}
      >
        {navigationItems.map((item) => {
          const Icon = item.icon

          return (
            <Link
              key={item.key}
              to={item.href}
              activeOptions={{ exact: item.href === '/' }}
              className="grid min-h-16 min-w-0 content-center place-items-center gap-ui-4 text-xs font-[650] text-white-200 min-[800px]:min-h-11 min-[800px]:grid-cols-[24px_minmax(0,1fr)] min-[800px]:justify-start min-[800px]:rounded-md min-[800px]:px-ui-12 min-[800px]:[place-items:center_start]"
              activeProps={{ className: 'bg-[#203b2f] text-white-100' }}
            >
              <Icon aria-hidden="true" size={20} strokeWidth={2} />
              <span className="max-w-full overflow-hidden text-ellipsis whitespace-nowrap px-ui-4">
                {translate(item.key)}
              </span>
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

function OnboardingRoutePage() {
  const translate = useTranslate()
  const navigate = useNavigate()
  const session = useAuthSession()

  if (session.state.status === 'loading') {
    return <p>{translate('account.loading')}</p>
  }
  if (session.state.status === 'unauthenticated') {
    return <Navigate to="/login" search={{ redirect: '/' }} />
  }

  return (
    <OnboardingPage
      translate={translate}
      accessToken={session.state.accessToken}
      onSuccess={() => void navigate({ to: '/' })}
    />
  )
}

function AuthCallbackRoutePage() {
  const translate = useTranslate()
  const navigate = useNavigate()
  const session = useAuthSession()
  const search = authCallbackRoute.useSearch()
  const [failed, setFailed] = useState<boolean>(false)

  useEffect(() => {
    if (search.code === '') {
      setFailed(true)
      return
    }
    void finishAuthCallback(search.code)
      .then(session.refresh)
      .then(() => navigate({ to: search.next }))
      .catch(() => setFailed(true))
  }, [navigate, search.code, search.next, session.refresh])

  if (failed) {
    return <p role="status">{translate('auth.genericError')}</p>
  }

  return <p>{translate('account.loading')}</p>
}

function PasswordResetRoutePage() {
  return <PasswordResetPage translate={useTranslate()} />
}

function PasswordUpdateRoutePage() {
  return <PasswordUpdatePage translate={useTranslate()} />
}

function WorkspaceHeader(props: { titleKey: NavigationLabelKey }) {
  const translate = useTranslate()

  return (
    <div className="grid gap-ui-8">
      <h2 className="m-0 text-[1.75rem] leading-[1.15]">{translate(props.titleKey)}</h2>
      <p className="m-0 text-base text-white-300">{translate('status.backendReady')}</p>
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
    return (
      <p className="m-0 text-right text-[0.8125rem] text-white-300">
        {props.translate('account.loading')}
      </p>
    )
  }

  if (session.state.status === 'unauthenticated') {
    return null
  }

  if (accountQuery.isLoading) {
    return (
      <p className="m-0 text-right text-[0.8125rem] text-white-300">
        {props.translate('account.loading')}
      </p>
    )
  }

  if (accountQuery.error instanceof ApiError && accountQuery.error.code === 'account_not_found') {
    return null
  }

  if (accountQuery.isError || accountQuery.data === undefined) {
    return (
      <p className="m-0 text-right text-[0.8125rem] text-white-300" role="status">
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
  const [logoutError, setLogoutError] = useState<string>('')
  const [logoutPending, setLogoutPending] = useState<boolean>(false)
  const queryClient = useQueryClient()
  const session = useAuthSession()

  async function handleLogout(): Promise<void> {
    setLogoutPending(true)
    setLogoutError('')
    try {
      await session.logout()
      queryClient.removeQueries({ queryKey: ['account'] })
    } catch {
      setLogoutError(props.translate('auth.logoutFailed'))
      setLogoutPending(false)
    }
  }

  return (
    <div className="grid justify-items-end gap-ui-4">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button
            type="button"
            className="grid min-w-0 grid-cols-[36px_minmax(0,1fr)] items-center gap-ui-10 rounded-md text-left outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50"
          >
            <Avatar className="size-9 border border-green-100/45 bg-[#163a2a] text-[#dff7ea]">
              <AvatarFallback className="bg-transparent text-[0.8125rem] font-extrabold text-inherit">
                {bandInitials(props.account.activeBand.bandName)}
              </AvatarFallback>
            </Avatar>
            <span className="grid min-w-0 gap-ui-2 text-right">
              <span className="overflow-hidden text-ellipsis whitespace-nowrap text-sm font-[750] text-white-100">
                {props.account.activeBand.bandName}
              </span>
              <span className="overflow-hidden text-ellipsis whitespace-nowrap text-xs text-white-300">
                {props.account.user.email} |{' '}
                {props.translate(roleLabelKey(props.account.activeBand.role))}
              </span>
            </span>
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-64">
          <DropdownMenuLabel className="grid gap-ui-2">
            <span className="overflow-hidden text-ellipsis whitespace-nowrap">
              {props.account.activeBand.bandName}
            </span>
            <span className="overflow-hidden text-ellipsis whitespace-nowrap text-xs font-normal text-muted-foreground">
              {props.account.user.email}
            </span>
            <span className="text-xs font-normal text-muted-foreground">
              {props.translate(roleLabelKey(props.account.activeBand.role))}
            </span>
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            disabled={logoutPending}
            onSelect={(event) => {
              event.preventDefault()
              void handleLogout()
            }}
          >
            {props.translate('auth.logout')}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      {logoutError === '' ? null : (
        <p className="m-0 text-right text-xs text-red-100" role="status">
          {logoutError}
        </p>
      )}
    </div>
  )
}

function ProtectedRoute(props: { redirect: ProtectedRoutePath; children: ReactNode }) {
  const session = useAuthSession()
  const translate = useTranslate()

  if (session.state.status === 'loading') {
    return (
      <div className="grid gap-ui-8">
        <p className="m-0 text-base text-white-300">{translate('account.loading')}</p>
      </div>
    )
  }

  if (session.state.status === 'unauthenticated') {
    return <Navigate to="/login" search={{ redirect: props.redirect }} />
  }

  return <AccountRequired>{props.children}</AccountRequired>
}

function AccountRequired(props: { children: ReactNode }) {
  const session = useAuthSession()
  const translate = useTranslate()
  const accountQuery = useQuery({
    queryKey: [
      'account',
      'required',
      session.state.status === 'authenticated' ? session.state.accessToken : ''
    ],
    queryFn: () =>
      getCurrentAccount(session.state.status === 'authenticated' ? session.state.accessToken : ''),
    enabled: session.state.status === 'authenticated',
    retry: false
  })

  if (accountQuery.error instanceof ApiError && accountQuery.error.code === 'account_not_found') {
    return <Navigate to="/onboarding" />
  }
  if (accountQuery.isError) {
    return <p role="status">{translate('account.genericError')}</p>
  }
  if (accountQuery.isLoading) {
    return <p>{translate('account.loading')}</p>
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

function headerLabelForPath(pathname: string): HeaderLabelKey {
  if (pathname === '/login') {
    return 'auth.loginTitle'
  }

  if (pathname === '/signup') {
    return 'auth.signupTitle'
  }

  if (pathname === '/onboarding') {
    return 'auth.onboardingTitle'
  }

  if (pathname === '/password-reset') {
    return 'auth.passwordReset'
  }

  if (pathname === '/password-update') {
    return 'auth.passwordUpdateTitle'
  }

  if (pathname.startsWith('/account/invites/accept')) {
    return 'account.acceptTitle'
  }

  const matchingItem = navigationItems
    .filter((item) => item.href !== '/')
    .find((item) => pathname === item.href || pathname.startsWith(`${item.href}/`))
  if (matchingItem !== undefined) {
    return matchingItem.key
  }

  return 'nav.inventory'
}
