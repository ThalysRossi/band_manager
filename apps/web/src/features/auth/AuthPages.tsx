import { useState } from 'react'
import type { TranslationKey } from 'i18n'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { onboardOwner, sendPasswordReset, setNewPassword, login, signup } from './api'

type Translate = (key: TranslationKey) => string

type AuthPageProps = {
  translate: Translate
  onLoginSuccess?: () => void
}

export function LoginPage(props: AuthPageProps) {
  const [status, setStatus] = useState<string>('')

  return (
    <Card className="w-[min(100%,420px)]">
      <CardHeader>
        <h2 className="m-0">{props.translate('auth.loginTitle')}</h2>
      </CardHeader>
      <CardContent className="grid gap-ui-16">
        <form
          className="grid gap-ui-16"
          onSubmit={(event) => {
            event.preventDefault()
            const values = new FormData(event.currentTarget)
            login({
              email: fieldValue(values, 'email'),
              password: fieldValue(values, 'password')
            })
              .then(() => {
                setStatus(props.translate('auth.loginReady'))
                if (props.onLoginSuccess !== undefined) {
                  props.onLoginSuccess()
                }
              })
              .catch(() => setStatus(props.translate('auth.genericError')))
          }}
        >
          <div className="grid gap-ui-8">
            <Label htmlFor="login-email">{props.translate('auth.emailLabel')}</Label>
            <Input id="login-email" name="email" type="email" autoComplete="email" />
          </div>
          <div className="grid gap-ui-8">
            <Label htmlFor="login-password">{props.translate('auth.passwordLabel')}</Label>
            <Input
              id="login-password"
              name="password"
              type="password"
              autoComplete="current-password"
            />
          </div>
          <Button type="submit">{props.translate('auth.loginSubmit')}</Button>
          <Button variant="link" asChild>
            <a href="/password-reset">{props.translate('auth.passwordReset')}</a>
          </Button>
        </form>
        {status === '' ? null : <p role="status">{status}</p>}
      </CardContent>
    </Card>
  )
}

export function SignupPage(props: AuthPageProps) {
  const [status, setStatus] = useState<string>('')

  return (
    <Card className="w-[min(100%,420px)]">
      <CardHeader>
        <h2 className="m-0">{props.translate('auth.signupTitle')}</h2>
        <CardDescription>{props.translate('auth.emailInstruction')}</CardDescription>
      </CardHeader>
      <CardContent className="grid gap-ui-16">
        <form
          className="grid gap-ui-16"
          onSubmit={(event) => {
            event.preventDefault()
            const values = new FormData(event.currentTarget)
            signup({
              email: fieldValue(values, 'email'),
              password: fieldValue(values, 'password')
            })
              .then(() => setStatus(props.translate('auth.emailVerificationRequired')))
              .catch(() => setStatus(props.translate('auth.genericError')))
          }}
        >
          <div className="grid gap-ui-8">
            <Label htmlFor="signup-email">{props.translate('auth.emailLabel')}</Label>
            <Input id="signup-email" name="email" type="email" autoComplete="email" />
          </div>
          <div className="grid gap-ui-8">
            <Label htmlFor="signup-password">{props.translate('auth.passwordLabel')}</Label>
            <Input
              id="signup-password"
              name="password"
              type="password"
              autoComplete="new-password"
            />
          </div>
          <Button type="submit">{props.translate('auth.signupSubmit')}</Button>
        </form>
        {status === '' ? null : <p role="status">{status}</p>}
      </CardContent>
    </Card>
  )
}

export function OnboardingPage(
  props: AuthPageProps & { accessToken: string; onSuccess: () => void }
) {
  const [status, setStatus] = useState<string>('')

  return (
    <Card className="w-[min(100%,420px)]">
      <CardHeader>
        <h2 className="m-0">{props.translate('auth.onboardingTitle')}</h2>
      </CardHeader>
      <CardContent className="grid gap-ui-16">
        <form
          className="grid gap-ui-16"
          onSubmit={(event) => {
            event.preventDefault()
            const values = new FormData(event.currentTarget)
            onboardOwner(props.accessToken, {
              bandName: fieldValue(values, 'bandName'),
              bandTimezone: fieldValue(values, 'bandTimezone')
            })
              .then(props.onSuccess)
              .catch(() => setStatus(props.translate('auth.genericError')))
          }}
        >
          <div className="grid gap-ui-8">
            <Label htmlFor="onboarding-band-name">{props.translate('auth.bandNameLabel')}</Label>
            <Input
              id="onboarding-band-name"
              name="bandName"
              type="text"
              autoComplete="organization"
            />
          </div>
          <div className="grid gap-ui-8">
            <Label htmlFor="onboarding-band-timezone">
              {props.translate('auth.timezoneLabel')}
            </Label>
            <Input
              id="onboarding-band-timezone"
              name="bandTimezone"
              type="text"
              autoComplete="off"
            />
          </div>
          <Button type="submit">{props.translate('auth.onboardingSubmit')}</Button>
        </form>
        {status === '' ? null : <p role="status">{status}</p>}
      </CardContent>
    </Card>
  )
}

export function PasswordResetPage(props: AuthPageProps) {
  const [status, setStatus] = useState<string>('')

  return (
    <Card className="w-[min(100%,420px)]">
      <CardHeader>
        <h2 className="m-0">{props.translate('auth.passwordReset')}</h2>
      </CardHeader>
      <CardContent className="grid gap-ui-16">
        <form
          className="grid gap-ui-16"
          onSubmit={(event) => {
            event.preventDefault()
            const values = new FormData(event.currentTarget)
            sendPasswordReset(fieldValue(values, 'email'))
              .then(() => setStatus(props.translate('auth.passwordResetSent')))
              .catch(() => setStatus(props.translate('auth.genericError')))
          }}
        >
          <div className="grid gap-ui-8">
            <Label htmlFor="reset-email">{props.translate('auth.emailLabel')}</Label>
            <Input id="reset-email" name="email" type="email" autoComplete="email" />
          </div>
          <Button type="submit">{props.translate('auth.passwordResetSubmit')}</Button>
        </form>
        {status === '' ? null : <p role="status">{status}</p>}
      </CardContent>
    </Card>
  )
}

export function PasswordUpdatePage(props: AuthPageProps) {
  const [status, setStatus] = useState<string>('')

  return (
    <Card className="w-[min(100%,420px)]">
      <CardHeader>
        <h2 className="m-0">{props.translate('auth.passwordUpdateTitle')}</h2>
      </CardHeader>
      <CardContent className="grid gap-ui-16">
        <form
          className="grid gap-ui-16"
          onSubmit={(event) => {
            event.preventDefault()
            const values = new FormData(event.currentTarget)
            setNewPassword(fieldValue(values, 'password'))
              .then(() => setStatus(props.translate('auth.passwordUpdated')))
              .catch(() => setStatus(props.translate('auth.genericError')))
          }}
        >
          <div className="grid gap-ui-8">
            <Label htmlFor="new-password">{props.translate('auth.passwordLabel')}</Label>
            <Input id="new-password" name="password" type="password" autoComplete="new-password" />
          </div>
          <Button type="submit">{props.translate('auth.passwordUpdateSubmit')}</Button>
        </form>
        {status === '' ? null : <p role="status">{status}</p>}
      </CardContent>
    </Card>
  )
}

function fieldValue(values: FormData, fieldName: string): string {
  const value = values.get(fieldName)
  if (typeof value !== 'string') {
    return ''
  }

  return value
}
