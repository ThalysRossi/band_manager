export type Locale = 'en' | 'pt-BR'

export type TranslationKey =
  | 'app.title'
  | 'auth.bandNameLabel'
  | 'auth.emailInstruction'
  | 'auth.emailLabel'
  | 'auth.loginSubmit'
  | 'auth.loginTitle'
  | 'auth.passwordReset'
  | 'auth.passwordLabel'
  | 'auth.signupSubmit'
  | 'auth.signupTitle'
  | 'auth.timezoneLabel'
  | 'auth.emailVerificationRequired'
  | 'auth.genericError'
  | 'auth.signupCreated'
  | 'auth.loginReady'
  | 'nav.inventory'
  | 'nav.merchBooth'
  | 'nav.reports'
  | 'nav.calendar'
  | 'status.backendReady'

export type TranslationDictionary = Record<TranslationKey, string>

export const translations: Record<Locale, TranslationDictionary> = {
  en: {
    'app.title': 'Band Manager',
    'auth.bandNameLabel': 'Band name',
    'auth.emailInstruction': 'Use a band-related email, for example really_awesome_band@email.com.',
    'auth.emailLabel': 'Email',
    'auth.loginSubmit': 'Log in',
    'auth.loginTitle': 'Log in',
    'auth.passwordLabel': 'Password',
    'auth.passwordReset': 'Reset password',
    'auth.signupSubmit': 'Create owner account',
    'auth.signupTitle': 'Create band account',
    'auth.timezoneLabel': 'Band timezone',
    'auth.emailVerificationRequired':
      'Check your email, verify the account, then log in to finish setup.',
    'auth.genericError': 'Authentication failed. Check the fields and try again.',
    'auth.signupCreated': 'Owner account created.',
    'auth.loginReady': 'Login successful.',
    'nav.inventory': 'Inventory',
    'nav.merchBooth': 'Merch Booth',
    'nav.reports': 'Reports',
    'nav.calendar': 'Calendar',
    'status.backendReady': 'Backend foundation is ready'
  },
  'pt-BR': {
    'app.title': 'Band Manager',
    'auth.bandNameLabel': 'Nome da banda',
    'auth.emailInstruction':
      'Use um email relacionado a banda, por exemplo really_awesome_band@email.com.',
    'auth.emailLabel': 'Email',
    'auth.loginSubmit': 'Entrar',
    'auth.loginTitle': 'Entrar',
    'auth.passwordLabel': 'Senha',
    'auth.passwordReset': 'Redefinir senha',
    'auth.signupSubmit': 'Criar conta de dono',
    'auth.signupTitle': 'Criar conta da banda',
    'auth.timezoneLabel': 'Fuso horario da banda',
    'auth.emailVerificationRequired':
      'Verifique seu email e depois entre para concluir a configuracao.',
    'auth.genericError': 'A autenticacao falhou. Confira os campos e tente novamente.',
    'auth.signupCreated': 'Conta de dono criada.',
    'auth.loginReady': 'Login realizado.',
    'nav.inventory': 'Estoque',
    'nav.merchBooth': 'Banca',
    'nav.reports': 'Relatorios',
    'nav.calendar': 'Calendario',
    'status.backendReady': 'Base do backend pronta'
  }
}
