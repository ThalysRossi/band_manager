import { apiRequest } from '../../shared/api/client'
import type { InventoryCategory, InventoryPhoto, InventorySize, Money } from '../inventory/api'

export type BoothItem = {
  productId: string
  variantId: string
  colourVariantId: string
  productName: string
  category: InventoryCategory
  size: InventorySize
  colour: string
  price: Money
  cost: Money
  quantity: number
  soldOut: boolean
  photo: InventoryPhoto
}

export type CashCheckoutItem = {
  variantId: string
  quantity: number
}

export type CashCheckoutRequest = {
  items: CashCheckoutItem[]
}

export type CashCheckoutResponse = {
  id: string
  status: 'finalized'
}

type BoothItemsResponse = {
  items: BoothItem[]
}

export async function listBoothItems(accessToken: string): Promise<BoothItem[]> {
  const response = await apiRequest<BoothItemsResponse>({
    accessToken,
    path: '/merch-booth/items',
    method: 'GET',
    body: null,
    idempotent: false
  })

  return response.items
}

export async function createCashCheckout(
  accessToken: string,
  request: CashCheckoutRequest
): Promise<CashCheckoutResponse> {
  return apiRequest<CashCheckoutResponse>({
    accessToken,
    path: '/merch-booth/checkouts/cash',
    method: 'POST',
    body: request,
    idempotent: true
  })
}
