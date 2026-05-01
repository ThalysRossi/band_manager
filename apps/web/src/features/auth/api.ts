import { createClient } from '@supabase/supabase-js'

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

type CurrentAccountResponse = {
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

  const accessToken = authResult.data.session.access_token
  return getCurrentAccount(accessToken)
}

async function createOwnerAccount(
  accessToken: string,
  values: SignupOwnerValues
): Promise<CurrentAccountResponse> {
  const response = await fetch(`${apiBaseURL()}/auth/signup`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${accessToken}`,
      'Content-Type': 'application/json',
      'Idempotency-Key': crypto.randomUUID()
    },
    body: JSON.stringify({
      email: values.email,
      bandName: values.bandName,
      bandTimezone: values.bandTimezone
    })
  })

  return parseAccountResponse(response)
}

async function getCurrentAccount(accessToken: string): Promise<CurrentAccountResponse> {
  const response = await fetch(`${apiBaseURL()}/me`, {
    headers: {
      Authorization: `Bearer ${accessToken}`
    }
  })

  return parseAccountResponse(response)
}

async function parseAccountResponse(response: Response): Promise<CurrentAccountResponse> {
  if (!response.ok) {
    const body = await response.json()
    throw new Error(body.message)
  }

  return response.json()
}

function apiBaseURL(): string {
  return requiredEnv('VITE_API_BASE_URL')
}

function createSupabaseClient() {
  return createClient(requiredEnv('VITE_SUPABASE_URL'), requiredEnv('VITE_SUPABASE_ANON_KEY'))
}

function requiredEnv(
  key: 'VITE_API_BASE_URL' | 'VITE_SUPABASE_URL' | 'VITE_SUPABASE_ANON_KEY'
): string {
  const value = import.meta.env[key]
  if (value === undefined || value.trim() === '') {
    throw new Error(`${key} is required`)
  }

  return value
}
