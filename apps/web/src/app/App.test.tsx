import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { InventoryProduct } from '../features/inventory/api'
import {
  mockCreatedProductCount,
  setMockCurrentAccountExists,
  setMockCurrentAccountRole,
  setMockInventoryProducts,
  setMockPhotoUploadRequestFails,
  setMockStorageUploadFails
} from '../test/apiMocks'
import { App } from './App'

const supabaseMock = vi.hoisted(() => {
  return {
    getSession: vi.fn(),
    signInWithPassword: vi.fn(),
    signOut: vi.fn(),
    signUp: vi.fn(),
    unsubscribe: vi.fn()
  }
})

const photoProcessingMock = vi.hoisted(() => {
  return {
    processInventoryPhoto: vi.fn()
  }
})

vi.mock('@supabase/supabase-js', () => {
  return {
    createClient: () => ({
      auth: {
        getSession: supabaseMock.getSession,
        signInWithPassword: supabaseMock.signInWithPassword,
        signOut: supabaseMock.signOut,
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

vi.mock('../features/inventory/photoProcessing', () => photoProcessingMock)

describe('App', () => {
  beforeEach(() => {
    window.history.pushState({}, '', '/')
    supabaseMock.getSession.mockReset()
    supabaseMock.signInWithPassword.mockReset()
    supabaseMock.signOut.mockReset()
    supabaseMock.signUp.mockReset()
    supabaseMock.unsubscribe.mockReset()
    photoProcessingMock.processInventoryPhoto.mockReset()
    supabaseMock.getSession.mockResolvedValue({ data: { session: null } })
    supabaseMock.signOut.mockResolvedValue({ error: null })
    Object.defineProperty(URL, 'createObjectURL', {
      value: vi.fn(() => 'blob:inventory-preview'),
      configurable: true
    })
    Object.defineProperty(URL, 'revokeObjectURL', {
      value: vi.fn(),
      configurable: true
    })
    Object.defineProperty(window, 'confirm', {
      value: vi.fn(() => true),
      configurable: true
    })
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

  it('redirects authenticated users without an account to onboarding', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    setMockCurrentAccountExists(false)
    window.history.pushState({}, '', '/merch-booth')

    render(<App />)

    expect(await screen.findByRole('heading', { name: 'Set up your band' })).toBeInTheDocument()

    await waitFor(() => {
      expect(window.location.pathname).toBe('/onboarding')
    })
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

  it('renders the credential signup form on the signup route', async () => {
    window.history.pushState({}, '', '/signup')

    render(<App />)

    expect(await screen.findByRole('heading', { name: 'Create account' })).toBeInTheDocument()
    expect(screen.getAllByText('Create account')).toHaveLength(3)
    expect(screen.getByText(/Verify your email/i)).toBeInTheDocument()
    expect(screen.queryByLabelText('Band name')).not.toBeInTheDocument()
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

  it('renders the empty inventory state for an owner', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())

    render(<App />)

    expect(await screen.findByRole('heading', { name: 'Inventory' })).toBeInTheDocument()
    expect(await screen.findByText('No inventory products yet.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Create product' })).toBeInTheDocument()
  })

  it('renders inventory products', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    setMockInventoryProducts([mockInventoryProduct()])

    render(<App />)

    expect(await screen.findByText('Logo Shirt')).toBeInTheDocument()
    expect(screen.getAllByText('Shirt').length).toBeGreaterThan(0)
    expect(screen.getByText('In stock')).toBeInTheDocument()
  })

  it('hides inventory create controls for a viewer', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    setMockCurrentAccountRole('viewer')
    setMockInventoryProducts([mockInventoryProduct()])

    render(<App />)

    expect(await screen.findByRole('heading', { name: 'Inventory' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Create product' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Edit product/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Delete product/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Add variant/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Edit variant/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Delete variant/i })).not.toBeInTheDocument()
  })

  it('creates an inventory product with photo upload', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    photoProcessingMock.processInventoryPhoto.mockResolvedValue(processedInventoryPhoto())

    render(<App />)

    fireEvent.change(await screen.findByLabelText('Product name'), {
      target: { value: 'Logo Shirt' }
    })
    fireEvent.change(screen.getByLabelText('Photo'), {
      target: { files: [new File(['photo'], 'photo.jpg', { type: 'image/jpeg' })] }
    })
    fireEvent.change(screen.getByLabelText('Colour'), { target: { value: 'Black' } })
    fireEvent.change(screen.getByLabelText('Price (BRL)'), { target: { value: '50' } })
    fireEvent.change(screen.getByLabelText('Cost (BRL)'), { target: { value: '20' } })
    fireEvent.change(screen.getByLabelText('Quantity'), { target: { value: '2' } })

    expect(await screen.findByText(/Photo ready/i)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Create product' }))

    expect(await screen.findByText('Product created.')).toBeInTheDocument()
    expect(await screen.findByText('Logo Shirt')).toBeInTheDocument()
    expect(mockCreatedProductCount()).toBe(1)
  })

  it('does not create an inventory product when photo upload fails', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    photoProcessingMock.processInventoryPhoto.mockResolvedValue(processedInventoryPhoto())
    setMockStorageUploadFails(true)

    render(<App />)

    fireEvent.change(await screen.findByLabelText('Product name'), {
      target: { value: 'Broken Shirt' }
    })
    fireEvent.change(screen.getByLabelText('Photo'), {
      target: { files: [new File(['photo'], 'photo.jpg', { type: 'image/jpeg' })] }
    })

    expect(await screen.findByText(/Photo ready/i)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Create product' }))

    expect(await screen.findByText('Photo upload failed.')).toBeInTheDocument()
    expect(mockCreatedProductCount()).toBe(0)
  })

  it('does not create an inventory product when the photo upload request fails', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    photoProcessingMock.processInventoryPhoto.mockResolvedValue(processedInventoryPhoto())
    setMockPhotoUploadRequestFails(true)

    render(<App />)

    fireEvent.change(await screen.findByLabelText('Product name'), {
      target: { value: 'Broken Shirt' }
    })
    fireEvent.change(screen.getByLabelText('Photo'), {
      target: { files: [new File(['photo'], 'photo.jpg', { type: 'image/jpeg' })] }
    })

    expect(await screen.findByText(/Photo ready/i)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Create product' }))

    expect(
      await screen.findByText('Photo upload request failed. Restart the API and check VITE_API_BASE_URL.')
    ).toBeInTheDocument()
    expect(mockCreatedProductCount()).toBe(0)
  })

  it('updates inventory product metadata', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    setMockInventoryProducts([mockInventoryProduct()])

    render(<App />)

    fireEvent.click(await screen.findByRole('button', { name: 'Edit product Logo Shirt' }))
    fireEvent.change(screen.getByLabelText('Edit product name'), {
      target: { value: 'Tour Hoodie' }
    })
    fireEvent.change(screen.getByLabelText('Edit category'), {
      target: { value: 'hoodie' }
    })
    fireEvent.click(screen.getByRole('button', { name: 'Save product' }))

    expect(await screen.findByText('Product updated.')).toBeInTheDocument()
    expect(await screen.findByText('Tour Hoodie')).toBeInTheDocument()
    expect(screen.getAllByText('Hoodie').length).toBeGreaterThan(0)
  })

  it('deletes an inventory product after confirmation', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    setMockInventoryProducts([mockInventoryProduct()])

    render(<App />)

    fireEvent.click(await screen.findByRole('button', { name: 'Delete product Logo Shirt' }))

    expect(window.confirm).toHaveBeenCalledWith('Delete this product from inventory?')
    expect(await screen.findByText('Product deleted.')).toBeInTheDocument()
    expect(await screen.findByText('No inventory products yet.')).toBeInTheDocument()
  })

  it('updates an inventory variant', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    setMockInventoryProducts([mockInventoryProduct()])

    render(<App />)

    fireEvent.click(await screen.findByRole('button', { name: 'Edit variant Logo Shirt M / Black' }))
    fireEvent.change(screen.getByLabelText('Edit variant colour'), {
      target: { value: 'Red' }
    })
    fireEvent.change(screen.getByLabelText('Edit variant quantity'), { target: { value: '4' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save variant' }))

    expect(await screen.findByText('Variant updated.')).toBeInTheDocument()
    expect(await screen.findByText('M / Red')).toBeInTheDocument()
    expect(await screen.findByText('4 in stock')).toBeInTheDocument()
  })

  it('creates an inventory variant for an existing product', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    setMockInventoryProducts([mockInventoryProduct()])

    render(<App />)

    fireEvent.click(await screen.findByRole('button', { name: 'Add variant Logo Shirt' }))
    fireEvent.change(screen.getByLabelText('New variant size'), { target: { value: 'g' } })
    fireEvent.change(screen.getByLabelText('New variant colour'), { target: { value: 'Red' } })
    fireEvent.change(screen.getByLabelText('New variant price (BRL)'), { target: { value: '60' } })
    fireEvent.change(screen.getByLabelText('New variant cost (BRL)'), { target: { value: '25' } })
    fireEvent.change(screen.getByLabelText('New variant quantity'), { target: { value: '3' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create variant' }))

    expect(await screen.findByText('Variant created.')).toBeInTheDocument()
    expect(await screen.findByText('2 variants')).toBeInTheDocument()
    expect(await screen.findByText('G / Red')).toBeInTheDocument()
  })

  it('deletes an inventory variant after confirmation', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    setMockInventoryProducts([mockInventoryProductWithTwoVariants()])

    render(<App />)

    fireEvent.click(await screen.findByRole('button', { name: 'Delete variant Logo Shirt M / Black' }))

    expect(window.confirm).toHaveBeenCalledWith('Delete this variant from inventory?')
    expect(await screen.findByText('Variant deleted.')).toBeInTheDocument()
    expect(await screen.findByText('1 variant')).toBeInTheDocument()
  })

  it('does not allow deletion of a product final variant', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    setMockInventoryProducts([mockInventoryProduct()])

    render(<App />)

    expect(
      await screen.findByRole('button', { name: 'Delete variant Logo Shirt M / Black' })
    ).toBeDisabled()
  })

  it('logs out from the header account dropdown', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    window.history.pushState({}, '', '/merch-booth')

    render(<App />)

    fireEvent.keyDown(await screen.findByRole('button', { name: /Os Testes/i }), {
      key: 'Enter',
      code: 'Enter'
    })
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Log out' }))

    await waitFor(() => {
      expect(supabaseMock.signOut).toHaveBeenCalledTimes(1)
    })
    expect(await screen.findByRole('heading', { name: 'Log in' })).toBeInTheDocument()
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

function processedInventoryPhoto() {
  return {
    full: {
      blob: new Blob(['full'], { type: 'image/webp' }),
      contentType: 'image/webp',
      sizeBytes: 1024,
      width: 1200,
      height: 900
    },
    display: {
      blob: new Blob(['display'], { type: 'image/webp' }),
      contentType: 'image/webp',
      sizeBytes: 512,
      width: 1280,
      height: 960
    }
  }
}

function mockInventoryProduct(): InventoryProduct {
  return {
    id: '33333333-3333-3333-3333-333333333333',
    bandId: '00000000-0000-0000-0000-000000000002',
    name: 'Logo Shirt',
    category: 'shirt',
    photo: {
      full: {
        objectKey: 'bands/test/photo/full.webp',
        contentType: 'image/webp',
        sizeBytes: 1024,
        width: 1200,
        height: 900,
        publicUrl: 'https://storage.example/full-public.webp'
      },
      display: {
        objectKey: 'bands/test/photo/display.webp',
        contentType: 'image/webp',
        sizeBytes: 512,
        width: 1280,
        height: 960,
        publicUrl: 'https://storage.example/display-public.webp'
      }
    },
    variants: [
      {
        id: '44444444-4444-4444-4444-444444444444',
        productId: '33333333-3333-3333-3333-333333333333',
        size: 'm',
        colour: 'Black',
        price: { amount: 5000, currency: 'BRL' },
        cost: { amount: 2000, currency: 'BRL' },
        quantity: 2,
        soldOut: false,
        createdAt: '2026-05-01T12:00:00Z',
        updatedAt: '2026-05-01T12:00:00Z'
      }
    ],
    createdAt: '2026-05-01T12:00:00Z',
    updatedAt: '2026-05-01T12:00:00Z'
  }
}

function mockInventoryProductWithTwoVariants(): InventoryProduct {
  const product = mockInventoryProduct()
  return {
    ...product,
    variants: [
      ...product.variants,
      {
        id: '44444444-4444-4444-4444-444444444445',
        productId: product.id,
        size: 'g',
        colour: 'Black',
        price: { amount: 5000, currency: 'BRL' },
        cost: { amount: 2000, currency: 'BRL' },
        quantity: 2,
        soldOut: false,
        createdAt: '2026-05-01T12:00:00Z',
        updatedAt: '2026-05-01T12:00:00Z'
      }
    ]
  }
}
