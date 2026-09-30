/**
 * core/api — HTTP client. No window/document (§102); token injection is
 * delegated to a provider set by platform/auth at bootstrap.
 */
import type { Envelope } from '../models/models'

let tokenProvider: () => string | null = () => null
let unauthorizedHandler: (() => void) | null = null

export function setTokenProvider(fn: () => string | null): void {
  tokenProvider = fn
}

/** platform 层注册 401 全局处理（登录过期：清会话 + 跳登录页），与小程序 request.js 同口径。 */
export function setUnauthorizedHandler(fn: () => void): void {
  unauthorizedHandler = fn
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

// 请求超时：与小程序 request.js 同口径（15s）。没有超时的话，网关挂起/宕机
// 场景下 fetch 永不 settle，页面会永远停在骨架屏（R5 宕机演练实测复现）。
const REQUEST_TIMEOUT_MS = 25000

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  const token = tokenProvider()
  if (token) headers.Authorization = `Bearer ${token}`

  const abort = new AbortController()
  const timer = setTimeout(() => abort.abort(), REQUEST_TIMEOUT_MS)
  let res: Response
  try {
    res = await fetch(path, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      signal: abort.signal,
    })
  } catch (e) {
    // 网络不可达/超时都归一到 NETWORK：页面据此区分"加载失败"与"空数据"
    const msg = e instanceof DOMException && e.name === 'AbortError' ? '请求超时，请检查网络后重试' : '网络不可用，请检查网络后重试'
    throw new ApiError('NETWORK', msg, 0)
  } finally {
    clearTimeout(timer)
  }
  const payload = (await res.json().catch(() => ({}))) as Envelope<T>
  if (!res.ok) {
    if (res.status === 401) unauthorizedHandler?.()
    throw new ApiError(payload.code ?? 'ERROR', payload.msg ?? `请求失败(${res.status})`, res.status)
  }
  return payload.data as T
}

export const http = {
  get: <T>(path: string) => request<T>('GET', path),
  post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
  put: <T>(path: string, body?: unknown) => request<T>('PUT', path, body),
}
