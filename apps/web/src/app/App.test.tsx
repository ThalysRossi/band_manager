import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { setMockCurrentAccountRole } from '../test/apiMocks'
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
    expect(screen.getAllByText('Inventory')).toHaveLength(2)
    expect(screen.getByRole('link', { name: /Inventory/i })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Merch Booth/i })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Account/i })).toBeInTheDocument()
  })

  it('redirects unauthenticated protected routes to login', async () => {
    window.history.pushState({}, '', '/merch-booth')

    render(<App />)

    expect(await screen.findByRole('heading', { name: 'Log in' })).toBeInTheDocument()

    await waitFor(() => {
      expect(window.location.pathname).toBe('/login')
      expect(window.location.search).toBe('?redirect=%2Fmerch-booth')
    })
  })

  it('renders protected workspace routes for authenticated users', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    window.history.pushState({}, '', '/merch-booth')

    render(<App />)

    expect(await screen.findByRole('heading', { name: 'Merch Booth' })).toBeInTheDocument()
    expect(screen.getAllByText('Merch Booth')).toHaveLength(3)
    expect(screen.getByText('Backend foundation is ready')).toBeInTheDocument()
    expect(await screen.findByText('Os Testes')).toBeInTheDocument()
    expect(screen.getByText('owner@example.com | Owner')).toBeInTheDocument()
  })

  it('returns to the requested protected route after login', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    supabaseMock.signInWithPassword.mockResolvedValue({
      data: {
        session: {
          access_token: 'access-token'
        }
      },
      error: null
    })
    window.history.pushState({}, '', '/login?redirect=%2Faccount')

    render(<App />)

    fireEvent.change(await screen.findByLabelText('Email'), {
      target: { value: 'owner@example.com' }
    })
    fireEvent.change(screen.getByLabelText('Password'), { target: { value: 'password-1' } })
    fireEvent.click(screen.getByRole('button', { name: 'Log in' }))

    expect(await screen.findByRole('heading', { name: 'Account' })).toBeInTheDocument()
    expect(await screen.findByText('owner@example.com')).toBeInTheDocument()

    await waitFor(() => {
      expect(window.location.pathname).toBe('/account')
    })
  })

  it('renders the owner signup form on the signup route', async () => {
    window.history.pushState({}, '', '/signup')

    render(<App />)

    expect(await screen.findByRole('heading', { name: 'Create band account' })).toBeInTheDocument()
    expect(screen.getAllByText('Create band account')).toHaveLength(2)
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
    setMockCurrentAccountRole('viewer')
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
