import type { TranslationKey } from 'i18n'

type Translate = (key: TranslationKey) => string

type AuthPageProps = {
  translate: Translate
}

export function LoginPage(props: AuthPageProps) {
  return (
    <div className="auth-panel">
      <h2>{props.translate('auth.loginTitle')}</h2>
      <form>
        <label>
          <span>{props.translate('auth.emailLabel')}</span>
          <input name="email" type="email" autoComplete="email" />
        </label>
        <button type="submit">{props.translate('auth.loginSubmit')}</button>
        <a href="/password-reset">{props.translate('auth.passwordReset')}</a>
      </form>
    </div>
  )
}

export function SignupPage(props: AuthPageProps) {
  return (
    <div className="auth-panel">
      <h2>{props.translate('auth.signupTitle')}</h2>
      <p>{props.translate('auth.emailInstruction')}</p>
      <form>
        <label>
          <span>{props.translate('auth.emailLabel')}</span>
          <input name="email" type="email" autoComplete="email" />
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
    </div>
  )
}
