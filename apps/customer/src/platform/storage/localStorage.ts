/**
 * platform/storage — the ONLY place touching browser storage (§102).
 */
const TOKEN_KEY = 'anmo.customer.token'

export function loadToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function saveToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY)
}
