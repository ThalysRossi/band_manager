import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { InventoryPhoto, InventoryProduct } from '../features/inventory/api'
import {
  mockCreatedProductCount,
  mockCurrentAccountRequestCount,
  setMockCurrentAccountExists,
  setMockCurrentAccountRole,
  setMockInventoryProducts,
  setMockPhotoUploadRequestFails,
  setMockStorageUploadFails
} from '../test/apiMocks'
import { App } from './App'

const supabaseMock = vi.hoisted(() => {
  return {
    createClient: vi.fn(),
    exchangeCodeForSession: vi.fn(),
    getSession: vi.fn(),
    resetPasswordForEmail: vi.fn(),
    signInWithPassword: vi.fn(),
    signOut: vi.fn(),
    signUp: vi.fn(),
    unsubscribe: vi.fn(),
    updateUser: vi.fn()
  }
})

const photoProcessingMock = vi.hoisted(() => {
  return {
    processInventoryPhoto: vi.fn()
  }
})

vi.mock('@supabase/supabase-js', () => {
  return {
    createClient: supabaseMock.createClient.mockImplementation(() => ({
      auth: {
        exchangeCodeForSession: supabaseMock.exchangeCodeForSession,
        getSession: supabaseMock.getSession,
        resetPasswordForEmail: supabaseMock.resetPasswordForEmail,
        signInWithPassword: supabaseMock.signInWithPassword,
        signOut: supabaseMock.signOut,
        signUp: supabaseMock.signUp,
        updateUser: supabaseMock.updateUser,
        onAuthStateChange: () => ({
          data: {
            subscription: {
              unsubscribe: supabaseMock.unsubscribe
            }
          }
        })
      }
    }))
  }
})

vi.mock('../features/inventory/photoProcessing', () => photoProcessingMock)

describe('App', () => {
  beforeEach(() => {
    window.history.pushState({}, '', '/')
    window.sessionStorage.clear()
    supabaseMock.createClient.mockClear()
    supabaseMock.exchangeCodeForSession.mockReset()
    supabaseMock.getSession.mockReset()
    supabaseMock.resetPasswordForEmail.mockReset()
    supabaseMock.signInWithPassword.mockReset()
    supabaseMock.signOut.mockReset()
    supabaseMock.signUp.mockReset()
    supabaseMock.unsubscribe.mockReset()
    supabaseMock.updateUser.mockReset()
    photoProcessingMock.processInventoryPhoto.mockReset()
    supabaseMock.getSession.mockResolvedValue({ data: { session: null } })
    supabaseMock.exchangeCodeForSession.mockResolvedValue({ error: null })
    supabaseMock.signOut.mockResolvedValue({ error: null })
    supabaseMock.updateUser.mockResolvedValue({ error: null })
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

  it('exchanges a recovery callback without requesting the application account', async () => {
    supabaseMock.getSession
      .mockResolvedValueOnce({ data: { session: null } })
      .mockResolvedValue(authenticatedSession())
    window.history.pushState({}, '', '/auth/callback?code=recovery-code&next=/password-update')

    render(<App />)

    expect(
      await screen.findByRole('heading', { name: 'Choose a new password' })
    ).toBeInTheDocument()
    expect(supabaseMock.exchangeCodeForSession).toHaveBeenCalledWith('recovery-code')
    expect(mockCurrentAccountRequestCount()).toBe(0)
  })

  it('shows a recovery-specific message when callback exchange fails', async () => {
    supabaseMock.exchangeCodeForSession.mockResolvedValue({
      error: { message: 'recovery link expired' }
    })
    window.history.pushState({}, '', '/auth/callback?code=expired-code&next=/password-update')

    render(<App />)

    expect(
      await screen.findByText('This recovery link is invalid or expired. Request a new one.')
    ).toBeInTheDocument()
  })

  it('returns to inventory after updating the password', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    window.history.pushState({}, '', '/password-update')

    render(<App />)

    fireEvent.change(await screen.findByLabelText('Password'), { target: { value: 'password-2' } })
    fireEvent.click(screen.getByRole('button', { name: 'Update password' }))

    expect(await screen.findByRole('heading', { name: 'Inventory' })).toBeInTheDocument()
    expect(supabaseMock.updateUser).toHaveBeenCalledWith({ password: 'password-2' })
  })

  it('shows a password-update-specific message when the recovery session is invalid', async () => {
    supabaseMock.updateUser.mockResolvedValue({
      error: { message: 'recovery session expired' }
    })
    window.history.pushState({}, '', '/password-update')

    render(<App />)

    fireEvent.change(await screen.findByLabelText('Password'), { target: { value: 'password-2' } })
    fireEvent.click(screen.getByRole('button', { name: 'Update password' }))

    expect(
      await screen.findByText('Password update failed. Request a new recovery link and try again.')
    ).toBeInTheDocument()
  })

  it('renders protected workspace routes for authenticated users', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    setMockInventoryProducts([mockInventoryProduct()])
    window.history.pushState({}, '', '/merch-booth')

    render(<App />)

    expect(await screen.findByRole('heading', { name: 'Merch Booth' })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Cart' })).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Open cart: 0' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Add to cart Logo Shirt Black' })).toBeDisabled()
    expect(
      screen.getByRole('combobox', { name: 'Select size Logo Shirt Black' })
    ).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Complete cash sale' })).not.toBeInTheDocument()
    expect(await screen.findByText('Os Testes')).toBeInTheDocument()
    expect(screen.getByText('owner@example.com | Owner')).toBeInTheDocument()
  })

  it('prevents sold-out variants from being added to the booth cart', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    setMockInventoryProducts([mockInventoryProductWithSoldOutVariant()])
    window.history.pushState({}, '', '/merch-booth')

    render(<App />)

    expect(await screen.findByRole('option', { name: 'G — Sold out' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Add to cart Logo Shirt Black' })).toBeDisabled()
  })

  it('limits cart quantity to stock and completes a cash sale', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    setMockInventoryProducts([mockInventoryProduct()])
    window.history.pushState({}, '', '/merch-booth')

    render(<App />)

    await addMShirtToCart()
    expect(screen.getByRole('link', { name: 'Open cart: 1' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('link', { name: 'Open cart: 1' }))

    expect(await screen.findByRole('heading', { name: 'Cart' })).toBeInTheDocument()
    fireEvent.click(
      screen.getByRole('button', { name: 'Increase quantity for Logo Shirt M / Black' })
    )

    expect(
      screen.getByRole('button', { name: 'Increase quantity for Logo Shirt M / Black' })
    ).toBeDisabled()

    fireEvent.click(screen.getByRole('button', { name: 'Complete cash sale' }))

    expect(await screen.findByText('Cash sale completed.')).toBeInTheDocument()
    expect(window.location.pathname).toBe('/merch-booth')
    expect(await screen.findByText('Sold out')).toBeInTheDocument()
  })

  it('restores the booth cart after a page refresh in the same tab', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    setMockInventoryProducts([mockInventoryProduct()])
    window.history.pushState({}, '', '/merch-booth')

    const firstRender = render(<App />)

    await addMShirtToCart()
    expect(screen.getByRole('link', { name: 'Open cart: 1' })).toBeInTheDocument()
    firstRender.unmount()
    window.history.pushState({}, '', '/merch-booth/cart')

    render(<App />)

    expect(await screen.findByRole('heading', { name: 'Cart' })).toBeInTheDocument()
    expect(screen.getByText('Logo Shirt')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Complete cash sale' })).toBeEnabled()
  })

  it('reconciles restored cart quantities against current stock', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    const product = mockInventoryProduct()
    setMockInventoryProducts([product])
    window.history.pushState({}, '', '/merch-booth')

    const firstRender = render(<App />)

    await addMShirtToCart()
    fireEvent.click(screen.getByRole('button', { name: 'Add to cart Logo Shirt Black' }))
    expect(screen.getByRole('link', { name: 'Open cart: 2' })).toBeInTheDocument()
    firstRender.unmount()
    setMockInventoryProducts([
      {
        ...product,
        variants: product.variants.map((variant) => ({ ...variant, quantity: 1 }))
      }
    ])
    window.history.pushState({}, '', '/merch-booth/cart')

    render(<App />)

    expect(
      await screen.findByText('The cart was updated to match current stock.')
    ).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Increase quantity for Logo Shirt M / Black' })
    ).toBeDisabled()
    await waitFor(() => {
      expect(window.sessionStorage.getItem(window.sessionStorage.key(0) ?? '')).toContain(
        '"quantity":1'
      )
    })
  })

  it('keeps the booth read-only for viewers', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    setMockCurrentAccountRole('viewer')
    setMockInventoryProducts([mockInventoryProduct()])
    window.history.pushState({}, '', '/merch-booth')

    render(<App />)

    expect(await screen.findByRole('heading', { name: 'Merch Booth' })).toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Add to cart Logo Shirt M / Black' })
    ).not.toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /Open cart/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Complete cash sale' })).not.toBeInTheDocument()
  })

  it('keeps the cart route read-only for viewers', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    setMockCurrentAccountRole('viewer')
    setMockInventoryProducts([mockInventoryProduct()])
    window.history.pushState({}, '', '/merch-booth/cart')

    render(<App />)

    expect(await screen.findByRole('heading', { name: 'Cart' })).toBeInTheDocument()
    expect(screen.getByText('Your role does not allow merch booth checkout.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Complete cash sale' })).not.toBeInTheDocument()
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
    expect(screen.getByRole('button', { name: 'Add product' })).toHaveAttribute(
      'aria-expanded',
      'true'
    )
  })

  it('collapses product creation when inventory has products and preserves a draft when toggled', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    setMockInventoryProducts([mockInventoryProduct()])
    window.history.pushState({}, '', '/')

    render(<App />)

    const toggle = await screen.findByRole('button', { name: 'Add product' })
    expect(toggle).toHaveAttribute('aria-expanded', 'false')
    expect(screen.getByLabelText('Product name')).not.toBeVisible()

    fireEvent.click(toggle)
    expect(toggle).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByLabelText('Product name')).toBeVisible()
    fireEvent.change(screen.getByLabelText('Product name'), { target: { value: 'Draft shirt' } })
    fireEvent.click(toggle)
    fireEvent.click(toggle)
    expect(screen.getByLabelText('Product name')).toHaveValue('Draft shirt')
  })

  it('renders inventory products', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    setMockInventoryProducts([mockInventoryProduct()])

    render(<App />)

    expect(await screen.findByText('Logo Shirt')).toBeInTheDocument()
    expect(screen.getAllByText('Shirt').length).toBeGreaterThan(0)
    expect(screen.getByText('In stock')).toBeInTheDocument()
  })

  it('shows one colour variant with nested sizes and one booth card per colour', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    setMockInventoryProducts([mockDeadBirdProduct()])

    render(<App />)

    expect(await screen.findByText('Dead Bird')).toBeInTheDocument()
    expect(screen.getByText('2 variants')).toBeInTheDocument()
    expect(screen.getByAltText('Dead Bird Black')).toHaveAttribute(
      'src',
      'https://storage.example/display-public.webp'
    )
    expect(screen.getByAltText('Dead Bird Orange')).toHaveAttribute(
      'src',
      'https://storage.example/orange-display.webp'
    )

    fireEvent.click(screen.getByRole('link', { name: 'Merch Booth' }))
    expect(
      await screen.findByRole('combobox', { name: 'Select size Dead Bird Black' })
    ).toBeInTheDocument()
    expect(
      screen.getByRole('combobox', { name: 'Select size Dead Bird Orange' })
    ).toBeInTheDocument()
    expect(screen.getAllByRole('button', { name: /Add to cart Dead Bird/ })).toHaveLength(2)
    fireEvent.change(screen.getByRole('combobox', { name: 'Select size Dead Bird Orange' }), {
      target: { value: '44444444-4444-4444-4444-444444444447' }
    })
    fireEvent.click(screen.getByRole('button', { name: 'Add to cart Dead Bird Orange' }))
    expect(screen.getByRole('link', { name: 'Open cart: 1' })).toBeInTheDocument()
  })

  it('hides inventory create controls for a viewer', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    setMockCurrentAccountRole('viewer')
    setMockInventoryProducts([mockInventoryProduct()])

    render(<App />)

    expect(await screen.findByRole('heading', { name: 'Inventory' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Create product' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Add product' })).not.toBeInTheDocument()
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

    expect(await screen.findByText('Product created.')).toBeVisible()
    expect(screen.getByRole('button', { name: 'Add product' })).toHaveAttribute(
      'aria-expanded',
      'false'
    )
    expect(await screen.findByText('Logo Shirt')).toBeInTheDocument()
    expect(mockCreatedProductCount()).toBe(1)
  })

  it('creates a non-clothing colour with one implicit stock row', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    photoProcessingMock.processInventoryPhoto.mockResolvedValue(processedInventoryPhoto())

    render(<App />)

    fireEvent.change(await screen.findByLabelText('Product name'), {
      target: { value: 'Dead Bird Vinyl' }
    })
    fireEvent.change(screen.getByLabelText('Category'), { target: { value: 'vinyl' } })
    expect(screen.queryByLabelText('Size')).not.toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('Colour'), { target: { value: 'Black' } })
    fireEvent.change(screen.getByLabelText('Quantity'), { target: { value: '2' } })
    fireEvent.change(screen.getByLabelText('Photo'), {
      target: { files: [new File(['photo'], 'vinyl.jpg', { type: 'image/jpeg' })] }
    })
    expect(await screen.findByText(/Photo ready/i)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Create product' }))
    expect(await screen.findByText('Product created.')).toBeVisible()

    fireEvent.click(screen.getByRole('link', { name: 'Merch Booth' }))
    expect(
      await screen.findByRole('button', { name: 'Add to cart Dead Bird Vinyl Black' })
    ).toBeEnabled()
    expect(screen.queryByRole('combobox', { name: /Select size/ })).not.toBeInTheDocument()
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
    fireEvent.change(screen.getByLabelText('Colour'), { target: { value: 'Black' } })

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
    fireEvent.change(screen.getByLabelText('Colour'), { target: { value: 'Black' } })

    expect(await screen.findByText(/Photo ready/i)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Create product' }))

    expect(
      await screen.findByText(
        'Photo upload request failed. Restart the API and check VITE_API_BASE_URL.'
      )
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

    fireEvent.click(
      await screen.findByRole('button', { name: 'Edit variant Logo Shirt M / Black' })
    )
    fireEvent.change(screen.getByLabelText('Edit variant colour'), {
      target: { value: 'Red' }
    })
    fireEvent.change(screen.getByLabelText('Edit variant quantity'), { target: { value: '4' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save variant' }))

    expect(await screen.findByText('Variant updated.')).toBeInTheDocument()
    expect(await screen.findByText('Red')).toBeInTheDocument()
    expect(await screen.findByText('4 in stock')).toBeInTheDocument()
  })

  it('creates an inventory variant for an existing product', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    photoProcessingMock.processInventoryPhoto.mockResolvedValue(processedInventoryPhoto())
    setMockInventoryProducts([mockInventoryProduct()])

    render(<App />)

    fireEvent.click(await screen.findByRole('button', { name: 'Add variant Logo Shirt' }))
    fireEvent.change(screen.getByLabelText('New variant size'), { target: { value: 'g' } })
    fireEvent.change(screen.getByLabelText('New variant colour'), { target: { value: 'Red' } })
    fireEvent.change(screen.getByLabelText('New variant price (BRL)'), { target: { value: '60' } })
    fireEvent.change(screen.getByLabelText('New variant cost (BRL)'), { target: { value: '25' } })
    fireEvent.change(screen.getByLabelText('New variant quantity'), { target: { value: '3' } })
    fireEvent.change(
      screen.getByLabelText('Photo', { selector: 'input[id^="inventory-create-variant-photo"]' }),
      {
        target: { files: [new File(['photo'], 'red.jpg', { type: 'image/jpeg' })] }
      }
    )
    fireEvent.click(screen.getByRole('button', { name: 'Create variant' }))

    expect(await screen.findByText('Variant created.')).toBeInTheDocument()
    expect(await screen.findByText('2 variants')).toBeInTheDocument()
    expect(await screen.findByText('Red')).toBeInTheDocument()
  })

  it('deletes an inventory variant after confirmation', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    setMockInventoryProducts([mockInventoryProductWithTwoVariants()])

    render(<App />)

    fireEvent.click(
      await screen.findByRole('button', { name: 'Delete variant Logo Shirt M / Black' })
    )

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

  it('logs out from the header account dropdown and clears the cart', async () => {
    supabaseMock.getSession.mockResolvedValue(authenticatedSession())
    setMockInventoryProducts([mockInventoryProduct()])
    window.history.pushState({}, '', '/merch-booth')

    render(<App />)

    await addMShirtToCart()
    await waitFor(() => {
      expect(window.sessionStorage.length).toBe(1)
    })
    fireEvent.keyDown(await screen.findByRole('button', { name: /Os Testes/i }), {
      key: 'Enter',
      code: 'Enter'
    })
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Log out' }))

    await waitFor(() => {
      expect(supabaseMock.signOut).toHaveBeenCalledTimes(1)
    })
    expect(window.sessionStorage.length).toBe(0)
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
    photo: mockPhoto(),
    variants: [
      {
        id: '44444444-4444-4444-4444-444444444444',
        colourVariantId: 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
        productId: '33333333-3333-3333-3333-333333333333',
        size: 'm',
        colour: 'Black',
        photo: mockPhoto(),
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

function mockPhoto(): InventoryPhoto {
  return {
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
  }
}

async function addMShirtToCart(): Promise<void> {
  const sizeSelect = await screen.findByRole('combobox', { name: 'Select size Logo Shirt Black' })
  fireEvent.change(sizeSelect, { target: { value: '44444444-4444-4444-4444-444444444444' } })
  fireEvent.click(screen.getByRole('button', { name: 'Add to cart Logo Shirt Black' }))
}

function mockInventoryProductWithTwoVariants(): InventoryProduct {
  const product = mockInventoryProduct()
  return {
    ...product,
    variants: [
      ...product.variants,
      {
        id: '44444444-4444-4444-4444-444444444445',
        colourVariantId: product.variants[0].colourVariantId,
        productId: product.id,
        size: 'g',
        colour: 'Black',
        photo: product.variants[0].photo,
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

function mockDeadBirdProduct(): InventoryProduct {
  const product = mockInventoryProduct()
  const blackM = product.variants[0]
  const orangePhoto: InventoryPhoto = {
    full: { ...mockPhoto().full, objectKey: 'bands/test/orange/full.webp' },
    display: {
      ...mockPhoto().display,
      objectKey: 'bands/test/orange/display.webp',
      publicUrl: 'https://storage.example/orange-display.webp'
    }
  }
  return {
    ...product,
    name: 'Dead Bird',
    variants: [
      { ...blackM, id: '44444444-4444-4444-4444-444444444445', size: 'p', quantity: 3 },
      { ...blackM, quantity: 3 },
      {
        ...blackM,
        id: '44444444-4444-4444-4444-444444444446',
        colourVariantId: 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
        size: 'g',
        colour: 'Orange',
        quantity: 1,
        photo: orangePhoto
      },
      {
        ...blackM,
        id: '44444444-4444-4444-4444-444444444447',
        colourVariantId: 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
        size: 'm',
        colour: 'Orange',
        quantity: 5,
        photo: orangePhoto
      }
    ]
  }
}

function mockInventoryProductWithSoldOutVariant(): InventoryProduct {
  const product = mockInventoryProduct()
  return {
    ...product,
    variants: [
      ...product.variants,
      {
        id: '44444444-4444-4444-4444-444444444445',
        colourVariantId: product.variants[0].colourVariantId,
        productId: product.id,
        size: 'g',
        colour: 'Black',
        photo: product.variants[0].photo,
        price: { amount: 5000, currency: 'BRL' },
        cost: { amount: 2000, currency: 'BRL' },
        quantity: 0,
        soldOut: true,
        createdAt: '2026-05-01T12:00:00Z',
        updatedAt: '2026-05-01T12:00:00Z'
      }
    ]
  }
}
