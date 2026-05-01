import { useState } from 'react'
import type { TranslationKey } from 'i18n'

import { login, signupOwner } from './api'

type Translate = (key: TranslationKey) => string

type AuthPageProps = {
  translate: Translate
}

export function LoginPage(props: AuthPageProps) {
  const [status, setStatus] = useState<string>('')

  return (
    <div className="auth-panel">
      <h2>{props.translate('auth.loginTitle')}</h2>
      <form
        onSubmit={(event) => {
          event.preventDefault()
          const values = new FormData(event.currentTarget)
          login({
            email: fieldValue(values, 'email'),
            password: fieldValue(values, 'password')
          })
            .then(() => setStatus(props.translate('auth.loginReady')))
            .catch(() => setStatus(props.translate('auth.genericError')))
        }}
      >
        <label>
          <span>{props.translate('auth.emailLabel')}</span>
          <input name="email" type="email" autoComplete="email" />
        </label>
        <label>
          <span>{props.translate('auth.passwordLabel')}</span>
          <input name="password" type="password" autoComplete="current-password" />
        </label>
        <button type="submit">{props.translate('auth.loginSubmit')}</button>
        <a href="/password-reset">{props.translate('auth.passwordReset')}</a>
      </form>
      {status === '' ? null : <p role="status">{status}</p>}
    </div>
  )
}

export function SignupPage(props: AuthPageProps) {
  const [status, setStatus] = useState<string>('')

  return (
    <div className="auth-panel">
      <h2>{props.translate('auth.signupTitle')}</h2>
      <p>{props.translate('auth.emailInstruction')}</p>
      <form
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
        <label>
          <span>{props.translate('auth.emailLabel')}</span>
          <input name="email" type="email" autoComplete="email" />
        </label>
        <label>
          <span>{props.translate('auth.passwordLabel')}</span>
          <input name="password" type="password" autoComplete="new-password" />
        </label>
        <label>
          <span>{props.translate('auth.bandNameLabel')}</span>
          <input name="bandName" type="text" autoComplete="organization" />
        </label>
        <label>
          <span>{props.translate('auth.timezoneLabel')}</span>
          <input name="bandTimezone" type="text" autoComplete="off" />
        </label>
        <button type="submit">{props.translate('auth.signupSubmit')}</button>
      </form>
      {status === '' ? null : <p role="status">{status}</p>}
    </div>
  )
}

function fieldValue(values: FormData, fieldName: string): string {
  const value = values.get(fieldName)
  if (typeof value !== 'string') {
    return ''
  }

  return value
}
