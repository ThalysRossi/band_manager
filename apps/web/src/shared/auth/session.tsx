import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { getAuthSession, logout as logoutFromProvider, subscribeToAuthSession } from './provider'
import type { AuthSession } from './provider'

type LoadingSessionState = {
  status: 'loading'
}

type UnauthenticatedSessionState = {
  status: 'unauthenticated'
}

type AuthenticatedSessionState = {
  status: 'authenticated'
  accessToken: string
  email: string
}

export type AuthSessionState =
  | LoadingSessionState
  | UnauthenticatedSessionState
  | AuthenticatedSessionState

type AuthSessionContextValue = {
  state: AuthSessionState
  refresh: () => Promise<void>
  logout: () => Promise<void>
}

const AuthSessionContext = createContext<AuthSessionContextValue | null>(null)

export function AuthSessionProvider(props: { children: ReactNode }) {
  const [state, setState] = useState<AuthSessionState>({ status: 'loading' })

  const refresh = useCallback(async (): Promise<void> => {
    setState(toAuthSessionState(await getAuthSession()))
  }, [])

  const logout = useCallback(async (): Promise<void> => {
    await logoutFromProvider()
    setState({ status: 'unauthenticated' })
  }, [])

  useEffect(() => {
    let active = true
    getAuthSession()
      .then((session) => {
        if (active) {
          setState(toAuthSessionState(session))
        }
      })
      .catch(() => {
        if (active) {
          setState({ status: 'unauthenticated' })
        }
      })

    const subscription = subscribeToAuthSession((session) => {
      setState(toAuthSessionState(session))
    })

    return () => {
      active = false
      subscription.unsubscribe()
    }
  }, [])

  const value = useMemo<AuthSessionContextValue>(() => {
    return {
      state,
      refresh,
      logout
    }
  }, [logout, refresh, state])

  return <AuthSessionContext.Provider value={value}>{props.children}</AuthSessionContext.Provider>
}

export function useAuthSession(): AuthSessionContextValue {
  const value = useContext(AuthSessionContext)
  if (value === null) {
    throw new Error('useAuthSession must be used inside AuthSessionProvider')
  }

  return value
}

function toAuthSessionState(session: AuthSession | null): AuthSessionState {
  if (session === null) {
    return { status: 'unauthenticated' }
  }

  return {
    status: 'authenticated',
    accessToken: session.accessToken,
    email: session.email
  }
}
