import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { App } from './App'

describe('App', () => {
  it('renders the translated navigation', () => {
    render(<App />)

    expect(screen.getByRole('heading', { name: 'Band Manager' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Inventory/i })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Merch Booth/i })).toBeInTheDocument()
  })

  it('renders the owner signup form on the signup route', async () => {
    window.history.pushState({}, '', '/signup')

    render(<App />)

    expect(await screen.findByRole('heading', { name: 'Create band account' })).toBeInTheDocument()
    expect(screen.getByText(/really_awesome_band@email.com/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Create owner account' })).toBeInTheDocument()
  })
})
