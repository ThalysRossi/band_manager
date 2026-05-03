import { apiRequest } from '../../shared/api/client'

export type Role = 'owner' | 'admin' | 'member' | 'viewer'

export type InviteStatus = 'pending' | 'accepted' | 'revoked' | 'expired'

export type AccountMember = {
  userId: string
  email: string
  bandId: string
  role: Role
  joinedAt: string
}

export type AccountInvite = {
  id: string
  email: string
  role: Role
  status: InviteStatus
  expiresAt: string
  createdAt: string
  updatedAt: string
  token?: string
}

type MembersResponse = {
  members: AccountMember[]
}

type InvitesResponse = {
  invites: AccountInvite[]
}

export async function listAccountMembers(accessToken: string): Promise<AccountMember[]> {
  const response = await apiRequest<MembersResponse>({
    accessToken,
    path: '/account/members',
    method: 'GET',
    body: null,
    idempotent: false
  })

  return response.members
}

export async function listAccountInvites(accessToken: string): Promise<AccountInvite[]> {
  const response = await apiRequest<InvitesResponse>({
    accessToken,
    path: '/account/invites',
    method: 'GET',
    body: null,
    idempotent: false
  })

  return response.invites
}

export async function createAccountInvite(
  accessToken: string,
  email: string
): Promise<AccountInvite> {
  return apiRequest<AccountInvite>({
    accessToken,
    path: '/account/invites',
    method: 'POST',
    body: { email },
    idempotent: true
  })
}

export async function revokeAccountInvite(
  accessToken: string,
  inviteId: string
): Promise<AccountInvite> {
  return apiRequest<AccountInvite>({
    accessToken,
    path: `/account/invites/${inviteId}/revoke`,
    method: 'POST',
    body: {},
    idempotent: true
  })
}

export async function acceptAccountInvite(
  accessToken: string,
  token: string
): Promise<AccountMember> {
  return apiRequest<AccountMember>({
    accessToken,
    path: '/account/invites/accept',
    method: 'POST',
    body: { token },
    idempotent: true
  })
}
