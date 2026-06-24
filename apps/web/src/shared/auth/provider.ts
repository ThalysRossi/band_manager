import { createSupabaseClient } from './supabase'

export type AuthSession = {
  accessToken: string
  email: string
}

export type AuthSubscription = {
  unsubscribe: () => void
}

export async function signupWithPassword(email: string, password: string): Promise<void> {
  const result = await createSupabaseClient().auth.signUp({
    email,
    password,
    options: {
      emailRedirectTo: `${window.location.origin}/auth/callback`
    }
  })
  throwIfAuthError(result.error)
}

export async function loginWithPassword(email: string, password: string): Promise<AuthSession> {
  const result = await createSupabaseClient().auth.signInWithPassword({ email, password })
  throwIfAuthError(result.error)
  if (result.data.session === null) {
    throw new Error('Authenticated session is required')
  }

  return toAuthSession(result.data.session, email)
}

export async function getAuthSession(): Promise<AuthSession | null> {
  const result = await createSupabaseClient().auth.getSession()
  throwIfAuthError(result.error)

  return result.data.session === null ? null : toAuthSession(result.data.session)
}

export async function logout(): Promise<void> {
  const result = await createSupabaseClient().auth.signOut()
  throwIfAuthError(result.error)
}

export function subscribeToAuthSession(
  listener: (session: AuthSession | null) => void
): AuthSubscription {
  const result = createSupabaseClient().auth.onAuthStateChange((_event, session) => {
    listener(session === null ? null : toAuthSession(session))
  })

  return {
    unsubscribe: () => result.data.subscription.unsubscribe()
  }
}

export async function completeAuthCallback(code: string): Promise<void> {
  const result = await createSupabaseClient().auth.exchangeCodeForSession(code)
  throwIfAuthError(result.error)
}

export async function requestPasswordReset(email: string): Promise<void> {
  const result = await createSupabaseClient().auth.resetPasswordForEmail(email, {
    redirectTo: `${window.location.origin}/auth/callback?next=/password-update`
  })
  throwIfAuthError(result.error)
}

export async function updatePassword(password: string): Promise<void> {
  const result = await createSupabaseClient().auth.updateUser({ password })
  throwIfAuthError(result.error)
}

function toAuthSession(
  session: { access_token: string; user?: { email?: string } },
  fallbackEmail?: string
): AuthSession {
  const email = session.user?.email ?? fallbackEmail
  if (email === undefined || email.trim() === '') {
    throw new Error('Authenticated session email is required')
  }

  return {
    accessToken: session.access_token,
    email
  }
}

function throwIfAuthError(error: { message: string } | null | undefined): void {
  if (error !== null && error !== undefined) {
    throw new Error(error.message)
  }
}
