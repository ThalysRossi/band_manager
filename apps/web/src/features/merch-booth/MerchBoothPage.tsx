import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { Plus, ShoppingCart } from 'lucide-react'
import type { TranslationKey } from 'i18n'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import type { BoothItem } from './api'
import { useMerchBoothCart } from './cart'
import { useMerchBooth } from './MerchBoothLayout'
import type { Translate } from './MerchBoothLayout'

export function MerchBoothPage() {
  const merchBooth = useMerchBooth()
  const cart = useMerchBoothCart()
  const canCheckout = merchBooth.account.activeBand.canWrite
  const colourGroups = groupBoothItems(merchBooth.items)

  return (
    <section className="grid gap-ui-24">
      <header className="flex items-start justify-between gap-ui-16">
        <div className="grid min-w-0 gap-ui-8">
          <h2 className="m-0 text-[1.75rem] leading-[1.15]">
            {merchBooth.translate('nav.merchBooth')}
          </h2>
          <p className="m-0 text-base text-white-300">
            {merchBooth.translate('merchBooth.itemCount')}: {colourGroups.length}
          </p>
        </div>
        {canCheckout ? (
          <Button variant="outline" asChild>
            <Link
              to="/merch-booth/cart"
              aria-label={`${merchBooth.translate('merchBooth.openCart')}: ${cart.totalUnits}`}
            >
              <ShoppingCart aria-hidden="true" />
              <span className="hidden min-[480px]:inline">
                {merchBooth.translate('merchBooth.cartTitle')}
              </span>
              <Badge className="min-w-ui-24 px-ui-4" aria-hidden="true">
                {cart.totalUnits}
              </Badge>
            </Link>
          </Button>
        ) : null}
      </header>

      {cart.checkoutSucceeded ? (
        <p className="m-0 text-sm text-white-200" role="status">
          {merchBooth.translate('merchBooth.cashCheckoutSuccess')}
        </p>
      ) : null}
      {cart.persistenceFailed ? (
        <p className="m-0 text-sm text-red-100" role="status">
          {merchBooth.translate('merchBooth.cartPersistenceError')}
        </p>
      ) : null}
      {cart.reconciled ? (
        <p className="m-0 text-sm text-white-200" role="status">
          {merchBooth.translate('merchBooth.cartReconciled')}
        </p>
      ) : null}

      {colourGroups.length === 0 ? (
        <p className="m-0 text-base text-white-300">{merchBooth.translate('merchBooth.empty')}</p>
      ) : (
        <div className="grid gap-ui-16 min-[600px]:grid-cols-2 min-[1200px]:grid-cols-3">
          {colourGroups.map((group) => (
            <BoothItemCard
              key={group.id}
              group={group}
              canCheckout={canCheckout}
              translate={merchBooth.translate}
              onAddToCart={cart.addItem}
            />
          ))}
        </div>
      )}
    </section>
  )
}

type BoothColourGroup = {
  id: string
  productName: string
  category: BoothItem['category']
  colour: string
  price: BoothItem['price']
  photo: BoothItem['photo']
  sizes: BoothItem[]
}

function groupBoothItems(items: BoothItem[]): BoothColourGroup[] {
  const groups = new Map<string, BoothColourGroup>()
  for (const item of items) {
    const existing = groups.get(item.colourVariantId)
    if (existing === undefined) {
      groups.set(item.colourVariantId, {
        id: item.colourVariantId,
        productName: item.productName,
        category: item.category,
        colour: item.colour,
        price: item.price,
        photo: item.photo,
        sizes: [item]
      })
    } else {
      existing.sizes.push(item)
    }
  }
  return Array.from(groups.values())
}

function BoothItemCard(props: {
  group: BoothColourGroup
  canCheckout: boolean
  translate: Translate
  onAddToCart: (variantId: string) => void
}) {
  const [selectedVariantID, setSelectedVariantID] = useState<string>('')
  const sized = props.group.category === 'shirt' || props.group.category === 'hoodie'
  const selectedItem = sized
    ? props.group.sizes.find((item) => item.variantId === selectedVariantID)
    : props.group.sizes[0]
  const totalQuantity = props.group.sizes.reduce((total, item) => total + item.quantity, 0)
  const soldOut = totalQuantity === 0

  return (
    <Card className="overflow-hidden py-0">
      <img
        src={props.group.photo.display.publicUrl}
        alt={`${props.group.productName} ${props.group.colour} ${props.translate('merchBooth.photoAlt')}`}
        className="aspect-[4/3] w-full bg-muted object-cover"
      />
      <CardHeader className="gap-ui-8 px-ui-16 pt-ui-16">
        <div className="flex flex-wrap items-start justify-between gap-ui-8">
          <h3 className="m-0 text-base leading-tight">{props.group.productName}</h3>
          <Badge variant="outline">{props.translate(categoryLabelKey(props.group.category))}</Badge>
        </div>
        <p className="m-0 text-sm text-white-300">{props.group.colour}</p>
      </CardHeader>
      <CardContent className="grid gap-ui-16 px-ui-16 pb-ui-16">
        <div className="flex items-center justify-between gap-ui-12">
          <strong className="text-base">{formatMoney(props.group.price.amount)}</strong>
          <Badge variant={soldOut ? 'destructive' : 'secondary'}>
            {soldOut
              ? props.translate('inventory.soldOut')
              : `${totalQuantity} ${props.translate('inventory.inStockCountSuffix')}`}
          </Badge>
        </div>
        {sized ? (
          <select
            className="h-9 w-full rounded-md border border-input bg-background px-ui-12 text-sm"
            aria-label={`${props.translate('merchBooth.selectSize')} ${props.group.productName} ${props.group.colour}`}
            value={selectedVariantID}
            onChange={(event) => setSelectedVariantID(event.currentTarget.value)}
          >
            <option value="">{props.translate('merchBooth.selectSize')}</option>
            {props.group.sizes.map((item) => (
              <option key={item.variantId} value={item.variantId} disabled={item.soldOut}>
                {props.translate(sizeLabelKey(item.size))} —{' '}
                {item.soldOut
                  ? props.translate('inventory.soldOut')
                  : `${item.quantity} ${props.translate('inventory.inStockCountSuffix')}`}
              </option>
            ))}
          </select>
        ) : null}
        {props.canCheckout ? (
          <Button
            type="button"
            variant="outline"
            disabled={selectedItem === undefined || selectedItem.soldOut}
            aria-label={`${props.translate('merchBooth.addToCart')} ${props.group.productName} ${props.group.colour}`}
            onClick={() => {
              if (selectedItem !== undefined) props.onAddToCart(selectedItem.variantId)
            }}
          >
            <Plus aria-hidden="true" />
            {props.translate('merchBooth.addToCart')}
          </Button>
        ) : null}
      </CardContent>
    </Card>
  )
}

export function boothItemLabel(item: BoothItem, translate: Translate): string {
  return `${item.productName} ${translate(sizeLabelKey(item.size))} / ${item.colour}`
}

export function formatMoney(amount: number): string {
  return new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(amount / 100)
}

function categoryLabelKey(category: BoothItem['category']): TranslationKey {
  return `inventory.category.${category}` as TranslationKey
}

function sizeLabelKey(size: BoothItem['size']): TranslationKey {
  return `inventory.size.${size}` as TranslationKey
}
