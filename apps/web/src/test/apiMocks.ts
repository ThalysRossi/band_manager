import { HttpResponse, http } from 'msw'

import type { AccountInvite, AccountMember, Role } from '../features/account/api'
import type { CurrentAccountResponse } from '../features/auth/api'

type MockCurrentAccountRole = Extract<Role, 'owner' | 'viewer'>

type MockAPIState = {
  currentAccountRole: MockCurrentAccountRole
}

const apiBaseURL = 'http://localhost:8080'

const mockAPIState: MockAPIState = {
  currentAccountRole: 'owner'
}

export const apiHandlers = [
  http.get(`${apiBaseURL}/me`, () => {
    return HttpResponse.json(currentAccountResponse(mockAPIState.currentAccountRole), {
      status: 200
    })
  }),
  http.get(`${apiBaseURL}/account/members`, () => {
    return HttpResponse.json({ members: accountMembers() }, { status: 200 })
  }),
  http.get(`${apiBaseURL}/account/invites`, () => {
    return HttpResponse.json({ invites: accountInvites() }, { status: 200 })
  }),
  http.post(`${apiBaseURL}/account/invites`, () => {
    return HttpResponse.json(createdInvite(), { status: 201 })
  }),
  http.post(`${apiBaseURL}/account/invites/accept`, () => {
    return HttpResponse.json(acceptedMember(), { status: 200 })
  }),
  http.post(`${apiBaseURL}/account/invites/:inviteId/revoke`, () => {
    return HttpResponse.json(revokedInvite(), { status: 200 })
  })
]

export function resetAPIMocks(): void {
  mockAPIState.currentAccountRole = 'owner'
}

export function setMockCurrentAccountRole(role: MockCurrentAccountRole): void {
  mockAPIState.currentAccountRole = role
}

function currentAccountResponse(role: MockCurrentAccountRole): CurrentAccountResponse {
  return {
    user: {
      id: '00000000-0000-0000-0000-000000000001',
      email: 'owner@example.com'
    },
    activeBand: {
      bandId: '00000000-0000-0000-0000-000000000002',
      bandName: 'Os Testes',
      role,
      canWrite: role === 'owner'
    }
  }
}

function accountMembers(): AccountMember[] {
  return [
    {
      userId: '00000000-0000-0000-0000-000000000001',
      email: 'owner@example.com',
      bandId: '00000000-0000-0000-0000-000000000002',
      role: 'owner',
      joinedAt: '2026-05-01T12:00:00Z'
    }
  ]
}

function accountInvites(): AccountInvite[] {
  return [
    {
      id: '11111111-1111-1111-1111-111111111111',
      email: 'viewer@example.com',
      role: 'viewer',
      status: 'pending',
      expiresAt: '2026-05-08T12:00:00Z',
      createdAt: '2026-05-01T12:00:00Z',
      updatedAt: '2026-05-01T12:00:00Z'
    }
  ]
}

function createdInvite(): AccountInvite {
  return {
    id: '22222222-2222-2222-2222-222222222222',
    email: 'new-viewer@example.com',
    role: 'viewer',
    status: 'pending',
    expiresAt: '2026-05-08T12:00:00Z',
    createdAt: '2026-05-01T12:00:00Z',
    updatedAt: '2026-05-01T12:00:00Z',
    token: 'token_new_viewer'
  }
}

function acceptedMember(): AccountMember {
  return {
    userId: '00000000-0000-0000-0000-000000000003',
    email: 'viewer@example.com',
    bandId: '00000000-0000-0000-0000-000000000002',
    role: 'viewer',
    joinedAt: '2026-05-01T12:00:00Z'
  }
}

function revokedInvite(): AccountInvite {
  return {
    id: '11111111-1111-1111-1111-111111111111',
    email: 'viewer@example.com',
    role: 'viewer',
    status: 'revoked',
    expiresAt: '2026-05-08T12:00:00Z',
    createdAt: '2026-05-01T12:00:00Z',
    updatedAt: '2026-05-01T12:00:00Z'
  }
}
