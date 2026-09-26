import { createApp } from 'vue'
import './style.css'
import App from './App.vue'
import { router } from './router'
import { setTokenProvider } from './core/api/http'
import { currentToken } from './platform/auth/session'
import { setSessionToken } from './core/store/session'
import { registerGlobalStyles } from './platform/notify/toast'

setTokenProvider(currentToken)
setSessionToken(currentToken())
registerGlobalStyles()

createApp(App).use(router).mount('#app')
