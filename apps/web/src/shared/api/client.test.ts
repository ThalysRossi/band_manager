import { HttpResponse, http } from 'msw'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { server } from '../../test/server'
import { apiRequest } from './client'

const getAuthSessionMock = vi.hoisted(() =>
  vi.fn<() => Promise<{ accessToken: string; email: string } | null>>()
)

vi.mock('../auth/provider', () => ({
  getAuthSession: getAuthSessionMock
}))

describe('apiRequest session', () => {
  beforeEach(() => {
    getAuthSessionMock.mockReset()
  })

  it('uses the current session token when the caller holds an older token', async () => {
    let receivedAuthorization: string | null = null
    getAuthSessionMock.mockResolvedValue({ accessToken: 'current-token', email: 'owner@example.com' })
    server.use(
      http.get('http://localhost:8080/me', ({ request }) => {
        receivedAuthorization = request.headers.get('Authorization')
        return HttpResponse.json({ userId: 'owner' })
      })
    )

    await apiRequest<{ userId: string }>({
      accessToken: 'expired-token',
      path: '/me',
      method: 'GET',
      body: null,
      idempotent: false
    })

    expect(receivedAuthorization).toBe('Bearer current-token')
  })

  it('does not send a request when the session is missing', async () => {
    let receivedRequests: number = 0
    getAuthSessionMock.mockResolvedValue(null)
    server.use(
      http.get('http://localhost:8080/me', () => {
        receivedRequests += 1
        return HttpResponse.json({ userId: 'owner' })
      })
    )

    await expect(
      apiRequest<{ userId: string }>({
        accessToken: 'expired-token',
        path: '/me',
        method: 'GET',
        body: null,
        idempotent: false
      })
    ).rejects.toMatchObject({ statusCode: 401, code: 'invalid_session' })
    expect(receivedRequests).toBe(0)
  })
})
