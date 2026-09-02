import { HttpResponse, http } from 'msw'

import type { AccountInvite, AccountMember, Role } from '../features/account/api'
import type { CurrentAccountResponse } from '../features/auth/api'
import type {
  CreateInventoryProductRequest,
  InventoryPhoto,
  InventoryPhotoUploadResponse,
  InventoryProduct
} from '../features/inventory/api'

type MockCurrentAccountRole = Extract<Role, 'owner' | 'viewer'>

type MockAPIState = {
  currentAccountRole: MockCurrentAccountRole
  currentAccountExists: boolean
  inventoryProducts: InventoryProduct[]
  createdProductCount: number
  photoUploadRequestFails: boolean
  storageUploadFails: boolean
}

const apiBaseURL = 'http://localhost:8080'

const mockAPIState: MockAPIState = {
  currentAccountRole: 'owner',
  currentAccountExists: true,
  inventoryProducts: [],
  createdProductCount: 0,
  photoUploadRequestFails: false,
  storageUploadFails: false
}

export const apiHandlers = [
  http.get(`${apiBaseURL}/me`, () => {
    if (!mockAPIState.currentAccountExists) {
      return HttpResponse.json(
        {
          code: 'account_not_found',
          message: 'Authenticated user has not completed onboarding'
        },
        { status: 404 }
      )
    }

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
  }),
  http.get(`${apiBaseURL}/inventory`, () => {
    return HttpResponse.json({ products: mockAPIState.inventoryProducts }, { status: 200 })
  }),
  http.post(`${apiBaseURL}/inventory/photos/upload-requests`, () => {
    if (mockAPIState.photoUploadRequestFails) {
      return HttpResponse.json({ code: 'not_found', message: 'missing route' }, { status: 404 })
    }

    return HttpResponse.json(photoUploadResponse(), { status: 201 })
  }),
  http.put('https://storage.example/full.webp', () => {
    if (mockAPIState.storageUploadFails) {
      return HttpResponse.text('upload failed', { status: 500 })
    }

    return new HttpResponse(null, { status: 200 })
  }),
  http.put('https://storage.example/display.webp', () => {
    if (mockAPIState.storageUploadFails) {
      return HttpResponse.text('upload failed', { status: 500 })
    }

    return new HttpResponse(null, { status: 200 })
  }),
  http.post(`${apiBaseURL}/inventory/products`, async ({ request }) => {
    const body = (await request.json()) as CreateInventoryProductRequest
    const createdProduct = inventoryProductFromRequest(body)
    mockAPIState.createdProductCount += 1
    mockAPIState.inventoryProducts = [...mockAPIState.inventoryProducts, createdProduct]

    return HttpResponse.json(createdProduct, { status: 201 })
  })
]

export function resetAPIMocks(): void {
  mockAPIState.currentAccountRole = 'owner'
  mockAPIState.currentAccountExists = true
  mockAPIState.inventoryProducts = []
  mockAPIState.createdProductCount = 0
  mockAPIState.photoUploadRequestFails = false
  mockAPIState.storageUploadFails = false
}

export function setMockCurrentAccountRole(role: MockCurrentAccountRole): void {
  mockAPIState.currentAccountRole = role
}

export function setMockCurrentAccountExists(exists: boolean): void {
  mockAPIState.currentAccountExists = exists
}

export function setMockInventoryProducts(products: InventoryProduct[]): void {
  mockAPIState.inventoryProducts = products
}

export function setMockStorageUploadFails(fails: boolean): void {
  mockAPIState.storageUploadFails = fails
}

export function setMockPhotoUploadRequestFails(fails: boolean): void {
  mockAPIState.photoUploadRequestFails = fails
}

export function mockCreatedProductCount(): number {
  return mockAPIState.createdProductCount
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

function photoUploadResponse(): InventoryPhotoUploadResponse {
  return {
    photo: {
      full: {
        objectKey: 'bands/test/inventory/photos/photo/full.webp',
        contentType: 'image/webp',
        sizeBytes: 1024,
        width: 1200,
        height: 900,
        publicUrl: 'https://storage.example/full-public.webp'
      },
      display: {
        objectKey: 'bands/test/inventory/photos/photo/display.webp',
        contentType: 'image/webp',
        sizeBytes: 512,
        width: 1280,
        height: 960,
        publicUrl: 'https://storage.example/display-public.webp'
      }
    },
    uploads: {
      full: {
        objectKey: 'bands/test/inventory/photos/photo/full.webp',
        signedUrl: 'https://storage.example/full.webp',
        token: 'full-token',
        expiresAt: '2026-05-01T14:00:00Z',
        publicUrl: 'https://storage.example/full-public.webp'
      },
      display: {
        objectKey: 'bands/test/inventory/photos/photo/display.webp',
        signedUrl: 'https://storage.example/display.webp',
        token: 'display-token',
        expiresAt: '2026-05-01T14:00:00Z',
        publicUrl: 'https://storage.example/display-public.webp'
      }
    }
  }
}

function inventoryProductFromRequest(request: CreateInventoryProductRequest): InventoryProduct {
  return {
    id: '33333333-3333-3333-3333-333333333333',
    bandId: '00000000-0000-0000-0000-000000000002',
    name: request.name,
    category: request.category,
    photo: inventoryPhotoFromRequest(request.photo),
    variants: request.variants.map((variant, index) => ({
      id: `44444444-4444-4444-4444-44444444444${index}`,
      productId: '33333333-3333-3333-3333-333333333333',
      size: variant.size,
      colour: variant.colour,
      price: variant.price,
      cost: variant.cost,
      quantity: variant.quantity,
      soldOut: variant.quantity === 0,
      createdAt: '2026-05-01T12:00:00Z',
      updatedAt: '2026-05-01T12:00:00Z'
    })),
    createdAt: '2026-05-01T12:00:00Z',
    updatedAt: '2026-05-01T12:00:00Z'
  }
}

function inventoryPhotoFromRequest(photo: CreateInventoryProductRequest['photo']): InventoryPhoto {
  return {
    full: {
      ...photo.full,
      publicUrl: 'https://storage.example/full-public.webp'
    },
    display: {
      ...photo.display,
      publicUrl: 'https://storage.example/display-public.webp'
    }
  }
}
