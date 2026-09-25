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

  return (
    <section className="grid gap-ui-24">
      <header className="flex items-start justify-between gap-ui-16">
        <div className="grid min-w-0 gap-ui-8">
          <h2 className="m-0 text-[1.75rem] leading-[1.15]">
            {merchBooth.translate('nav.merchBooth')}
          </h2>
          <p className="m-0 text-base text-white-300">
            {merchBooth.translate('merchBooth.itemCount')}: {merchBooth.items.length}
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

      {merchBooth.items.length === 0 ? (
        <p className="m-0 text-base text-white-300">
          {merchBooth.translate('merchBooth.empty')}
        </p>
      ) : (
        <div className="grid gap-ui-16 min-[600px]:grid-cols-2 min-[1200px]:grid-cols-3">
          {merchBooth.items.map((item) => (
            <BoothItemCard
              key={item.variantId}
              item={item}
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

function BoothItemCard(props: {
  item: BoothItem
  canCheckout: boolean
  translate: Translate
  onAddToCart: (variantId: string) => void
}) {
  const variantLabel = boothItemLabel(props.item, props.translate)

  return (
    <Card className="overflow-hidden py-0">
      <img
        src={props.item.photo.display.publicUrl}
        alt={`${props.item.productName} ${props.translate('merchBooth.photoAlt')}`}
        className="aspect-[4/3] w-full bg-muted object-contain"
      />
      <CardHeader className="gap-ui-8 px-ui-16 pt-ui-16">
        <div className="flex flex-wrap items-start justify-between gap-ui-8">
          <h3 className="m-0 text-base leading-tight">{props.item.productName}</h3>
          <Badge variant="outline">{props.translate(categoryLabelKey(props.item.category))}</Badge>
        </div>
        <p className="m-0 text-sm text-white-300">
          {props.translate(sizeLabelKey(props.item.size))} / {props.item.colour}
        </p>
      </CardHeader>
      <CardContent className="grid gap-ui-16 px-ui-16 pb-ui-16">
        <div className="flex items-center justify-between gap-ui-12">
          <strong className="text-base">{formatMoney(props.item.price.amount)}</strong>
          <Badge variant={props.item.soldOut ? 'destructive' : 'secondary'}>
            {props.item.soldOut
              ? props.translate('inventory.soldOut')
              : `${props.item.quantity} ${props.translate('inventory.inStockCountSuffix')}`}
          </Badge>
        </div>
        {props.canCheckout ? (
          <Button
            type="button"
            variant="outline"
            disabled={props.item.soldOut}
            aria-label={`${props.translate('merchBooth.addToCart')} ${variantLabel}`}
            onClick={() => props.onAddToCart(props.item.variantId)}
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
