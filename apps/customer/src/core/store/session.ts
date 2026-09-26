/**
 * core/store — reactive session state. Storage access is delegated to the
 * platform layer (§102: core has no window/localStorage).
 */
import { reactive } from 'vue'

interface SessionState {
  token: string | null
  memberName: string
  memberNo: string
}

export const session = reactive<SessionState>({
  token: null,
  memberName: '',
  memberNo: '',
})

export function setSessionToken(token: string | null): void {
  session.token = token
}
