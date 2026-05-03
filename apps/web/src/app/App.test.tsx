import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { App } from './App'

const supabaseMock = vi.hoisted(() => {
  return {
    getSession: vi.fn(),
    signInWithPassword: vi.fn(),
    signUp: vi.fn(),
    unsubscribe: vi.fn()
  }
})

vi.mock('@supabase/supabase-js', () => {
  return {
    createClient: () => ({
      auth: {
        getSession: supabaseMock.getSession,
        signInWithPassword: supabaseMock.signInWithPassword,
        signUp: supabaseMock.signUp,
        onAuthStateChange: () => ({
          data: {
            subscription: {
              unsubscribe: supabaseMock.unsubscribe
            }
          }
        })
      }
    })
  }
})

describe('App', () => {
  beforeEach(() => {
    window.history.pushState({}, '', '/')
    supabaseMock.getSession.mockReset()
    supabaseMock.signInWithPassword.mockReset()
    supabaseMock.signUp.mockReset()
    supabaseMock.unsubscribe.mockReset()
    supabaseMock.getSession.mockResolvedValue({ data: { session: null } })
    vi.stubGlobal('fetch', vi.fn(accountFetch))
    vi.stubGlobal('crypto', {
      randomUUID: () => 'test-idempotency-key'
    })
    vi.stubGlobal('navigator', {
      language: 'en-US',
      clipboard: {
        writeText: vi.fn().mockResolvedValue(undefined)
      }
    })
  })

  it('renders the translated navigation', () => {
    supabaseMock.getSession.mockReturnValue(new Promise(() => undefined))

    render(<App />)

    expect(screen.getByRole('heading', { name: 'Band Manager' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Inventory/i })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Merch Booth/i })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Account/i })).toBeInTheDocument()
  })

  it('renders the owner signup form on the signup route', async () => {
    window.history.pushState({}, '', '/signup')

    render(<App />)

    expect(await screen.findByRole('heading', { name: 'Create band account' })).toBeInTheDocument()
    expect(screen.getByText(/really_awesome_band@email.com/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Create owner account' })).toBeInTheDocument()
  })

  it('renders account members and invites for an owner', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    window.history.pushState({}, '', '/account')

    render(<App />)

    expect(await screen.findByRole('heading', { name: 'Account' })).toBeInTheDocument()
    expect(await screen.findByText('owner@example.com')).toBeInTheDocument()
    expect(screen.getByText('viewer@example.com')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Create invite' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Revoke' })).toBeInTheDocument()
  })

  it('hides invite mutation controls for a viewer', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    vi.stubGlobal('fetch', vi.fn(viewerAccountFetch))
    window.history.pushState({}, '', '/account')

    render(<App />)

    expect(await screen.findByRole('heading', { name: 'Account' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Create invite' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Revoke' })).not.toBeInTheDocument()
  })

  it('shows a copyable invite link after owner invite creation', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    window.history.pushState({}, '', '/account')

    render(<App />)

    const emailInput = await screen.findByLabelText('Viewer email')
    fireEvent.change(emailInput, { target: { value: 'new-viewer@example.com' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create invite' }))

    expect(await screen.findByText(/token_new_viewer/i)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Copy invite link' }))

    await waitFor(() => {
      expect(screen.getByText('Invite link copied.')).toBeInTheDocument()
    })
  })

  it('accepts an invite token after an authenticated session is available', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    window.history.pushState({}, '', '/account/invites/accept?token=token_accept')

    render(<App />)

    expect(await screen.findByText(/Invite accepted for viewer@example.com/i)).toBeInTheDocument()
  })

  it('preserves an invite token through login before accepting', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    supabaseMock.getSession.mockResolvedValueOnce({ data: { session: null } })
    supabaseMock.signInWithPassword.mockResolvedValue({
      data: {
        session: {
          access_token: 'access-token'
        }
      },
      error: null
    })
    window.history.pushState({}, '', '/account/invites/accept?token=token_accept')

    render(<App />)

    expect(
      await screen.findByText('Log in with the invited email to accept this invite.')
    ).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('Email'), { target: { value: 'viewer@example.com' } })
    fireEvent.change(screen.getByLabelText('Password'), { target: { value: 'password-1' } })
    fireEvent.click(screen.getByRole('button', { name: 'Log in' }))

    expect(await screen.findByText(/Invite accepted for viewer@example.com/i)).toBeInTheDocument()
  })
})

function authenticatedSession() {
  return {
    data: {
      session: {
        access_token: 'access-token',
        user: {
          email: 'owner@example.com'
        }
      }
    }
  }
}

async function accountFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  const url = requestURL(input)
  const method = init?.method ?? 'GET'

  if (url.endsWith('/me')) {
    return jsonResponse(currentAccountResponse('owner'), 200)
  }

  if (url.endsWith('/account/members')) {
    return jsonResponse(
      {
        members: [
          {
            userId: '00000000-0000-0000-0000-000000000001',
            email: 'owner@example.com',
            bandId: '00000000-0000-0000-0000-000000000002',
            role: 'owner',
            joinedAt: '2026-05-01T12:00:00Z'
          }
        ]
      },
      200
    )
  }

  if (url.endsWith('/account/invites') && method === 'GET') {
    return jsonResponse(
      {
        invites: [
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
      },
      200
    )
  }

  if (url.endsWith('/account/invites') && method === 'POST') {
    return jsonResponse(
      {
        id: '22222222-2222-2222-2222-222222222222',
        email: 'new-viewer@example.com',
        role: 'viewer',
        status: 'pending',
        expiresAt: '2026-05-08T12:00:00Z',
        createdAt: '2026-05-01T12:00:00Z',
        updatedAt: '2026-05-01T12:00:00Z',
        token: 'token_new_viewer'
      },
      201
    )
  }

  if (url.endsWith('/account/invites/accept') && method === 'POST') {
    return jsonResponse(
      {
        userId: '00000000-0000-0000-0000-000000000003',
        email: 'viewer@example.com',
        bandId: '00000000-0000-0000-0000-000000000002',
        role: 'viewer',
        joinedAt: '2026-05-01T12:00:00Z'
      },
      200
    )
  }

  if (url.endsWith('/account/invites/11111111-1111-1111-1111-111111111111/revoke')) {
    return jsonResponse(
      {
        id: '11111111-1111-1111-1111-111111111111',
        email: 'viewer@example.com',
        role: 'viewer',
        status: 'revoked',
        expiresAt: '2026-05-08T12:00:00Z',
        createdAt: '2026-05-01T12:00:00Z',
        updatedAt: '2026-05-01T12:00:00Z'
      },
      200
    )
  }

  return jsonResponse({ message: 'not found' }, 404)
}

async function viewerAccountFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  const url = requestURL(input)
  if (url.endsWith('/me')) {
    return jsonResponse(currentAccountResponse('viewer'), 200)
  }

  return accountFetch(input, init)
}

function currentAccountResponse(role: 'owner' | 'viewer') {
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

function requestURL(input: RequestInfo | URL): string {
  if (typeof input === 'string') {
    return input
  }

  if (input instanceof URL) {
    return input.toString()
  }

  return input.url
}

function jsonResponse(body: unknown, status: number): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: {
      'Content-Type': 'application/json'
    }
  })
}
