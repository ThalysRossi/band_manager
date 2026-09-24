import { beforeEach, describe, expect, it, vi } from 'vitest'

const createClientMock = vi.hoisted(() => vi.fn())

vi.mock('@supabase/supabase-js', () => ({
  createClient: createClientMock
}))

describe('getSupabaseClient', () => {
  beforeEach(() => {
    createClientMock.mockReset()
    createClientMock.mockReturnValue({})
    vi.resetModules()
  })

  it('creates one client for repeated auth access in the same browser context', async () => {
    const { getSupabaseClient } = await import('./supabase')

    const firstClient = getSupabaseClient()
    const secondClient = getSupabaseClient()

    expect(secondClient).toBe(firstClient)
    expect(createClientMock).toHaveBeenCalledTimes(1)
  })
})
