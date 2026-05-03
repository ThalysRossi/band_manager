import { apiRequest } from '../../shared/api/client'
import { createSupabaseClient } from '../../shared/auth/supabase'

export type SignupOwnerValues = {
  email: string
  password: string
  bandName: string
  bandTimezone: string
}

export type LoginValues = {
  email: string
  password: string
}

export type CurrentAccountResponse = {
  user: {
    id: string
    email: string
  }
  activeBand: {
    bandId: string
    bandName: string
    role: 'owner' | 'admin' | 'member' | 'viewer'
    canWrite: boolean
  }
}

export async function signupOwner(
  values: SignupOwnerValues
): Promise<CurrentAccountResponse | null> {
  const supabase = createSupabaseClient()
  const authResult = await supabase.auth.signUp({
    email: values.email,
    password: values.password
  })
  if (authResult.error !== null) {
    throw new Error(authResult.error.message)
  }

  const accessToken = authResult.data.session?.access_token
  if (accessToken === undefined) {
    return null
  }

  return createOwnerAccount(accessToken, values)
}

export async function login(values: LoginValues): Promise<CurrentAccountResponse> {
  const supabase = createSupabaseClient()
  const authResult = await supabase.auth.signInWithPassword({
    email: values.email,
    password: values.password
  })
  if (authResult.error !== null) {
    throw new Error(authResult.error.message)
  }

  const session = authResult.data.session
  if (session === null) {
    throw new Error('Authenticated session is required')
  }

  const accessToken = session.access_token
  return getCurrentAccount(accessToken)
}

export async function getCurrentAccount(accessToken: string): Promise<CurrentAccountResponse> {
  return apiRequest<CurrentAccountResponse>({
    accessToken,
    path: '/me',
    method: 'GET',
    body: null,
    idempotent: false
  })
}

async function createOwnerAccount(
  accessToken: string,
  values: SignupOwnerValues
): Promise<CurrentAccountResponse> {
  return apiRequest<CurrentAccountResponse>({
    accessToken,
    path: '/auth/signup',
    method: 'POST',
    body: {
      email: values.email,
      bandName: values.bandName,
      bandTimezone: values.bandTimezone
    },
    idempotent: true
  })
}
