// core/api — HTTP client（与顾客端同口径：相对路径 + Bearer token + 统一包络）
// 成功包络 {data} 或分页 {data,total,page,per_page}；错误 {code,msg}。

let tokenProvider: () => string | null = () => null
let onUnauthorized: () => void = () => {}

export function setTokenProvider(fn: () => string | null): void {
  tokenProvider = fn
}

export function setUnauthorizedHandler(fn: () => void): void {
  onUnauthorized = fn
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

interface Envelope<T> {
  data?: T
  total?: number
  page?: number
  per_page?: number
  code?: string
  msg?: string
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
    if (res.status === 401) onUnauthorized()
    throw new ApiError(payload.code ?? 'ERROR', payload.msg ?? `请求失败(${res.status})`, res.status)
  }
  return payload.data as T
}

export interface PageResult<T> {
  data: T[]
  total: number
  page: number
  per_page: number
}

async function requestPage<T>(method: string, path: string, body?: unknown): Promise<PageResult<T>> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  const token = tokenProvider()
  if (token) headers.Authorization = `Bearer ${token}`

  const res = await fetch(path, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  const payload = (await res.json().catch(() => ({}))) as Envelope<T[]>
  if (!res.ok) {
    if (res.status === 401) onUnauthorized()
    throw new ApiError(payload.code ?? 'ERROR', payload.msg ?? `请求失败(${res.status})`, res.status)
  }
  return {
    data: payload.data ?? [],
    total: payload.total ?? 0,
    page: payload.page ?? 1,
    per_page: payload.per_page ?? payload.data?.length ?? 0,
  }
}

export const http = {
  get: <T>(path: string) => request<T>('GET', path),
  post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
  put: <T>(path: string, body?: unknown) => request<T>('PUT', path, body),
  delete: <T>(path: string) => request<T>('DELETE', path),
  getPage: <T>(path: string) => requestPage<T>('GET', path),
  /**
   * 导出下载（D30）：blob + objectURL 保存，返回响应头 X-Export-Rows 供 toast。
   * blob 下载浏览器不读 Content-Disposition，文件名由调用方自造（方案 v3）。
   */
  downloadBlob: async (path: string, body: unknown, filename: string): Promise<number> => {
    const headers: Record<string, string> = { 'Content-Type': 'application/json' }
    const token = tokenProvider()
    if (token) headers.Authorization = `Bearer ${token}`
    const res = await fetch(path, { method: 'POST', headers, body: JSON.stringify(body) })
    if (!res.ok) {
      if (res.status === 401) onUnauthorized()
      const payload = (await res.json().catch(() => ({}))) as Envelope<unknown>
      throw new ApiError(payload.code ?? 'ERROR', payload.msg ?? `请求失败(${res.status})`, res.status)
    }
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(url)
    return Number(res.headers.get('X-Export-Rows') ?? 0)
  },
}

/** 导出文件名：anmo-<域名>-YYYYMMDD.<ext>（方案 v3：前端自造，非 Content-Disposition） */
export function exportFilename(domain: string, format: 'xlsx' | 'txt'): string {
  const d = new Date()
  const day = `${d.getFullYear()}${String(d.getMonth() + 1).padStart(2, '0')}${String(d.getDate()).padStart(2, '0')}`
  return `anmo-${domain}-${day}.${format}`
}

// 每次写操作的幂等键（AGENTS.md D-幂等：请求级 UUID）
export function idemKey(): string {
  return typeof crypto !== 'undefined' && crypto.randomUUID
    ? crypto.randomUUID()
    : `web-${Date.now()}-${Math.random().toString(16).slice(2)}`
}
