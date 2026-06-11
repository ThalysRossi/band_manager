import { apiRequest } from '../../shared/api/client'
import {
  completeAuthCallback,
  loginWithPassword,
  requestPasswordReset,
  signupWithPassword,
  updatePassword
} from '../../shared/auth/provider'

export type SignupValues = {
  email: string
  password: string
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

export type OnboardingValues = {
  bandName: string
  bandTimezone: string
}

export async function signup(values: SignupValues): Promise<void> {
  return signupWithPassword(values.email, values.password)
}

export async function login(values: LoginValues): Promise<void> {
  await loginWithPassword(values.email, values.password)
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

export async function onboardOwner(
  accessToken: string,
  values: OnboardingValues
): Promise<CurrentAccountResponse> {
  return apiRequest<CurrentAccountResponse>({
    accessToken,
    path: '/account/onboarding',
    method: 'POST',
    body: {
      bandName: values.bandName,
      bandTimezone: values.bandTimezone
    },
    idempotent: true
  })
}

export async function finishAuthCallback(code: string): Promise<void> {
  return completeAuthCallback(code)
}

export async function sendPasswordReset(email: string): Promise<void> {
  return requestPasswordReset(email)
}

export async function setNewPassword(password: string): Promise<void> {
  return updatePassword(password)
}
