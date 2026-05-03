import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import type { Session } from '@supabase/supabase-js'

import { createSupabaseClient } from './supabase'

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
}

const AuthSessionContext = createContext<AuthSessionContextValue | null>(null)

export function AuthSessionProvider(props: { children: ReactNode }) {
  const [state, setState] = useState<AuthSessionState>({ status: 'loading' })

  const refresh = useCallback(async (): Promise<void> => {
    const supabase = createSupabaseClient()
    const result = await supabase.auth.getSession()
    setState(toAuthSessionState(result.data.session))
  }, [])

  useEffect(() => {
    let active = true
    const supabase = createSupabaseClient()

    supabase.auth
      .getSession()
      .then((result) => {
        if (active) {
          setState(toAuthSessionState(result.data.session))
        }
      })
      .catch(() => {
        if (active) {
          setState({ status: 'unauthenticated' })
        }
      })

    const listener = supabase.auth.onAuthStateChange((_event, session) => {
      setState(toAuthSessionState(session))
    })

    return () => {
      active = false
      listener.data.subscription.unsubscribe()
    }
  }, [])

  const value = useMemo<AuthSessionContextValue>(() => {
    return {
      state,
      refresh
    }
  }, [refresh, state])

  return <AuthSessionContext.Provider value={value}>{props.children}</AuthSessionContext.Provider>
}

export function useAuthSession(): AuthSessionContextValue {
  const value = useContext(AuthSessionContext)
  if (value === null) {
    throw new Error('useAuthSession must be used inside AuthSessionProvider')
  }

  return value
}

function toAuthSessionState(session: Session | null): AuthSessionState {
  if (session === null) {
    return { status: 'unauthenticated' }
  }

  const email = session.user.email
  if (email === undefined || email.trim() === '') {
    return { status: 'unauthenticated' }
  }

  return {
    status: 'authenticated',
    accessToken: session.access_token,
    email
  }
}
