import { createApp } from 'vue'
import './style.css'
import App from './App.vue'
import { router } from './router'
import { setTokenProvider, setUnauthorizedHandler } from './core/api/http'
import { currentToken, signOut } from './platform/auth/session'
import { setSessionToken } from './core/store/session'
import { registerGlobalStyles } from './platform/notify/toast'

setTokenProvider(currentToken)
setSessionToken(currentToken())
registerGlobalStyles()

// 401（登录过期/被顶下线）：清会话并回登录页，带上回跳地址（与小程序 markAuth(false) 同口径）
setUnauthorizedHandler(() => {
  signOut()
  setSessionToken(null)
  router.push({ name: 'login', query: { redirect: router.currentRoute.value.fullPath } })
})

createApp(App).use(router).mount('#app')
