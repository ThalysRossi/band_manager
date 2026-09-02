import { requiredEnv } from '../config/env'

type RequestBody = Record<string, unknown>

type ApiRequest = {
  accessToken: string
  path: string
  method: 'DELETE' | 'GET' | 'POST' | 'PUT'
  body: RequestBody | null
  idempotent: boolean
}

export class ApiError extends Error {
  readonly statusCode: number
  readonly code: string

  constructor(statusCode: number, code: string, message: string) {
    super(message)
    this.statusCode = statusCode
    this.code = code
  }
}

export async function apiRequest<TResponse>(request: ApiRequest): Promise<TResponse> {
  const headers = requestHeaders(request.accessToken, request.idempotent)
  const response = await fetch(`${apiBaseURL()}${request.path}`, {
    method: request.method,
    headers,
    body: request.body === null ? null : JSON.stringify(request.body)
  })

  return parseJSONResponse<TResponse>(response)
}

export function apiBaseURL(): string {
  return requiredEnv('VITE_API_BASE_URL')
}

async function parseJSONResponse<TResponse>(response: Response): Promise<TResponse> {
  if (response.status === 204) {
    return undefined as TResponse
  }

  const body = await response.json()
  if (!response.ok) {
    throw new ApiError(response.status, errorCode(body), errorMessage(body))
  }

  return body as TResponse
}

function errorCode(body: unknown): string {
  if (typeof body !== 'object' || body === null || !('code' in body)) {
    return 'api_request_failed'
  }
  return typeof body.code === 'string' ? body.code : 'api_request_failed'
}

function errorMessage(body: unknown): string {
  if (typeof body !== 'object' || body === null || !('message' in body)) {
    return 'API request failed'
  }

  const message = body.message
  if (typeof message !== 'string' || message.trim() === '') {
    return 'API request failed'
  }

  return message
}

function requestHeaders(accessToken: string, idempotent: boolean): HeadersInit {
  const headers: Record<string, string> = {
    Authorization: `Bearer ${accessToken}`,
    'Content-Type': 'application/json'
  }

  if (idempotent) {
    headers['Idempotency-Key'] = crypto.randomUUID()
  }

  return headers
}
