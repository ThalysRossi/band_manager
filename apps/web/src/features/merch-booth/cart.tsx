import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState
} from 'react'
import type { ReactNode } from 'react'

import type { BoothItem, CashCheckoutItem } from './api'

export type CartScope = {
  userId: string
  bandId: string
}

export type CartLine = {
  variantId: string
  quantity: number
}

type StoredCart = {
  version: 1
  lines: CartLine[]
}

type InitialCartState = {
  lines: CartLine[]
  persistenceFailed: boolean
  reconciled: boolean
}

export type MerchBoothCartContextValue = {
  lines: CartLine[]
  totalUnits: number
  persistenceFailed: boolean
  reconciled: boolean
  checkoutSucceeded: boolean
  addItem: (variantId: string) => void
  setQuantity: (variantId: string, quantity: number) => void
  clear: () => void
  completeCheckout: () => void
}

type MerchBoothCartProviderProps = {
  scope: CartScope
  items: BoothItem[]
  children: ReactNode
}

const storagePrefix = 'band_manager:merch-booth-cart:v1'
const MerchBoothCartContext = createContext<MerchBoothCartContextValue | null>(null)

export function MerchBoothCartProvider(props: MerchBoothCartProviderProps) {
  const [initialState] = useState<InitialCartState>(() => createInitialCartState(props.scope, props.items))
  const [lines, setLines] = useState<CartLine[]>(initialState.lines)
  const [persistenceFailed, setPersistenceFailed] = useState<boolean>(initialState.persistenceFailed)
  const [reconciled, setReconciled] = useState<boolean>(initialState.reconciled)
  const [checkoutSucceeded, setCheckoutSucceeded] = useState<boolean>(false)

  useEffect(() => {
    const result = reconcileCartLines(lines, props.items)
    if (sameCartLines(lines, result)) {
      return
    }

    setLines(result)
    setReconciled(true)
  }, [lines, props.items])

  useEffect(() => {
    try {
      persistCart(props.scope, lines)
    } catch {
      setPersistenceFailed(true)
    }
  }, [lines, props.scope.bandId, props.scope.userId])

  const addItem = useCallback(
    (variantId: string): void => {
      setCheckoutSucceeded(false)
      setLines((currentLines) => addCartItem(currentLines, props.items, variantId))
    },
    [props.items]
  )

  const setQuantity = useCallback(
    (variantId: string, quantity: number): void => {
      setCheckoutSucceeded(false)
      setLines((currentLines) => setCartQuantity(currentLines, props.items, variantId, quantity))
    },
    [props.items]
  )

  const clear = useCallback((): void => {
    try {
      clearStoredMerchBoothCart(props.scope)
    } catch {
      setPersistenceFailed(true)
    }
    setLines([])
    setReconciled(false)
  }, [props.scope.bandId, props.scope.userId])

  const completeCheckout = useCallback((): void => {
    try {
      clearStoredMerchBoothCart(props.scope)
    } catch {
      setPersistenceFailed(true)
    }
    setLines([])
    setReconciled(false)
    setCheckoutSucceeded(true)
  }, [props.scope.bandId, props.scope.userId])

  const value = useMemo<MerchBoothCartContextValue>(() => {
    return {
      lines,
      totalUnits: totalCartUnits(lines),
      persistenceFailed,
      reconciled,
      checkoutSucceeded,
      addItem,
      setQuantity,
      clear,
      completeCheckout
    }
  }, [addItem, checkoutSucceeded, clear, completeCheckout, lines, persistenceFailed, reconciled, setQuantity])

  return <MerchBoothCartContext.Provider value={value}>{props.children}</MerchBoothCartContext.Provider>
}

export function useMerchBoothCart(): MerchBoothCartContextValue {
  const value = useContext(MerchBoothCartContext)
  if (value === null) {
    throw new Error('useMerchBoothCart must be used inside MerchBoothCartProvider')
  }

  return value
}

export function clearStoredMerchBoothCart(scope: CartScope): void {
  window.sessionStorage.removeItem(cartStorageKey(scope))
}

export function toCashCheckoutItems(lines: CartLine[]): CashCheckoutItem[] {
  return lines.map((line) => ({ variantId: line.variantId, quantity: line.quantity }))
}

function createInitialCartState(scope: CartScope, items: BoothItem[]): InitialCartState {
  try {
    const storedLines = loadStoredCart(scope)
    const lines = reconcileCartLines(storedLines, items)
    return {
      lines,
      persistenceFailed: false,
      reconciled: !sameCartLines(storedLines, lines)
    }
  } catch {
    return {
      lines: [],
      persistenceFailed: true,
      reconciled: false
    }
  }
}

function loadStoredCart(scope: CartScope): CartLine[] {
  const value = window.sessionStorage.getItem(cartStorageKey(scope))
  if (value === null) {
    return []
  }

  const parsed: unknown = JSON.parse(value)
  if (!isStoredCart(parsed)) {
    throw new Error(`Invalid merch booth cart data for band ${scope.bandId}`)
  }

  return parsed.lines
}

function persistCart(scope: CartScope, lines: CartLine[]): void {
  const key = cartStorageKey(scope)
  if (lines.length === 0) {
    window.sessionStorage.removeItem(key)
    return
  }

  const cart: StoredCart = { version: 1, lines }
  window.sessionStorage.setItem(key, JSON.stringify(cart))
}

function cartStorageKey(scope: CartScope): string {
  return `${storagePrefix}:${scope.userId}:${scope.bandId}`
}

function isStoredCart(value: unknown): value is StoredCart {
  if (!isRecord(value) || value.version !== 1 || !Array.isArray(value.lines)) {
    return false
  }

  return value.lines.every(isCartLine)
}

function isCartLine(value: unknown): value is CartLine {
  return (
    isRecord(value) &&
    typeof value.variantId === 'string' &&
    value.variantId !== '' &&
    typeof value.quantity === 'number' &&
    Number.isInteger(value.quantity) &&
    value.quantity > 0
  )
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function reconcileCartLines(lines: CartLine[], items: BoothItem[]): CartLine[] {
  return lines.flatMap((line) => {
    const item = items.find((candidate) => candidate.variantId === line.variantId)
    if (item === undefined || item.quantity <= 0 || item.soldOut) {
      return []
    }

    return [{ ...line, quantity: Math.min(line.quantity, item.quantity) }]
  })
}

function addCartItem(lines: CartLine[], items: BoothItem[], variantId: string): CartLine[] {
  const item = items.find((candidate) => candidate.variantId === variantId)
  if (item === undefined || item.quantity <= 0 || item.soldOut) {
    return lines
  }

  const existingLine = lines.find((line) => line.variantId === variantId)
  if (existingLine === undefined) {
    return [...lines, { variantId, quantity: 1 }]
  }
  if (existingLine.quantity >= item.quantity) {
    return lines
  }

  return lines.map((line) => {
    return line.variantId === variantId ? { ...line, quantity: line.quantity + 1 } : line
  })
}

function setCartQuantity(
  lines: CartLine[],
  items: BoothItem[],
  variantId: string,
  quantity: number
): CartLine[] {
  if (quantity <= 0) {
    return lines.filter((line) => line.variantId !== variantId)
  }

  const item = items.find((candidate) => candidate.variantId === variantId)
  if (item === undefined || item.quantity <= 0 || item.soldOut) {
    return lines.filter((line) => line.variantId !== variantId)
  }

  return lines.map((line) => {
    return line.variantId === variantId
      ? { ...line, quantity: Math.min(quantity, item.quantity) }
      : line
  })
}

function totalCartUnits(lines: CartLine[]): number {
  return lines.reduce((total, line) => total + line.quantity, 0)
}

function sameCartLines(left: CartLine[], right: CartLine[]): boolean {
  return (
    left.length === right.length &&
    left.every((line, index) => {
      const candidate = right[index]
      return candidate?.variantId === line.variantId && candidate.quantity === line.quantity
    })
  )
}
