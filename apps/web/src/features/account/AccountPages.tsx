import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { TranslationKey } from 'i18n'

import { LoginPage } from '../auth/AuthPages'
import { getCurrentAccount } from '../auth/api'
import type { CurrentAccountResponse } from '../auth/api'
import { useAuthSession } from '../../shared/auth/session'
import {
  acceptAccountInvite,
  createAccountInvite,
  listAccountInvites,
  listAccountMembers,
  revokeAccountInvite
} from './api'
import type { AccountInvite, AccountMember, InviteStatus, Role } from './api'

type Translate = (key: TranslationKey) => string

type AccountPageProps = {
  translate: Translate
}

type AcceptInvitePageProps = {
  translate: Translate
  token: string
}

type AccountOverview = {
  members: AccountMember[]
  invites: AccountInvite[]
}

export function AccountPage(props: AccountPageProps) {
  const session = useAuthSession()

  if (session.state.status === 'loading') {
    return <StatusPanel message={props.translate('account.loading')} />
  }

  if (session.state.status === 'unauthenticated') {
    return (
      <section className="account-layout">
        <WorkspaceTitle title={props.translate('account.title')} />
        <p className="account-muted">{props.translate('account.loginRequired')}</p>
        <LoginPage translate={props.translate} onLoginSuccess={() => void session.refresh()} />
      </section>
    )
  }

  return (
    <AuthenticatedAccountPage accessToken={session.state.accessToken} translate={props.translate} />
  )
}

export function AcceptInvitePage(props: AcceptInvitePageProps) {
  const session = useAuthSession()
  const [acceptedMember, setAcceptedMember] = useState<AccountMember | null>(null)
  const [acceptError, setAcceptError] = useState<string>('')
  const [attemptedToken, setAttemptedToken] = useState<string>('')
  const queryClient = useQueryClient()

  const acceptMutation = useMutation({
    mutationFn: (input: { accessToken: string; token: string }) =>
      acceptAccountInvite(input.accessToken, input.token),
    onSuccess: (member) => {
      setAcceptedMember(member)
      setAcceptError('')
      void queryClient.invalidateQueries({ queryKey: ['account'] })
    },
    onError: (error) => {
      setAcceptedMember(null)
      setAcceptError(
        error instanceof Error ? error.message : props.translate('account.genericError')
      )
    }
  })

  useEffect(() => {
    if (props.token.trim() === '') {
      return
    }

    if (session.state.status !== 'authenticated') {
      return
    }

    if (attemptedToken === props.token) {
      return
    }

    setAttemptedToken(props.token)
    acceptMutation.mutate({ accessToken: session.state.accessToken, token: props.token })
  }, [acceptMutation, attemptedToken, props.token, session.state])

  if (props.token.trim() === '') {
    return (
      <section className="account-layout">
        <WorkspaceTitle title={props.translate('account.acceptTitle')} />
        <StatusPanel message={props.translate('account.acceptMissingToken')} />
      </section>
    )
  }

  if (session.state.status === 'loading') {
    return <StatusPanel message={props.translate('account.loading')} />
  }

  if (session.state.status === 'unauthenticated') {
    return (
      <section className="account-layout">
        <WorkspaceTitle title={props.translate('account.acceptTitle')} />
        <p className="account-muted">{props.translate('account.acceptLoginPrompt')}</p>
        <LoginPage translate={props.translate} onLoginSuccess={() => void session.refresh()} />
      </section>
    )
  }

  if (acceptedMember !== null) {
    return (
      <section className="account-layout">
        <WorkspaceTitle title={props.translate('account.acceptTitle')} />
        <StatusPanel
          message={`${props.translate('account.acceptSuccess')} ${acceptedMember.email}`}
        />
      </section>
    )
  }

  return (
    <section className="account-layout">
      <WorkspaceTitle title={props.translate('account.acceptTitle')} />
      <StatusPanel
        message={
          acceptError === ''
            ? props.translate('account.acceptLoading')
            : `${props.translate('account.genericError')} ${acceptError}`
        }
      />
    </section>
  )
}

function AuthenticatedAccountPage(props: { accessToken: string; translate: Translate }) {
  const [createdInvite, setCreatedInvite] = useState<AccountInvite | null>(null)
  const [formStatus, setFormStatus] = useState<string>('')
  const [copyStatus, setCopyStatus] = useState<string>('')
  const queryClient = useQueryClient()

  const accountQuery = useQuery({
    queryKey: ['account', 'current', props.accessToken],
    queryFn: () => getCurrentAccount(props.accessToken)
  })

  const overviewQuery = useQuery({
    queryKey: ['account', 'overview', props.accessToken],
    queryFn: async (): Promise<AccountOverview> => {
      const [members, invites] = await Promise.all([
        listAccountMembers(props.accessToken),
        listAccountInvites(props.accessToken)
      ])

      return { members, invites }
    }
  })

  const createInviteMutation = useMutation({
    mutationFn: (email: string) => createAccountInvite(props.accessToken, email),
    onSuccess: (invite) => {
      setCreatedInvite(invite)
      setFormStatus(props.translate('account.inviteCreated'))
      void queryClient.invalidateQueries({ queryKey: ['account', 'overview', props.accessToken] })
    },
    onError: (error) => {
      setCreatedInvite(null)
      setFormStatus(
        error instanceof Error ? error.message : props.translate('account.genericError')
      )
    }
  })

  const revokeInviteMutation = useMutation({
    mutationFn: (inviteId: string) => revokeAccountInvite(props.accessToken, inviteId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['account', 'overview', props.accessToken] })
    }
  })

  const account = accountQuery.data
  const overview = overviewQuery.data
  const ownerCanManage = account?.activeBand.role === 'owner'
  const pendingInviteLink = useMemo(() => {
    if (createdInvite?.token === undefined) {
      return ''
    }

    return inviteLink(createdInvite.token)
  }, [createdInvite])

  if (accountQuery.isLoading || overviewQuery.isLoading) {
    return <StatusPanel message={props.translate('account.loading')} />
  }

  if (
    accountQuery.isError ||
    overviewQuery.isError ||
    account === undefined ||
    overview === undefined
  ) {
    return <StatusPanel message={props.translate('account.genericError')} />
  }

  return (
    <section className="account-layout">
      <WorkspaceTitle title={props.translate('account.title')} />

      {ownerCanManage ? (
        <section className="account-section" aria-labelledby="account-create-invite-title">
          <h3 id="account-create-invite-title">{props.translate('account.createInviteTitle')}</h3>
          <form
            className="account-inline-form"
            onSubmit={(event) => {
              event.preventDefault()
              const values = new FormData(event.currentTarget)
              const email = fieldValue(values, 'email')
              if (!email.includes('@')) {
                setFormStatus(props.translate('account.inviteEmailInvalid'))
                return
              }

              createInviteMutation.mutate(email)
            }}
          >
            <label>
              <span>{props.translate('account.emailLabel')}</span>
              <input name="email" type="email" autoComplete="email" />
            </label>
            <button type="submit">{props.translate('account.createInviteSubmit')}</button>
          </form>
          {formStatus === '' ? null : <p role="status">{formStatus}</p>}
          {pendingInviteLink === '' ? null : (
            <div className="account-token-row">
              <code>{pendingInviteLink}</code>
              <button
                type="button"
                onClick={() => {
                  navigator.clipboard
                    .writeText(pendingInviteLink)
                    .then(() => setCopyStatus(props.translate('account.copySuccess')))
                    .catch(() => setCopyStatus(props.translate('account.copyFailed')))
                }}
              >
                {props.translate('account.copyInviteLink')}
              </button>
            </div>
          )}
          {copyStatus === '' ? null : <p role="status">{copyStatus}</p>}
        </section>
      ) : null}

      <section className="account-section" aria-labelledby="account-members-title">
        <h3 id="account-members-title">{props.translate('account.membersTitle')}</h3>
        <MembersTable members={overview.members} translate={props.translate} />
      </section>

      <section className="account-section" aria-labelledby="account-invites-title">
        <h3 id="account-invites-title">{props.translate('account.invitesTitle')}</h3>
        <InvitesTable
          invites={overview.invites}
          canManage={ownerCanManage}
          translate={props.translate}
          revokingInviteID={revokeInviteMutation.variables ?? ''}
          onRevoke={(inviteId) => revokeInviteMutation.mutate(inviteId)}
        />
      </section>
    </section>
  )
}

function MembersTable(props: { members: AccountMember[]; translate: Translate }) {
  if (props.members.length === 0) {
    return <p className="account-muted">{props.translate('account.noMembers')}</p>
  }

  return (
    <div className="account-table-wrap">
      <table className="account-table">
        <thead>
          <tr>
            <th>{props.translate('account.emailHeader')}</th>
            <th>{props.translate('account.roleHeader')}</th>
            <th>{props.translate('account.joinedAtHeader')}</th>
          </tr>
        </thead>
        <tbody>
          {props.members.map((member) => (
            <tr key={member.userId}>
              <td>{member.email}</td>
              <td>{props.translate(roleLabelKey(member.role))}</td>
              <td>{formatDate(member.joinedAt)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function InvitesTable(props: {
  invites: AccountInvite[]
  canManage: boolean
  translate: Translate
  revokingInviteID: string
  onRevoke: (inviteId: string) => void
}) {
  if (props.invites.length === 0) {
    return <p className="account-muted">{props.translate('account.noInvites')}</p>
  }

  return (
    <div className="account-table-wrap">
      <table className="account-table">
        <thead>
          <tr>
            <th>{props.translate('account.emailHeader')}</th>
            <th>{props.translate('account.roleHeader')}</th>
            <th>{props.translate('account.statusHeader')}</th>
            <th>{props.translate('account.expiresAtHeader')}</th>
            {props.canManage ? <th>{props.translate('account.actionsHeader')}</th> : null}
          </tr>
        </thead>
        <tbody>
          {props.invites.map((invite) => (
            <tr key={invite.id}>
              <td>{invite.email}</td>
              <td>{props.translate(roleLabelKey(invite.role))}</td>
              <td>{props.translate(inviteStatusLabelKey(invite.status))}</td>
              <td>{formatDate(invite.expiresAt)}</td>
              {props.canManage ? (
                <td>
                  {invite.status === 'pending' ? (
                    <button
                      type="button"
                      onClick={() => props.onRevoke(invite.id)}
                      disabled={props.revokingInviteID === invite.id}
                    >
                      {props.translate('account.revokeInvite')}
                    </button>
                  ) : null}
                </td>
              ) : null}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function WorkspaceTitle(props: { title: string }) {
  return (
    <div className="workspace-header">
      <h2>{props.title}</h2>
    </div>
  )
}

function StatusPanel(props: { message: string }) {
  return (
    <div className="workspace-header">
      <p role="status">{props.message}</p>
    </div>
  )
}

function roleLabelKey(role: Role): TranslationKey {
  return `account.role.${role}`
}

function inviteStatusLabelKey(status: InviteStatus): TranslationKey {
  return `account.inviteStatus.${status}`
}

function inviteLink(token: string): string {
  const url = new URL('/account/invites/accept', window.location.origin)
  url.searchParams.set('token', token)

  return url.toString()
}

function formatDate(value: string): string {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short'
  }).format(new Date(value))
}

function fieldValue(values: FormData, fieldName: string): string {
  const value = values.get(fieldName)
  if (typeof value !== 'string') {
    return ''
  }

  return value.trim()
}
