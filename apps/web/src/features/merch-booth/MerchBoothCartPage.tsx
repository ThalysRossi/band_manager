import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { ArrowLeft, Minus, Plus, ShoppingCart, Trash2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { createCashCheckout } from './api'
import type { BoothItem } from './api'
import { toCashCheckoutItems, useMerchBoothCart } from './cart'
import type { CartLine } from './cart'
import { boothItemLabel, formatMoney } from './MerchBoothPage'
import { useMerchBooth } from './MerchBoothLayout'
import type { Translate } from './MerchBoothLayout'

export function MerchBoothCartPage() {
  const merchBooth = useMerchBooth()
  const cart = useMerchBoothCart()
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const [checkoutError, setCheckoutError] = useState<boolean>(false)
  const resolvedLines = resolveCartLines(cart.lines, merchBooth.items)
  const total = totalCartAmount(resolvedLines)

  const cashCheckoutMutation = useMutation({
    mutationFn: () =>
      createCashCheckout(merchBooth.accessToken, { items: toCashCheckoutItems(cart.lines) }),
    onSuccess: async () => {
      cart.completeCheckout()
      await queryClient.invalidateQueries({
        queryKey: ['merch-booth', 'items', merchBooth.accessToken]
      })
      await navigate({ to: '/merch-booth' })
    },
    onError: () => {
      setCheckoutError(true)
    }
  })

  if (!merchBooth.account.activeBand.canWrite) {
    return (
      <section className="grid gap-ui-24">
        <CartHeader translate={merchBooth.translate} />
        <p className="m-0 text-base text-white-300" role="status">
          {merchBooth.translate('merchBooth.cartReadOnly')}
        </p>
      </section>
    )
  }

  return (
    <section className="grid gap-ui-24">
      <CartHeader translate={merchBooth.translate} />

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
      {checkoutError ? (
        <p className="m-0 text-sm text-red-100" role="status">
          {merchBooth.translate('merchBooth.checkoutError')}
        </p>
      ) : null}

      {resolvedLines.length === 0 ? (
        <Card>
          <CardContent className="grid justify-items-center gap-ui-16 text-center">
            <ShoppingCart aria-hidden="true" className="size-8 text-white-300" />
            <p className="m-0 text-sm text-white-300">
              {merchBooth.translate('merchBooth.cartEmpty')}
            </p>
            <Button variant="outline" asChild>
              <Link to="/merch-booth">{merchBooth.translate('merchBooth.backToBooth')}</Link>
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-ui-24 min-[900px]:grid-cols-[minmax(0,1fr)_280px] min-[900px]:items-start">
          <ul className="m-0 grid list-none gap-ui-12 p-0">
            {resolvedLines.map((line) => (
              <CartLineRow
                key={line.item.variantId}
                line={line}
                translate={merchBooth.translate}
                onSetQuantity={cart.setQuantity}
              />
            ))}
          </ul>

          <Card className="min-[900px]:sticky min-[900px]:top-ui-24">
            <CardHeader>
              <h3 className="m-0 text-base leading-tight">
                {merchBooth.translate('merchBooth.orderSummary')}
              </h3>
            </CardHeader>
            <CardContent className="grid gap-ui-16">
              <div className="flex items-center justify-between gap-ui-12 border-t border-border pt-ui-16">
                <span className="text-sm font-medium">
                  {merchBooth.translate('merchBooth.total')}
                </span>
                <strong>{formatMoney(total)}</strong>
              </div>
              <Button
                type="button"
                disabled={cashCheckoutMutation.isPending}
                onClick={() => {
                  setCheckoutError(false)
                  cashCheckoutMutation.mutate()
                }}
              >
                {merchBooth.translate('merchBooth.cashCheckout')}
              </Button>
            </CardContent>
          </Card>
        </div>
      )}
    </section>
  )
}

type ResolvedCartLine = {
  item: BoothItem
  quantity: number
}

function CartHeader(props: { translate: Translate }) {
  return (
    <header className="grid gap-ui-16">
      <Button variant="ghost" className="w-fit px-0 hover:bg-transparent" asChild>
        <Link to="/merch-booth">
          <ArrowLeft aria-hidden="true" />
          {props.translate('merchBooth.backToBooth')}
        </Link>
      </Button>
      <h2 className="m-0 text-[1.75rem] leading-[1.15]">
        {props.translate('merchBooth.cartTitle')}
      </h2>
    </header>
  )
}

function CartLineRow(props: {
  line: ResolvedCartLine
  translate: Translate
  onSetQuantity: (variantId: string, quantity: number) => void
}) {
  const label = boothItemLabel(props.line.item, props.translate)

  return (
    <li className="grid grid-cols-[72px_minmax(0,1fr)] gap-ui-12 border-b border-border pb-ui-16">
      <img
        src={props.line.item.photo.display.publicUrl}
        alt={`${props.line.item.productName} ${props.translate('merchBooth.photoAlt')}`}
        className="aspect-square size-[72px] rounded-md bg-muted object-cover"
      />
      <div className="grid min-w-0 gap-ui-12">
        <div className="flex items-start justify-between gap-ui-12">
          <div className="grid min-w-0 gap-ui-2">
            <strong className="overflow-hidden text-ellipsis whitespace-nowrap text-sm">
              {props.line.item.productName}
            </strong>
            <span className="text-xs text-white-300">{label}</span>
          </div>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label={`${props.translate('merchBooth.removeFromCart')} ${label}`}
            onClick={() => props.onSetQuantity(props.line.item.variantId, 0)}
          >
            <Trash2 aria-hidden="true" />
          </Button>
        </div>
        <div className="flex flex-wrap items-center justify-between gap-ui-12">
          <div className="flex items-center gap-ui-8">
            <Button
              type="button"
              variant="outline"
              size="icon"
              aria-label={`${props.translate('merchBooth.decreaseQuantity')} ${label}`}
              disabled={props.line.quantity <= 1}
              onClick={() => props.onSetQuantity(props.line.item.variantId, props.line.quantity - 1)}
            >
              <Minus aria-hidden="true" />
            </Button>
            <span className="w-ui-24 text-center text-sm tabular-nums">{props.line.quantity}</span>
            <Button
              type="button"
              variant="outline"
              size="icon"
              aria-label={`${props.translate('merchBooth.increaseQuantity')} ${label}`}
              disabled={props.line.quantity >= props.line.item.quantity}
              onClick={() => props.onSetQuantity(props.line.item.variantId, props.line.quantity + 1)}
            >
              <Plus aria-hidden="true" />
            </Button>
          </div>
          <strong className="text-sm">
            {formatMoney(props.line.item.price.amount * props.line.quantity)}
          </strong>
        </div>
      </div>
    </li>
  )
}

function resolveCartLines(lines: CartLine[], items: BoothItem[]): ResolvedCartLine[] {
  return lines.flatMap((line) => {
    const item = items.find((candidate) => candidate.variantId === line.variantId)
    return item === undefined ? [] : [{ item, quantity: line.quantity }]
  })
}

function totalCartAmount(lines: ResolvedCartLine[]): number {
  return lines.reduce((total, line) => total + line.item.price.amount * line.quantity, 0)
}
