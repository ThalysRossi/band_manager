import { useState } from 'react'
import type { TranslationKey } from 'i18n'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { login, signupOwner } from './api'
import type { CurrentAccountResponse } from './api'

type Translate = (key: TranslationKey) => string

type AuthPageProps = {
  translate: Translate
  onLoginSuccess?: (account: CurrentAccountResponse) => void
}

export function LoginPage(props: AuthPageProps) {
  const [status, setStatus] = useState<string>('')

  return (
    <Card className="auth-panel">
      <CardHeader>
        <h2>{props.translate('auth.loginTitle')}</h2>
      </CardHeader>
      <CardContent className="auth-content">
        <form
          className="auth-form"
          onSubmit={(event) => {
            event.preventDefault()
            const values = new FormData(event.currentTarget)
            login({
              email: fieldValue(values, 'email'),
              password: fieldValue(values, 'password')
            })
              .then((account) => {
                setStatus(props.translate('auth.loginReady'))
                if (props.onLoginSuccess !== undefined) {
                  props.onLoginSuccess(account)
                }
              })
              .catch(() => setStatus(props.translate('auth.genericError')))
          }}
        >
          <div className="form-field">
            <Label htmlFor="login-email">{props.translate('auth.emailLabel')}</Label>
            <Input id="login-email" name="email" type="email" autoComplete="email" />
          </div>
          <div className="form-field">
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
    <Card className="auth-panel">
      <CardHeader>
        <h2>{props.translate('auth.signupTitle')}</h2>
        <CardDescription>{props.translate('auth.emailInstruction')}</CardDescription>
      </CardHeader>
      <CardContent className="auth-content">
        <form
          className="auth-form"
          onSubmit={(event) => {
            event.preventDefault()
            const values = new FormData(event.currentTarget)
            signupOwner({
              email: fieldValue(values, 'email'),
              password: fieldValue(values, 'password'),
              bandName: fieldValue(values, 'bandName'),
              bandTimezone: fieldValue(values, 'bandTimezone')
            })
              .then((account) => {
                if (account === null) {
                  setStatus(props.translate('auth.emailVerificationRequired'))
                  return
                }

                setStatus(props.translate('auth.signupCreated'))
              })
              .catch(() => setStatus(props.translate('auth.genericError')))
          }}
        >
          <div className="form-field">
            <Label htmlFor="signup-email">{props.translate('auth.emailLabel')}</Label>
            <Input id="signup-email" name="email" type="email" autoComplete="email" />
          </div>
          <div className="form-field">
            <Label htmlFor="signup-password">{props.translate('auth.passwordLabel')}</Label>
            <Input
              id="signup-password"
              name="password"
              type="password"
              autoComplete="new-password"
            />
          </div>
          <div className="form-field">
            <Label htmlFor="signup-band-name">{props.translate('auth.bandNameLabel')}</Label>
            <Input id="signup-band-name" name="bandName" type="text" autoComplete="organization" />
          </div>
          <div className="form-field">
            <Label htmlFor="signup-band-timezone">{props.translate('auth.timezoneLabel')}</Label>
            <Input id="signup-band-timezone" name="bandTimezone" type="text" autoComplete="off" />
          </div>
          <Button type="submit">{props.translate('auth.signupSubmit')}</Button>
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
