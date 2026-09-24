import { createContext, useContext } from 'react'
import type { ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import type { TranslationKey } from 'i18n'

import { getCurrentAccount } from '../auth/api'
import type { CurrentAccountResponse } from '../auth/api'
import { listBoothItems } from './api'
import type { BoothItem } from './api'
import { MerchBoothCartProvider } from './cart'

export type Translate = (key: TranslationKey) => string

type MerchBoothContextValue = {
  accessToken: string
  account: CurrentAccountResponse
  items: BoothItem[]
  translate: Translate
}

type MerchBoothLayoutProps = {
  accessToken: string
  translate: Translate
  children: ReactNode
}

const MerchBoothContext = createContext<MerchBoothContextValue | null>(null)

export function MerchBoothLayout(props: MerchBoothLayoutProps) {
  const accountQuery = useQuery({
    queryKey: ['account', 'current', props.accessToken],
    queryFn: () => getCurrentAccount(props.accessToken)
  })
  const boothItemsQuery = useQuery({
    queryKey: ['merch-booth', 'items', props.accessToken],
    queryFn: () => listBoothItems(props.accessToken)
  })

  if (accountQuery.isLoading || boothItemsQuery.isLoading) {
    return <StatusPanel message={props.translate('merchBooth.loading')} />
  }

  if (
    accountQuery.isError ||
    boothItemsQuery.isError ||
    accountQuery.data === undefined ||
    boothItemsQuery.data === undefined
  ) {
    return <StatusPanel message={props.translate('merchBooth.error')} />
  }

  const account = accountQuery.data
  const items = boothItemsQuery.data
  const scope = { userId: account.user.id, bandId: account.activeBand.bandId }
  const value: MerchBoothContextValue = {
    accessToken: props.accessToken,
    account,
    items,
    translate: props.translate
  }

  return (
    <MerchBoothCartProvider key={`${scope.userId}:${scope.bandId}`} scope={scope} items={items}>
      <MerchBoothContext.Provider value={value}>{props.children}</MerchBoothContext.Provider>
    </MerchBoothCartProvider>
  )
}

export function useMerchBooth(): MerchBoothContextValue {
  const value = useContext(MerchBoothContext)
  if (value === null) {
    throw new Error('useMerchBooth must be used inside MerchBoothLayout')
  }

  return value
}

function StatusPanel(props: { message: string }) {
  return (
    <section>
      <p className="m-0 text-base text-white-300" role="status">
        {props.message}
      </p>
    </section>
  )
}
