/**
 * platform/auth — session lifecycle glue between storage and the app.
 */
import { loadToken, saveToken, clearToken } from '../storage/localStorage'

export function currentToken(): string | null {
  return loadToken()
}

export function signIn(token: string): void {
  saveToken(token)
}

export function signOut(): void {
  clearToken()
}
