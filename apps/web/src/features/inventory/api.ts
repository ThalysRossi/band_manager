import { apiRequest } from '../../shared/api/client'

export type InventoryCategory =
  | 'shirt'
  | 'hoodie'
  | 'tote_bag'
  | 'patch'
  | 'sticker'
  | 'vinyl'
  | 'cd'
  | 'cassette'
  | 'accessory'

export type InventorySize =
  | 'not_applicable'
  | 'one_size'
  | 'pp'
  | 'p'
  | 'm'
  | 'g'
  | 'gg'
  | 'xgg'

export type Money = {
  amount: number
  currency: 'BRL'
}

export type InventoryPhotoVariantManifest = {
  objectKey: string
  contentType: 'image/webp'
  sizeBytes: number
  width: number
  height: number
}

export type InventoryPhotoManifest = {
  full: InventoryPhotoVariantManifest
  display: InventoryPhotoVariantManifest
}

export type InventoryPhotoVariant = InventoryPhotoVariantManifest & {
  publicUrl: string
}

export type InventoryPhoto = {
  full: InventoryPhotoVariant
  display: InventoryPhotoVariant
}

export type InventoryVariant = {
  id: string
  productId: string
  size: InventorySize
  colour: string
  price: Money
  cost: Money
  quantity: number
  soldOut: boolean
  createdAt: string
  updatedAt: string
}

export type InventoryProduct = {
  id: string
  bandId: string
  name: string
  category: InventoryCategory
  photo: InventoryPhoto
  variants: InventoryVariant[]
  createdAt: string
  updatedAt: string
}

export type InventoryVariantRequest = {
  size: InventorySize
  colour: string
  price: Money
  cost: Money
  quantity: number
}

export type CreateInventoryProductRequest = {
  name: string
  category: InventoryCategory
  photo: InventoryPhotoManifest
  variants: InventoryVariantRequest[]
}

export type UpdateInventoryProductRequest = {
  name: string
  category: InventoryCategory
  photo: InventoryPhotoManifest
}

export type UpdateInventoryVariantRequest = InventoryVariantRequest

export type PhotoUploadVariantRequest = {
  contentType: 'image/webp'
  sizeBytes: number
  width: number
  height: number
}

export type InventoryPhotoUploadRequest = {
  full: PhotoUploadVariantRequest
  display: PhotoUploadVariantRequest
}

export type InventoryPhotoUploadTarget = {
  objectKey: string
  signedUrl: string
  token: string
  expiresAt: string
  publicUrl: string
}

export type InventoryPhotoUploadResponse = {
  photo: InventoryPhoto
  uploads: {
    full: InventoryPhotoUploadTarget
    display: InventoryPhotoUploadTarget
  }
}

type InventoryResponse = {
  products: InventoryProduct[]
}

export async function listInventory(accessToken: string): Promise<InventoryProduct[]> {
  const response = await apiRequest<InventoryResponse>({
    accessToken,
    path: '/inventory',
    method: 'GET',
    body: null,
    idempotent: false
  })

  return response.products
}

export async function createInventoryPhotoUploadRequest(
  accessToken: string,
  request: InventoryPhotoUploadRequest
): Promise<InventoryPhotoUploadResponse> {
  return apiRequest<InventoryPhotoUploadResponse>({
    accessToken,
    path: '/inventory/photos/upload-requests',
    method: 'POST',
    body: {
      full: request.full,
      display: request.display
    },
    idempotent: false
  })
}

export async function uploadInventoryPhotoVariant(
  target: InventoryPhotoUploadTarget,
  blob: Blob
): Promise<void> {
  const response = await fetch(target.signedUrl, {
    method: 'PUT',
    headers: {
      'Content-Type': blob.type
    },
    body: blob
  })

  if (!response.ok) {
    const body = await response.text()
    throw new Error(
      `Photo upload failed object_key="${target.objectKey}" status_code=${response.status} response_body="${body}"`
    )
  }
}

export async function createInventoryProduct(
  accessToken: string,
  request: CreateInventoryProductRequest
): Promise<InventoryProduct> {
  return apiRequest<InventoryProduct>({
    accessToken,
    path: '/inventory/products',
    method: 'POST',
    body: {
      name: request.name,
      category: request.category,
      photo: request.photo,
      variants: request.variants
    },
    idempotent: true
  })
}

export async function updateInventoryProduct(
  accessToken: string,
  productID: string,
  request: UpdateInventoryProductRequest
): Promise<InventoryProduct> {
  return apiRequest<InventoryProduct>({
    accessToken,
    path: `/inventory/products/${productID}`,
    method: 'PUT',
    body: {
      name: request.name,
      category: request.category,
      photo: request.photo
    },
    idempotent: true
  })
}

export async function deleteInventoryProduct(
  accessToken: string,
  productID: string
): Promise<void> {
  await apiRequest<void>({
    accessToken,
    path: `/inventory/products/${productID}`,
    method: 'DELETE',
    body: null,
    idempotent: true
  })
}

export async function updateInventoryVariant(
  accessToken: string,
  variantID: string,
  request: UpdateInventoryVariantRequest
): Promise<InventoryVariant> {
  return apiRequest<InventoryVariant>({
    accessToken,
    path: `/inventory/variants/${variantID}`,
    method: 'PUT',
    body: request,
    idempotent: true
  })
}

export async function deleteInventoryVariant(accessToken: string, variantID: string): Promise<void> {
  await apiRequest<void>({
    accessToken,
    path: `/inventory/variants/${variantID}`,
    method: 'DELETE',
    body: null,
    idempotent: true
  })
}
