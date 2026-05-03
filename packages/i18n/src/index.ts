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
  | 'account.acceptLoading'
  | 'account.acceptLoginPrompt'
  | 'account.acceptMissingToken'
  | 'account.acceptSuccess'
  | 'account.acceptTitle'
  | 'account.actionsHeader'
  | 'account.copyFailed'
  | 'account.copyInviteLink'
  | 'account.copySuccess'
  | 'account.createInviteSubmit'
  | 'account.createInviteTitle'
  | 'account.emailHeader'
  | 'account.emailLabel'
  | 'account.expiresAtHeader'
  | 'account.genericError'
  | 'account.inviteCreated'
  | 'account.inviteEmailInvalid'
  | 'account.invitesTitle'
  | 'account.inviteStatus.accepted'
  | 'account.inviteStatus.expired'
  | 'account.inviteStatus.pending'
  | 'account.inviteStatus.revoked'
  | 'account.joinedAtHeader'
  | 'account.loading'
  | 'account.loginRequired'
  | 'account.membersTitle'
  | 'account.noInvites'
  | 'account.noMembers'
  | 'account.revokeInvite'
  | 'account.role.admin'
  | 'account.role.member'
  | 'account.role.owner'
  | 'account.role.viewer'
  | 'account.roleHeader'
  | 'account.statusHeader'
  | 'account.title'
  | 'nav.inventory'
  | 'nav.merchBooth'
  | 'nav.reports'
  | 'nav.calendar'
  | 'nav.account'
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
    'account.acceptLoading': 'Accepting invite...',
    'account.acceptLoginPrompt': 'Log in with the invited email to accept this invite.',
    'account.acceptMissingToken': 'Invite token is missing.',
    'account.acceptSuccess': 'Invite accepted for',
    'account.acceptTitle': 'Accept invite',
    'account.actionsHeader': 'Actions',
    'account.copyFailed': 'Invite link could not be copied.',
    'account.copyInviteLink': 'Copy invite link',
    'account.copySuccess': 'Invite link copied.',
    'account.createInviteSubmit': 'Create invite',
    'account.createInviteTitle': 'Invite viewer',
    'account.emailHeader': 'Email',
    'account.emailLabel': 'Viewer email',
    'account.expiresAtHeader': 'Expires',
    'account.genericError': 'Account request failed.',
    'account.inviteCreated': 'Invite created.',
    'account.inviteEmailInvalid': 'Enter a valid email address.',
    'account.invitesTitle': 'Invites',
    'account.inviteStatus.accepted': 'Accepted',
    'account.inviteStatus.expired': 'Expired',
    'account.inviteStatus.pending': 'Pending',
    'account.inviteStatus.revoked': 'Revoked',
    'account.joinedAtHeader': 'Joined',
    'account.loading': 'Loading account...',
    'account.loginRequired': 'Log in to manage account access.',
    'account.membersTitle': 'Members',
    'account.noInvites': 'No invites yet.',
    'account.noMembers': 'No members yet.',
    'account.revokeInvite': 'Revoke',
    'account.role.admin': 'Admin',
    'account.role.member': 'Member',
    'account.role.owner': 'Owner',
    'account.role.viewer': 'Viewer',
    'account.roleHeader': 'Role',
    'account.statusHeader': 'Status',
    'account.title': 'Account',
    'nav.inventory': 'Inventory',
    'nav.merchBooth': 'Merch Booth',
    'nav.reports': 'Reports',
    'nav.calendar': 'Calendar',
    'nav.account': 'Account',
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
    'account.acceptLoading': 'Aceitando convite...',
    'account.acceptLoginPrompt': 'Entre com o email convidado para aceitar este convite.',
    'account.acceptMissingToken': 'O token do convite esta ausente.',
    'account.acceptSuccess': 'Convite aceito para',
    'account.acceptTitle': 'Aceitar convite',
    'account.actionsHeader': 'Acoes',
    'account.copyFailed': 'Nao foi possivel copiar o link do convite.',
    'account.copyInviteLink': 'Copiar link do convite',
    'account.copySuccess': 'Link do convite copiado.',
    'account.createInviteSubmit': 'Criar convite',
    'account.createInviteTitle': 'Convidar visualizador',
    'account.emailHeader': 'Email',
    'account.emailLabel': 'Email do visualizador',
    'account.expiresAtHeader': 'Expira',
    'account.genericError': 'A requisicao de conta falhou.',
    'account.inviteCreated': 'Convite criado.',
    'account.inviteEmailInvalid': 'Informe um email valido.',
    'account.invitesTitle': 'Convites',
    'account.inviteStatus.accepted': 'Aceito',
    'account.inviteStatus.expired': 'Expirado',
    'account.inviteStatus.pending': 'Pendente',
    'account.inviteStatus.revoked': 'Revogado',
    'account.joinedAtHeader': 'Entrada',
    'account.loading': 'Carregando conta...',
    'account.loginRequired': 'Entre para gerenciar o acesso da conta.',
    'account.membersTitle': 'Membros',
    'account.noInvites': 'Nenhum convite ainda.',
    'account.noMembers': 'Nenhum membro ainda.',
    'account.revokeInvite': 'Revogar',
    'account.role.admin': 'Admin',
    'account.role.member': 'Membro',
    'account.role.owner': 'Dono',
    'account.role.viewer': 'Visualizador',
    'account.roleHeader': 'Papel',
    'account.statusHeader': 'Status',
    'account.title': 'Conta',
    'nav.inventory': 'Estoque',
    'nav.merchBooth': 'Banca',
    'nav.reports': 'Relatorios',
    'nav.calendar': 'Calendario',
    'nav.account': 'Conta',
    'status.backendReady': 'Base do backend pronta'
  }
}
