/**
 * core/api — HTTP client. No window/document (§102); token injection is
 * delegated to a provider set by platform/auth at bootstrap.
 */
import type { Envelope } from '../models/models'

let tokenProvider: () => string | null = () => null

export function setTokenProvider(fn: () => string | null): void {
  tokenProvider = fn
}

export class ApiError extends Error {
  code: string
  status: number

  constructor(code: string, message: string, status: number) {
    super(message)
    this.code = code
    this.status = status
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  const token = tokenProvider()
  if (token) headers.Authorization = `Bearer ${token}`

  const res = await fetch(path, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  const payload = (await res.json().catch(() => ({}))) as Envelope<T>
  if (!res.ok) {
    throw new ApiError(payload.code ?? 'ERROR', payload.msg ?? `请求失败(${res.status})`, res.status)
  }
  return payload.data as T
}

export const http = {
  get: <T>(path: string) => request<T>('GET', path),
  post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
  put: <T>(path: string, body?: unknown) => request<T>('PUT', path, body),
}
