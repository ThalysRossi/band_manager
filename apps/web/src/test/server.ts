import { setupServer } from 'msw/node'

import { apiHandlers } from './apiMocks'

export const server = setupServer(...apiHandlers)
